package main

import (
	"fmt"
	"math/rand"
	"time"
)

// seed 预置一批镜像与容器,启动即可看到一个"正在运行的 Docker 主机"
func (s *store) seed() {
	now := time.Now()
	rng := rand.New(rand.NewSource(now.UnixNano()))

	type seedImg struct {
		tag     string
		sizeMB  int64
		command string
	}
	images := []seedImg{
		{"nginx:alpine", 43, `nginx -g 'daemon off;'`},
		{"redis:7-alpine", 41, `redis-server`},
		{"mysql:8.0", 586, `mysqld`},
		{"postgres:16-alpine", 268, `postgres`},
		{"busybox:latest", 4, `sh`},
		{"portainer/portainer-ce:2.21.4", 298, `/portainer`},
		{"ubuntu:24.04", 78, `/bin/bash`},
	}
	imgIDs := map[string]string{}
	for _, im := range images {
		id := "sha256:" + newID()
		imgIDs[im.tag] = id
		s.images = append(s.images, &mImage{
			ID:       id,
			RepoTags: []string{im.tag},
			Created:  now.Add(-time.Duration(24+rng.Intn(300)) * time.Hour).Unix(),
			Size:     im.sizeMB * 1024 * 1024,
		})
	}

	type seedCtr struct {
		name    string
		image   string
		state   string
		upAgo   time.Duration // 启动于多久前
		ports   []mPort
		cpuBase float64
		memMB   float64
	}
	seedCtrs := []seedCtr{
		{"web-nginx", "nginx:alpine", stateRunning, 2*time.Hour + 13*time.Minute, []mPort{
			{IP: "0.0.0.0", PrivatePort: 80, PublicPort: 8080, Type: "tcp"},
			{IP: "0.0.0.0", PrivatePort: 443, PublicPort: 8443, Type: "tcp"},
		}, 1.2, 32},
		{"cache-redis", "redis:7-alpine", stateRunning, 26*time.Hour, []mPort{
			{IP: "0.0.0.0", PrivatePort: 6379, PublicPort: 6379, Type: "tcp"},
		}, 0.6, 18},
		{"db-mysql", "mysql:8.0", stateRunning, 3*24*time.Hour+5*time.Hour, []mPort{
			{IP: "0.0.0.0", PrivatePort: 3306, PublicPort: 3306, Type: "tcp"},
		}, 4.8, 412},
		{"db-postgres", "postgres:16-alpine", stateExited, 40 * time.Minute, []mPort{
			{IP: "0.0.0.0", PrivatePort: 5432, PublicPort: 15432, Type: "tcp"},
		}, 0, 0},
		{"tools-busybox", "busybox:latest", stateCreated, 0, nil, 0, 0},
		{"ui-portainer", "portainer/portainer-ce:2.21.4", stateExited, 5 * 24 * time.Hour, []mPort{
			{IP: "0.0.0.0", PrivatePort: 9443, PublicPort: 9443, Type: "tcp"},
		}, 0, 0},
	}

	cmdOf := func(tag string) string {
		for _, im := range images {
			if im.tag == tag {
				return im.command
			}
		}
		return "sleep infinity"
	}

	for _, sc := range seedCtrs {
		c := &mContainer{
			ID:        newID(),
			Name:      sc.name,
			Image:     sc.image,
			ImageID:   imgIDs[sc.image],
			Command:   cmdOf(sc.image),
			Created:   now.Add(-sc.upAgo).Add(-time.Minute * 2),
			State:     sc.state,
			Ports:     sc.ports,
			Env:       []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"},
			Labels:    map[string]string{"source": "mock-dockerd"},
			cpuBase:   sc.cpuBase,
			memBase:   sc.memMB * 1024 * 1024,
		}
		switch sc.state {
		case stateRunning:
			c.StartedAt = now.Add(-sc.upAgo)
			c.Pid = 1000 + rng.Intn(8000)
		case stateExited:
			c.Finished = now.Add(-sc.upAgo)
			c.ExitCode = 0
		}
		// 预生成一些历史日志
		c.logLines = genHistoryLogs(c, sc.upAgo, rng)
		s.containers = append(s.containers, c)
	}

	s.persistLocked()
	fmt.Printf("[mock-dockerd] seeded %d images, %d containers\n", len(s.images), len(s.containers))
}
