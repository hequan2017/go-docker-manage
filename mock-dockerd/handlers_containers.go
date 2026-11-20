package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ---------- GET /containers/json ----------

func handleContainerList(s *store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		all := r.URL.Query().Get("all") == "1"
		containers := s.listContainers(all)
		out := make([]map[string]any, 0, len(containers))
		for _, c := range containers {
			names := []string{"/" + c.Name}
			ports := make([]map[string]any, 0, len(c.Ports))
			for _, p := range c.Ports {
				m := map[string]any{"IP": p.IP, "PrivatePort": p.PrivatePort, "Type": p.Type}
				if p.PublicPort != 0 {
					m["PublicPort"] = p.PublicPort
				}
				ports = append(ports, m)
			}
			out = append(out, map[string]any{
				"Id":      c.ID,
				"Names":   names,
				"Image":   c.Image,
				"ImageID": c.ImageID,
				"Command": c.Command,
				"Created": c.Created.Unix(),
				"Ports":   ports,
				"Labels":  c.Labels,
				"State":   c.State,
				"Status":  c.statusText(),
				"HostConfig": map[string]any{
					"NetworkMode": "bridge",
				},
				"NetworkSettings": map[string]any{
					"Networks": map[string]any{
						"bridge": map[string]any{
							"Gateway":   "172.17.0.1",
							"IPAddress": containerIP(c),
						},
					},
				},
				"Mounts": []any{},
			})
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// containerIP 由 ID 前几位生成稳定的容器 IP
func containerIP(c *mContainer) string {
	var v int
	for i := 0; i < 4 && i < len(c.ID); i++ {
		v += int(c.ID[i])
	}
	return fmt.Sprintf("172.17.0.%d", 2+v%200)
}

// ---------- POST /containers/create?name=xxx ----------

type createBody struct {
	Image        string            `json:"Image"`
	Env          []string          `json:"Env"`
	Labels       map[string]string `json:"Labels"`
	ExposedPorts map[string]struct{} `json:"ExposedPorts"`
	HostConfig   struct {
		PortBindings map[string][]portBinding `json:"PortBindings"`
		NetworkMode  string                   `json:"NetworkMode"`
	} `json:"HostConfig"`
}

type portBinding struct {
	HostIp   string `json:"HostIp"`
	HostPort string `json:"HostPort"`
}

func handleContainerCreate(s *store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body createBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if body.Image == "" {
			writeError(w, http.StatusBadRequest, "no image specified")
			return
		}
		// 镜像必须已存在(与真实 daemon 行为一致,先 pull 再 create)
		img := s.findImage(body.Image)
		if img == nil {
			notFoundImage(w, body.Image)
			return
		}
		name := strings.TrimPrefix(r.URL.Query().Get("name"), "/")
		if name == "" {
			name = "auto-" + newID()[:12]
		}
		if existed := s.findContainer(name); existed != nil {
			writeError(w, http.StatusConflict,
				fmt.Sprintf("Conflict. The container name %q is already in use by %s", "/"+name, existed.ID))
			return
		}

		c := &mContainer{
			ID:       newID(),
			Name:     name,
			Image:    body.Image,
			ImageID:  img.ID,
			Command:  commandOfImage(body.Image),
			Created:  time.Now(),
			State:    stateCreated,
			Env:      append([]string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"}, body.Env...),
			Labels:   body.Labels,
			cpuBase:  0.3 + rand.Float64()*2,
			memBase:  (16 + rand.Float64()*128) * 1024 * 1024,
		}
		if c.Labels == nil {
			c.Labels = map[string]string{}
		}
		c.Labels["source"] = "mock-dockerd"
		// 端口绑定 → Ports 列表
		for portProto, bindings := range body.HostConfig.PortBindings {
			privatePort, proto := parsePortProto(portProto)
			for _, b := range bindings {
				public, _ := strconv.Atoi(b.HostPort)
				c.Ports = append(c.Ports, mPort{
					IP:          b.HostIp,
					PrivatePort: uint16(privatePort),
					PublicPort:  uint16(public),
					Type:        proto,
				})
			}
		}
		s.addContainer(c)

		writeJSON(w, http.StatusCreated, map[string]any{"Id": c.ID, "Warnings": []string{}})
	}
}

func parsePortProto(portProto string) (int, string) {
	parts := strings.SplitN(portProto, "/", 2)
	port, _ := strconv.Atoi(parts[0])
	proto := "tcp"
	if len(parts) == 2 && parts[1] != "" {
		proto = parts[1]
	}
	return port, proto
}

func commandOfImage(image string) string {
	base := image
	if i := strings.IndexAny(base, ":@"); i > 0 {
		base = base[:i]
	}
	switch {
	case strings.Contains(base, "nginx"):
		return `nginx -g 'daemon off;'`
	case strings.Contains(base, "redis"):
		return "redis-server"
	case strings.Contains(base, "mysql"):
		return "mysqld"
	case strings.Contains(base, "postgres"):
		return "postgres"
	case strings.Contains(base, "mongo"):
		return "mongod"
	default:
		return "sleep infinity"
	}
}

// ---------- GET /containers/{id}/json ----------

func handleContainerInspect(s *store, w http.ResponseWriter, r *http.Request, ref string) {
	c := s.findContainer(ref)
	if c == nil {
		notFoundContainer(w, ref)
		return
	}
	startedAt := dockerZeroTime()
	finishedAt := dockerZeroTime()
	if !c.StartedAt.IsZero() {
		startedAt = c.StartedAt.UTC().Format(time.RFC3339Nano)
	}
	if !c.Finished.IsZero() {
		finishedAt = c.Finished.UTC().Format(time.RFC3339Nano)
	}

	// 端口映射表(镜像暴露端口 + 已映射端口)
	portsMap := map[string]any{}
	for _, p := range c.Ports {
		key := fmt.Sprintf("%d/%s", p.PrivatePort, p.Type)
		portsMap[key] = []map[string]string{{
			"HostIp":   p.IP,
			"HostPort": strconv.Itoa(int(p.PublicPort)),
		}}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"Id":      c.ID,
		"Created": c.Created.UTC().Format(time.RFC3339Nano),
		"Path":    strings.Fields(c.Command)[0],
		"Args":    strings.Fields(c.Command)[1:],
		"State": map[string]any{
			"Status":     c.State,
			"Running":    c.State == stateRunning || c.State == statePaused,
			"Paused":     c.State == statePaused,
			"Restarting": false,
			"OOMKilled":  false,
			"Dead":       c.State == stateDead,
			"Pid":        c.Pid,
			"ExitCode":   c.ExitCode,
			"Error":      "",
			"StartedAt":  startedAt,
			"FinishedAt": finishedAt,
		},
		"Image":        c.ImageID,
		"Name":         "/" + c.Name,
		"RestartCount": 0,
		"Driver":       "overlay2",
		"Platform":     "linux",
		"HostConfig": map[string]any{
			"NetworkMode": "bridge",
		},
		"Config": map[string]any{
			"Hostname": c.ID[:12],
			"Env":      c.Env,
			"Image":    c.Image,
			"Labels":   c.Labels,
		},
		"NetworkSettings": map[string]any{
			"IPAddress": containerIP(c),
			"Ports":     portsMap,
			"Networks": map[string]any{
				"bridge": map[string]any{
					"Gateway":   "172.17.0.1",
					"IPAddress": containerIP(c),
					"MacAddress": "02:42:ac:11:00:02",
				},
			},
		},
		"Mounts": []any{},
	})
}

// ---------- POST /containers/{id}/start|stop|restart|kill|pause|unpause ----------

func handleContainerAction(s *store, w http.ResponseWriter, r *http.Request, ref, action string) {
	c := s.findContainer(ref)
	if c == nil {
		notFoundContainer(w, ref)
		return
	}
	noop, err := func() (bool, error) {
		c.mu.Lock()
		defer c.mu.Unlock()
		return c.applyAction(action)
	}()
	if err != nil {
		writeError(w, http.StatusConflict, fmt.Sprintf("cannot %s container %s: %v", action, c.Name, err))
		return
	}
	if noop {
		writeNoContent(w) // 304 语义,SDK 视为成功
		return
	}
	s.mu.Lock()
	s.persistLocked()
	s.mu.Unlock()
	writeNoContent(w)
}

// ---------- DELETE /containers/{id}?force=1 ----------

func handleContainerRemove(s *store, w http.ResponseWriter, r *http.Request, ref string) {
	c := s.findContainer(ref)
	if c == nil {
		notFoundContainer(w, ref)
		return
	}
	force := r.URL.Query().Get("force") == "1"
	if c.State == stateRunning && !force {
		writeError(w, http.StatusConflict,
			fmt.Sprintf("You cannot remove a running container %s. Stop the container before attempting removal or force remove", c.ID))
		return
	}
	if !s.removeContainer(c.ID) {
		notFoundContainer(w, ref)
		return
	}
	writeNoContent(w)
}

// ---------- GET /containers/{id}/logs ----------

// handleContainerLogs 返回 docker 多路复流格式:
// [streamType(1/2), 0, 0, 0, size(4字节大端)] + payload
func handleContainerLogs(s *store, w http.ResponseWriter, r *http.Request, ref string) {
	q := r.URL.Query()
	showStdout, showStderr := q.Get("stdout") == "1", q.Get("stderr") == "1"
	if !showStdout && !showStderr {
		writeError(w, http.StatusBadRequest, "Bad parameters: you must choose at least one stream")
		return
	}
	c := s.findContainer(ref)
	if c == nil {
		notFoundContainer(w, ref)
		return
	}
	tail := 0 // 0 = 全部
	if t := q.Get("tail"); t != "" && t != "all" {
		if n, err := strconv.Atoi(t); err == nil && n > 0 {
			tail = n
		}
	}
	timestamps := q.Get("timestamps") == "1"

	c.mu.Lock()
	lines := make([]logLine, len(c.logLines))
	copy(lines, c.logLines)
	c.mu.Unlock()

	var filtered []logLine
	for _, ln := range lines {
		if (ln.stream == 1 && showStdout) || (ln.stream == 2 && showStderr) {
			filtered = append(filtered, ln)
		}
	}
	if tail > 0 && len(filtered) > tail {
		filtered = filtered[len(filtered)-tail:]
	}

	contentType := "application/vnd.docker.multiplexed-stream"
	if len(filtered) > 0 {
		first := filtered[0].stream
		single := true
		for _, ln := range filtered {
			if ln.stream != first {
				single = false
				break
			}
		}
		if single && !(showStdout && showStderr) {
			contentType = "application/vnd.docker.raw-stream"
		}
	}
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)

	buf := make([]byte, 8)
	for _, ln := range filtered {
		payload := ln.text
		if timestamps {
			payload = ln.UTC().Format(time.RFC3339Nano) + " " + payload
		}
		buf[0] = ln.stream
		binary.BigEndian.PutUint32(buf[4:8], uint32(len(payload)))
		if _, err := w.Write(buf); err != nil {
			return
		}
		if _, err := w.Write([]byte(payload)); err != nil {
			return
		}
	}
}

