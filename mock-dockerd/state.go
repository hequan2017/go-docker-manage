// mock-dockerd:模拟 Docker Engine API 的内存 daemon。
// 用官方 docker SDK 即可无缝访问,后续将节点地址换成真实 docker 即完成对接切换。
package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// 容器状态
const (
	stateCreated = "created"
	stateRunning = "running"
	statePaused  = "paused"
	stateExited  = "exited"
	stateDead    = "dead"
)

type mPort struct {
	IP          string `json:"IP"`
	PrivatePort uint16 `json:"PrivatePort"`
	PublicPort  uint16 `json:"PublicPort,omitempty"`
	Type        string `json:"Type"`
}

// mContainer 内存容器对象(字段与 Docker API 响应对齐)
type mContainer struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"` // 不带 /
	Image     string    `json:"image"`
	ImageID   string    `json:"image_id"`
	Command   string    `json:"command"`
	Created   time.Time `json:"created"`
	State     string    `json:"state"`
	StartedAt time.Time `json:"started_at"`
	Finished  time.Time `json:"finished_at"`
	ExitCode  int       `json:"exit_code"`
	Pid       int       `json:"pid"`
	Ports     []mPort   `json:"ports"`
	Env       []string  `json:"env"`
	Labels    map[string]string `json:"labels"`

	// stats 基线,让每个容器的资源占用有辨识度
	cpuBase float64 `json:"-"`
	memBase float64 `json:"-"` // 字节

	// 单调递增的 CPU 计数器(保证 pre/cur 差值为正,CPU% 可算)
	lastTotal  uint64    `json:"-"` // 累计 CPU 时间 ns
	lastSystem uint64    `json:"-"` // 累计系统 CPU 时间 ns
	lastSample time.Time `json:"-"`

	mu        sync.Mutex `json:"-"`
	logLines  []logLine  `json:"-"`
}

type logLine struct {
	time.Time
	stream byte // 1 stdout / 2 stderr
	text   string
}

type mImage struct {
	ID       string   `json:"id"` // sha256:...
	RepoTags []string `json:"repo_tags"`
	Created  int64    `json:"created"`
	Size     int64    `json:"size"`
}

type store struct {
	mu        sync.RWMutex
	containers []*mContainer
	images     []*mImage
	dataPath   string // 非空则持久化到 JSON
}

func newStore(dataPath string) *store {
	return &store{dataPath: dataPath}
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func newID() string { return randomHex(32) }

// ---------- 容器 ----------

func (s *store) addContainer(c *mContainer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.containers = append(s.containers, c)
	s.persistLocked()
}

// findContainer 按 完整ID / ID唯一前缀 / 名称 查找(与真实 docker 行为一致)
func (s *store) findContainer(ref string) *mContainer {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ref = strings.TrimPrefix(ref, "/")
	// 精确 ID
	for _, c := range s.containers {
		if c.ID == ref || c.Name == ref {
			return c
		}
	}
	// 唯一前缀
	var hit *mContainer
	count := 0
	for _, c := range s.containers {
		if strings.HasPrefix(c.ID, ref) {
			hit = c
			count++
		}
	}
	if count == 1 {
		return hit
	}
	return nil
}

func (s *store) removeContainer(ref string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, c := range s.containers {
		if c.ID == ref || c.Name == strings.TrimPrefix(ref, "/") {
			s.containers = append(s.containers[:i], s.containers[i+1:]...)
			s.persistLocked()
			return true
		}
	}
	return false
}

func (s *store) listContainers(all bool) []*mContainer {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*mContainer
	for _, c := range s.containers {
		if !all && c.State != stateRunning {
			continue
		}
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Created.Before(out[j].Created) })
	return out
}

func (s *store) counts() (total, running, paused, stopped int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	total = len(s.containers)
	for _, c := range s.containers {
		switch c.State {
		case stateRunning:
			running++
		case statePaused:
			paused++
		default:
			stopped++
		}
	}
	return
}

// ---------- 镜像 ----------

func (s *store) listImages() []*mImage {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*mImage, len(s.images))
	copy(out, s.images)
	return out
}

