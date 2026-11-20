package main

import (
	"encoding/json"
	"log"
	"os"
	"time"
)

// persistSnapshot 持久化快照结构(只保存可恢复的部分,日志不落盘)
type persistSnapshot struct {
	Containers []ctrSnapshot `json:"containers"`
	Images     []*mImage     `json:"images"`
}

type ctrSnapshot struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Image     string    `json:"image"`
	ImageID   string    `json:"image_id"`
	Command   string    `json:"command"`
	Created   time.Time `json:"created"`
	State     string    `json:"state"`
	StartedAt time.Time `json:"started_at"`
	Finished  time.Time `json:"finished_at"`
	ExitCode  int       `json:"exit_code"`
	Ports     []mPort   `json:"ports"`
	Env       []string  `json:"env"`
	CPUBase   float64   `json:"cpu_base"`
	MemBase   float64   `json:"mem_base"`
}

// persistLocked 保存到 dataPath(调用方需持锁)
func (s *store) persistLocked() {
	if s.dataPath == "" {
		return
	}
	snap := persistSnapshot{Images: s.images}
	for _, c := range s.containers {
		snap.Containers = append(snap.Containers, ctrSnapshot{
			ID: c.ID, Name: c.Name, Image: c.Image, ImageID: c.ImageID,
			Command: c.Command, Created: c.Created, State: c.State,
			StartedAt: c.StartedAt, Finished: c.Finished,
			ExitCode: c.ExitCode, Ports: c.Ports, Env: c.Env,
			CPUBase: c.cpuBase, MemBase: c.memBase,
		})
	}
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		log.Printf("[mock-dockerd] persist marshal: %v", err)
		return
	}
	if err := os.WriteFile(s.dataPath, data, 0o644); err != nil {
		log.Printf("[mock-dockerd] persist write: %v", err)
	}
}

// restore 从 dataPath 恢复;文件不存在或损坏返回 false(调用方重新 seed)
func (s *store) restore() bool {
	if s.dataPath == "" {
		return false
	}
	data, err := os.ReadFile(s.dataPath)
	if err != nil {
		return false
	}
	var snap persistSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		log.Printf("[mock-dockerd] restore: %v, reseeding", err)
		return false
	}
	s.images = snap.Images
	for _, cs := range snap.Containers {
		s.containers = append(s.containers, &mContainer{
			ID: cs.ID, Name: cs.Name, Image: cs.Image, ImageID: cs.ImageID,
			Command: cs.Command, Created: cs.Created, State: cs.State,
			StartedAt: cs.StartedAt, Finished: cs.Finished,
			ExitCode: cs.ExitCode, Pid: 0, Ports: cs.Ports, Env: cs.Env,
			Labels: map[string]string{"source": "mock-dockerd"},
			cpuBase: cs.CPUBase, memBase: cs.MemBase,
		})
	}
	log.Printf("[mock-dockerd] restored %d containers, %d images from %s",
		len(s.containers), len(s.images), s.dataPath)
	return true
}
