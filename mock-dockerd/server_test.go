package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/errdefs"
	dc "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
)

// newTestServer 启动 mock-dockerd 并返回指向它的 docker SDK 客户端
func newTestServer(t *testing.T) (*dc.Client, *store) {
	t.Helper()
	s := newStore("")
	s.mu.Lock()
	s.seed()
	s.mu.Unlock()
	srv := httptest.NewServer(newRouter(s))
	t.Cleanup(srv.Close)

	addr := strings.TrimPrefix(srv.URL, "http://")
	cli, err := dc.NewClientWithOpts(
		dc.WithHost("tcp://"+addr),
		dc.WithAPIVersionNegotiation(),
	)
	if err != nil {
		t.Fatalf("create docker client: %v", err)
	}
	t.Cleanup(func() { _ = cli.Close() })
	return cli, s
}

func TestPingVersionInfo(t *testing.T) {
	cli, s := newTestServer(t)
	ctx := context.Background()

	ping, err := cli.Ping(ctx)
	if err != nil {
		t.Fatalf("ping: %v", err)
	}
	if ping.APIVersion == "" || ping.OSType != "linux" {
		t.Fatalf("unexpected ping: %+v", ping)
	}

	ver, err := cli.ServerVersion(ctx)
	if err != nil {
		t.Fatalf("version: %v", err)
	}
	if ver.APIVersion != mockAPIVersion || ver.Os != "linux" {
		t.Fatalf("unexpected version: %+v", ver)
	}

	info, err := cli.Info(ctx)
	if err != nil {
		t.Fatalf("info: %v", err)
	}
	total, running, _, _ := s.counts()
	if info.Containers != total || info.ContainersRunning != running {
		t.Fatalf("info counts mismatch: %+v", info)
	}
}

func TestContainerLifecycle(t *testing.T) {
	cli, s := newTestServer(t)
	ctx := context.Background()

	list, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(list) != 6 {
		t.Fatalf("expect 6 seeded containers, got %d", len(list))
	}
	runningOnly, err := cli.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		t.Fatalf("list running: %v", err)
	}
	if len(runningOnly) != 3 {
		t.Fatalf("expect 3 running containers, got %d", len(runningOnly))
	}
	// 选一个 running 容器做生命周期断言
	var first struct {
		ID    string
		Names []string
		Image string
		State string
	}
	for _, c := range list {
		if c.State == "running" {
			first.ID, first.Names, first.Image, first.State = c.ID, c.Names, c.Image, c.State
			break
		}
	}
	if first.ID == "" {
		t.Fatal("no running container found")
	}
	if first.Names[0] == "" || first.Image == "" || first.State == "" {
		t.Fatalf("container summary fields missing: %+v", first)
	}

	// inspect
	inspect, err := cli.ContainerInspect(ctx, first.ID)
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if !inspect.State.Running && first.State == "running" {
		t.Fatalf("inspect state mismatch: %+v", inspect.State)
	}

	// stop → exited
	if err := cli.ContainerStop(ctx, first.ID, container.StopOptions{}); err != nil {
		t.Fatalf("stop: %v", err)
	}
	inspect, err = cli.ContainerInspect(ctx, first.ID)
	if err != nil || inspect.State.Status != "exited" {
		t.Fatalf("after stop: %v %+v", err, inspect.State)
	}
	// start → running
	if err := cli.ContainerStart(ctx, first.ID, container.StartOptions{}); err != nil {
		t.Fatalf("start: %v", err)
	}
	inspect, err = cli.ContainerInspect(ctx, first.ID)
	if err != nil || inspect.State.Status != "running" {
		t.Fatalf("after start: %v %+v", err, inspect.State)
	}
	// restart
	if err := cli.ContainerRestart(ctx, first.ID, container.StopOptions{}); err != nil {
		t.Fatalf("restart: %v", err)
	}
	_ = s
}

func TestCreateAndRemoveContainer(t *testing.T) {
	cli, _ := newTestServer(t)
	ctx := context.Background()

	// 镜像不存在 → 404
	_, err := cli.ContainerCreate(ctx, &container.Config{Image: "not-exist:1"}, nil, nil, nil, "")
	if err == nil || !errdefs.IsNotFound(err) {
		t.Fatalf("expect not-found error, got %v", err)
	}

	// 先拉镜像再创建
	rc, err := cli.ImagePull(ctx, "hello-world:latest", image.PullOptions{})
	if err != nil {
		t.Fatalf("pull: %v", err)
	}
	_, _ = io.Copy(io.Discard, rc)
	_ = rc.Close()

	created, err := cli.ContainerCreate(ctx,
		&container.Config{
			Image:        "hello-world:latest",
			Env:          []string{"FOO=bar"},
			ExposedPorts: nat.PortSet{"80/tcp": {}},
		},
		&container.HostConfig{
			PortBindings: nat.PortMap{
				"80/tcp": []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "18080"}},
			},
		}, nil, nil, "hello-test")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.ID == "" {
		t.Fatal("empty container id")
	}

	// running 才能看日志:先启动
	if err := cli.ContainerStart(ctx, created.ID, container.StartOptions{}); err != nil {
		t.Fatalf("start created: %v", err)
	}

	// 删除 running 容器不 force → 409
	err = cli.ContainerRemove(ctx, created.ID, container.RemoveOptions{})
	if err == nil || !errdefs.IsConflict(err) {
		t.Fatalf("expect conflict on remove running, got %v", err)
	}
	// force 删除
	if err := cli.ContainerRemove(ctx, created.ID, container.RemoveOptions{Force: true}); err != nil {
		t.Fatalf("force remove: %v", err)
	}
}