// ---------- GET /containers/{id}/stats?stream=false ----------

func handleContainerStats(s *store, w http.ResponseWriter, r *http.Request, ref string) {
	c := s.findContainer(ref)
	if c == nil {
		notFoundContainer(w, ref)
		return
	}
	stream := r.URL.Query().Get("stream") == "1"
	now := time.Now()

	var body []map[string]any
	func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		body = append(body, c.statsSnapshot(now.Add(-time.Second), now))
		if stream {
			// 简化实现:额外输出一个采样点后结束(follow 场景由 SDK 读到 EOF)
			body = append(body, c.statsSnapshot(now, now.Add(time.Second)))
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)
	if stream {
		for _, b := range body {
			_ = enc.Encode(b)
		}
	} else {
		_ = enc.Encode(body[0])
	}
}

// statsSnapshot 生成一次 v1 StatsJSON 采样(cpu/mem 在基线上随机波动)
func (c *mContainer) statsSnapshot(pre, cur time.Time) map[string]any {
	rng := rand.New(rand.NewSource(cur.UnixNano() + int64(len(c.ID))))
	running := c.State == stateRunning || c.State == statePaused

	cpuNow := c.cpuBase * (0.7 + rng.Float64()*0.6) // 百分比
	memNow := c.memBase * (0.8 + rng.Float64()*0.4)
	var cpuPre float64
	if pre.After(c.Created) {
		cpuPre = c.cpuBase * (0.7 + rng.Float64()*0.6)
	}
	if !running {
		cpuNow, cpuPre, memNow = 0, 0, 0
	}
	hz := 100.0
	totalUsage := cpuNow / 100.0 * hz * float64(cur.UnixNano()) / 1e9 * 1e6
	preUsage := cpuPre / 100.0 * hz * float64(pre.UnixNano()) / 1e9 * 1e6
	systemUsage := float64(cur.UnixNano()) / 1e6 * 4 // 4 核累计
	preSystem := float64(pre.UnixNano()) / 1e6 * 4

	netRx := uint64(memNow / 10)
	netTx := uint64(memNow / 25)

	snap := map[string]any{
		"read":    cur.UTC().Format(time.RFC3339Nano),
		"preread": pre.UTC().Format(time.RFC3339Nano),
		"pids_stats": map[string]any{
			"current": map[bool]int{true: 3 + rng.Intn(40), false: 0}[running],
		},
		"blkio_stats": map[string]any{"io_service_bytes_recursive": nil},
		"num_procs":   0,
		"storage_stats": map[string]any{},
		"cpu_stats": map[string]any{
			"cpu_usage": map[string]any{
				"total_usage":       uint64(totalUsage),
				"usage_in_kernelmode": uint64(totalUsage * 0.15),
				"usage_in_usermode":  uint64(totalUsage * 0.85),
			},
			"system_cpu_usage": uint64(systemUsage),
			"online_cpus":      4,
			"throttling_data":  map[string]any{"periods": 0, "throttled_periods": 0, "throttled_time": 0},
		},
		"precpu_stats": map[string]any{
			"cpu_usage": map[string]any{
				"total_usage":        uint64(preUsage),
				"usage_in_kernelmode": uint64(preUsage * 0.15),
				"usage_in_usermode":   uint64(preUsage * 0.85),
			},
			"system_cpu_usage": uint64(preSystem),
			"online_cpus":      4,
			"throttling_data":  map[string]any{"periods": 0, "throttled_periods": 0, "throttled_time": 0},
		},
		"memory_stats": map[string]any{
			"usage":     uint64(memNow),
			"max_usage": uint64(memNow * 1.2),
			"limit":     8 * 1024 * 1024 * 1024,
			"stats":     map[string]any{"inactive_file": uint64(memNow / 3)},
		},
	}
	if running {
		snap["networks"] = map[string]any{
			"eth0": map[string]any{
				"rx_bytes": netRx, "tx_bytes": netTx,
				"rx_packets": netRx / 800, "tx_packets": netTx / 800,
				"rx_errors": 0, "tx_errors": 0, "rx_dropped": 0, "tx_dropped": 0,
			},
		}
	}
	return snap
}
