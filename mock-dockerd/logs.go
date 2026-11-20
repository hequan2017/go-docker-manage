package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// genHistoryLogs 为种子容器生成历史日志
func genHistoryLogs(c *mContainer, upAgo time.Duration, rng *rand.Rand) []logLine {
	if c == nil {
		return nil
	}
	if c.State != stateRunning {
		// 已退出的容器留下退出前的少量日志
		n := 3 + rng.Intn(4)
		lines := make([]logLine, 0, n)
		for i := 0; i < n; i++ {
			lines = append(lines, logLine{
				Time:   c.Finished.Add(-time.Duration(n-i) * time.Second * 5),
				stream: 1,
				text:   logTemplate(c.Image, i, rng),
			})
		}
		return lines
	}
	n := 20
	lines := make([]logLine, 0, n)
	start := time.Now().Add(-upAgo)
	for i := 0; i < n; i++ {
		lines = append(lines, logLine{
			Time:   start.Add(time.Duration(i) * (upAgo / time.Duration(n))),
			stream: byte(1 + rng.Intn(8)%2),
			text:   logTemplate(c.Image, i, rng),
		})
	}
	return lines
}

// logTemplate 按镜像类型生成一条仿真日志
func logTemplate(image string, seq int, rng *rand.Rand) string {
	ts := dockerLogTime(time.Now())
	switch {
	case strings.Contains(image, "nginx"):
		codes := []int{200, 200, 200, 304, 404, 500}
		return fmt.Sprintf(`192.168.112.1 - - [%s] "GET /api/items?page=%d HTTP/1.1" %d %d "-" "curl/8.10.1"`,
			ts, 1+rng.Intn(9), codes[rng.Intn(len(codes))], 512+rng.Intn(8192))
	case strings.Contains(image, "redis"):
		if seq%7 == 0 {
			return fmt.Sprintf("1:M %s * Background saving terminated with success", ts)
		}
		if seq%11 == 0 {
			return fmt.Sprintf("1:M %s * DB saved on disk", ts)
		}
		return fmt.Sprintf("1:M %s * %d changes in 60 seconds. Saving...", ts, 10+rng.Intn(500))
	case strings.Contains(image, "mysql"):
		if seq%9 == 0 {
			return fmt.Sprintf("%s [Note] [MY-011825] [Server] InnoDB: Buffer pool(s) load completed", ts)
		}
		return fmt.Sprintf("%s [Note] [MY-010931] [Server] %d connections", ts, 2+rng.Intn(64))
	case strings.Contains(image, "postgres"):
		return fmt.Sprintf("%s [%%d] LOG:  checkpoint starting: time", ts)
	case strings.Contains(image, "portainer"):
		return fmt.Sprintf("%s INFO (pid %d) [api] GET /api/endpoints/1/docker/containers/json 200", ts, 100+rng.Intn(900))
	default:
		return fmt.Sprintf("%s app[%d]: heartbeat ok, queue_size=%d", ts, seq, rng.Intn(100))
	}
}

// dockerLogTime nginx 风格时间
func dockerLogTime(t time.Time) string {
	return t.Format("02/Jan/2006:15:04:05 +0000")
}

// startLogPumper 定期为 running 容器追加日志
func (s *store) startLogPumper(stop <-chan struct{}) {
	go func() {
		rng := rand.New(rand.NewSource(time.Now().UnixNano()))
		ticker := time.NewTicker(4 * time.Second)
		defer ticker.Stop()
		seq := 0
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				seq++
				s.mu.RLock()
				running := make([]*mContainer, 0, 8)
				for _, c := range s.containers {
					if c.State == stateRunning {
						running = append(running, c)
					}
				}
				s.mu.RUnlock()
				for _, c := range running {
					c.mu.Lock()
					c.logLines = append(c.logLines, logLine{
						Time:   time.Now(),
						stream: 1,
						text:   logTemplate(c.Image, seq, rng),
					})
					if len(c.logLines) > 500 { // 环形上限
						c.logLines = c.logLines[len(c.logLines)-500:]
					}
					c.mu.Unlock()
				}
			}
		}
	}()
}