func TestContainerLogsStdcopy(t *testing.T) {
	cli, _ := newTestServer(t)
	ctx := context.Background()

	list, err := cli.ContainerList(ctx, container.ListOptions{})
	if err != nil || len(list) == 0 {
		t.Fatalf("list running: %v n=%d", err, len(list))
	}
	id := list[0].ID

	logs, err := cli.ContainerLogs(ctx, id, container.LogsOptions{
		ShowStdout: true, ShowStderr: true, Tail: "10", Timestamps: true,
	})
	if err != nil {
		t.Fatalf("logs: %v", err)
	}
	defer logs.Close()
	var stdout, stderr bytes.Buffer
	if _, err := stdcopy.StdCopy(&stdout, &stderr, logs); err != nil {
		t.Fatalf("stdcopy: %v", err)
	}
	if stdout.Len() == 0 && stderr.Len() == 0 {
		t.Fatal("no logs captured")
	}
	t.Logf("captured logs:\n%s", stdout.String())
}

func TestContainerStats(t *testing.T) {
	cli, _ := newTestServer(t)
	ctx := context.Background()

	list, err := cli.ContainerList(ctx, container.ListOptions{})
	if err != nil || len(list) == 0 {
		t.Fatalf("list running: %v", err)
	}

	resp, err := cli.ContainerStats(ctx, list[0].ID, false)
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	defer resp.Body.Close()
	var v container.StatsResponse
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		t.Fatalf("decode stats: %v", err)
	}
	if v.MemoryStats.Limit == 0 || v.CPUStats.CPUUsage.TotalUsage == 0 {
		t.Fatalf("stats fields empty: %+v", v)
	}
	// 计算一次 CPU%(业务服务同样算法)
	cpuDelta := float64(v.CPUStats.CPUUsage.TotalUsage) - float64(v.PreCPUStats.CPUUsage.TotalUsage)
	sysDelta := float64(v.CPUStats.SystemUsage) - float64(v.PreCPUStats.SystemUsage)
	if sysDelta > 0 {
		t.Logf("cpu percent ≈ %.2f%%", cpuDelta/sysDelta*float64(v.CPUStats.OnlineCPUs)*100)
	}
}

func TestImageLifecycle(t *testing.T) {
	cli, _ := newTestServer(t)
	ctx := context.Background()

	images, err := cli.ImageList(ctx, image.ListOptions{})
	if err != nil {
		t.Fatalf("image list: %v", err)
	}
	if len(images) != 7 {
		t.Fatalf("expect 7 seeded images, got %d", len(images))
	}

	// pull 新镜像
	rc, err := cli.ImagePull(ctx, "traefik:v3", image.PullOptions{})
	if err != nil {
		t.Fatalf("pull: %v", err)
	}
	var lastStatus string
	dec := json.NewDecoder(rc)
	for {
		var msg struct {
			Status string `json:"status"`
		}
		if err := dec.Decode(&msg); err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("decode pull stream: %v", err)
		}
		if msg.Status != "" {
			lastStatus = msg.Status
		}
	}
	_ = rc.Close()
	if !strings.Contains(lastStatus, "Downloaded") {
		t.Fatalf("unexpected final pull status: %q", lastStatus)
	}

	// 再次 pull → up to date
	rc2, err := cli.ImagePull(ctx, "traefik:v3", image.PullOptions{})
	if err != nil {
		t.Fatalf("pull again: %v", err)
	}
	body, _ := io.ReadAll(rc2)
	_ = rc2.Close()
	if !strings.Contains(string(body), "up to date") {
		t.Fatalf("expect up-to-date message, got %s", body)
	}

	// 删除
	resp, err := cli.ImageRemove(ctx, "traefik:v3", image.RemoveOptions{Force: true})
	if err != nil {
		t.Fatalf("image remove: %v", err)
	}
	if len(resp) == 0 {
		t.Fatal("expect untagged/deleted entries")
	}
}