func (s *store) findImage(ref string) *mImage {
	s.mu.RLock()
	defer s.mu.RUnlock()
	// tag 或 sha256:ID 完整匹配
	for _, img := range s.images {
		if img.ID == ref {
			return img
		}
		for _, t := range img.RepoTags {
			if t == ref {
				return img
			}
		}
	}
	// ID 前缀(去掉 sha256: 后比较)
	bare := strings.TrimPrefix(ref, "sha256:")
	var hit *mImage
	count := 0
	for _, img := range s.images {
		id := strings.TrimPrefix(img.ID, "sha256:")
		if strings.HasPrefix(id, bare) {
			hit = img
			count++
		}
	}
	if count == 1 {
		return hit
	}
	return nil
}

func (s *store) upsertImage(img *mImage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, old := range s.images {
		for _, t := range old.RepoTags {
			for _, nt := range img.RepoTags {
				if t == nt { // 同 tag 覆盖
					s.images[i] = img
					s.persistLocked()
					return
				}
			}
		}
	}
	s.images = append(s.images, img)
	s.persistLocked()
}

func (s *store) removeImage(ref string) *mImage {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, img := range s.images {
		if img.ID == ref {
			s.images = append(s.images[:i], s.images[i+1:]...)
			s.persistLocked()
			return img
		}
		for _, t := range img.RepoTags {
			if t == ref {
				s.images = append(s.images[:i], s.images[i+1:]...)
				s.persistLocked()
				return img
			}
		}
	}
	return nil
}

// imageInUse 镜像是否被容器引用
func (s *store) imageInUse(img *mImage) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.containers {
		if c.ImageID == img.ID {
			return true
		}
		for _, t := range img.RepoTags {
			if c.Image == t {
				return true
			}
		}
	}
	return false
}

// ---------- 容器状态机 ----------

func dockerZeroTime() string { return "0001-01-01T00:00:00Z" }

// applyAction 应用容器动作,返回 http 状态语义:
// noop=true 表示已是目标状态(304);err=非法状态(409)
func (c *mContainer) applyAction(action string) (noop bool, err error) {
	now := time.Now()
	switch action {
	case "start":
		if c.State == stateRunning {
			return true, nil
		}
		if c.State != stateCreated && c.State != stateExited && c.State != stateDead {
			return false, fmt.Errorf("cannot start container in %s state", c.State)
		}
		c.State = stateRunning
		c.StartedAt = now
		c.Finished = time.Time{}
		c.ExitCode = 0
		c.Pid = 1000 + int(now.UnixNano()%9000)
	case "stop", "kill":
		if c.State == stateExited || c.State == stateCreated {
			return true, nil
		}
		if c.State != stateRunning && c.State != statePaused {
			return false, fmt.Errorf("cannot stop container in %s state", c.State)
		}
		c.State = stateExited
		c.Finished = now
		c.Pid = 0
		if action == "kill" {
			c.ExitCode = 137
		}
	case "restart":
		if c.State == stateCreated {
			return false, fmt.Errorf("cannot restart a created container")
		}
		c.State = stateRunning
		c.StartedAt = now
		c.Finished = time.Time{}
	case "pause":
		if c.State != stateRunning {
			return false, fmt.Errorf("cannot pause container in %s state", c.State)
		}
		c.State = statePaused
	case "unpause":
		if c.State != statePaused {
			return false, fmt.Errorf("cannot unpause container in %s state", c.State)
		}
		c.State = stateRunning
	default:
		return false, fmt.Errorf("unknown action %q", action)
	}
	return false, nil
}

// statusText 生成 containers/json 的 Status 字段
func (c *mContainer) statusText() string {
	switch c.State {
	case stateRunning:
		return "Up " + humanDuration(time.Since(c.StartedAt))
	case statePaused:
		return "Up " + humanDuration(time.Since(c.StartedAt)) + " (paused)"
	case stateExited:
		return fmt.Sprintf("Exited (%d) %s ago", c.ExitCode, humanDuration(time.Since(c.Finished)))
	case stateCreated:
		return "Created"
	default:
		return "Dead"
	}
}

// humanDuration 简易的人类可读时长(对齐 docker 风格)
func humanDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%d seconds", max(int(d.Seconds()), 1))
	}
	if d < time.Hour {
		return fmt.Sprintf("%d minutes", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%d hours", int(d.Hours()))
	}
	return fmt.Sprintf("%d days", int(d.Hours()/24))
}
