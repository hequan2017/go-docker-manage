package logic

import (
	"context"
	"net"
	"net/http"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/hequan2017/go-docker-manage/app/node/rpc/internal/config"
	"github.com/hequan2017/go-docker-manage/app/node/rpc/internal/svc"
	"github.com/hequan2017/go-docker-manage/app/node/rpc/pb"
	"github.com/hequan2017/go-docker-manage/pkg/dockerx/dockertest"
)

func newTestSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()
	var c config.Config
	c.DB.Path = filepath.Join(t.TempDir(), "node.db")
	s := svc.NewServiceContext(c)
	// 先于 TempDir 清理关闭连接,否则 Windows 上删不掉被占用的 db 文件
	if sqlDB, err := s.DB.DB(); err == nil {
		t.Cleanup(func() { _ = sqlDB.Close() })
	}
	return s
}

func versionRoute() map[string]http.HandlerFunc {
	return map[string]http.HandlerFunc{
		"GET /version": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"Version":"27.5.1","ApiVersion":"1.47"}`))
		},
	}
}

// createAndCheck 注册节点、做一次健康检查,返回检查结果与检查后的节点状态
func createAndCheck(t *testing.T, dockerURL string) (*pb.HealthCheckReply, string) {
	t.Helper()
	l := NewNodeLogic(context.Background(), newTestSvc(t))
	created, err := l.CreateNode(&pb.CreateNodeReq{Name: "n1", DockerApiUrl: dockerURL})
	if err != nil {
		t.Fatalf("CreateNode: %v", err)
	}
	reply, err := l.HealthCheck(&pb.HealthCheckReq{Id: created.Id})
	if err != nil {
		t.Fatalf("HealthCheck: %v", err)
	}
	got, err := l.GetNode(&pb.GetNodeReq{Id: created.Id})
	if err != nil {
		t.Fatalf("GetNode: %v", err)
	}
	return reply, got.Node.Status
}

func TestHealthCheckHealthyTCP(t *testing.T) {
	reply, nodeStatus := createAndCheck(t, dockertest.NewServer(t, versionRoute()))
	if !reply.Ok || reply.DockerVersion != "27.5.1" {
		t.Fatalf("unexpected reply: %+v", reply)
	}
	if nodeStatus != "healthy" {
		t.Fatalf("node status = %q, want healthy", nodeStatus)
	}
}

func TestHealthCheckUnreachable(t *testing.T) {
	// 占一个端口再关掉,保证连接被拒绝
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	reply, nodeStatus := createAndCheck(t, "tcp://"+addr)
	if reply.Ok || reply.Message == "" {
		t.Fatalf("unexpected reply: %+v", reply)
	}
	if nodeStatus != "unhealthy" {
		t.Fatalf("node status = %q, want unhealthy", nodeStatus)
	}
}

// 部署环境常用 unix:///var/run/docker.sock;Windows 版 Docker SDK 不支持 unix 协议
func TestHealthCheckUnixSocket(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("docker client does not support unix:// on Windows")
	}
	sock := filepath.Join(t.TempDir(), "docker.sock")
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_ping" {
			w.Header().Set("API-Version", "1.47")
			_, _ = w.Write([]byte("OK"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"Version":"27.5.1","ApiVersion":"1.47"}`))
	})}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })

	reply, _ := createAndCheck(t, "unix://"+sock)
	if !reply.Ok {
		t.Fatalf("unexpected reply: %+v", reply)
	}
}
