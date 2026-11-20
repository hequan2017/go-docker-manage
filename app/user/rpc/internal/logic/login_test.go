package logic

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/hequan2017/go-docker-manage/app/user/rpc/internal/config"
	"github.com/hequan2017/go-docker-manage/app/user/rpc/internal/svc"
	"github.com/hequan2017/go-docker-manage/app/user/rpc/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func newTestSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()
	var c config.Config
	c.DB.Path = filepath.Join(t.TempDir(), "user.db")
	s := svc.NewServiceContext(c)
	// 先于 TempDir 清理关闭连接,否则 Windows 上删不掉被占用的 db 文件
	if sqlDB, err := s.DB.DB(); err == nil {
		t.Cleanup(func() { _ = sqlDB.Close() })
	}
	return s
}

func TestLoginFailuresAreUnauthenticated(t *testing.T) {
	ctx := context.Background()
	s := newTestSvc(t)
	if _, err := NewRegisterLogic(ctx, s).Register(&pb.RegisterReq{Username: "alice", Password: "secret123"}); err != nil {
		t.Fatalf("register: %v", err)
	}

	cases := map[string]*pb.LoginReq{
		"wrong password": {Username: "alice", Password: "bad-password"},
		"unknown user":   {Username: "bob", Password: "secret123"},
	}
	for name, req := range cases {
		_, err := NewLoginLogic(ctx, s).Login(req)
		if got := status.Code(err); got != codes.Unauthenticated {
			t.Errorf("%s: code = %v, want Unauthenticated", name, got)
		}
	}
}

func TestLoginSuccess(t *testing.T) {
	ctx := context.Background()
	s := newTestSvc(t)
	if _, err := NewRegisterLogic(ctx, s).Register(&pb.RegisterReq{Username: "alice", Password: "secret123"}); err != nil {
		t.Fatalf("register: %v", err)
	}
	reply, err := NewLoginLogic(ctx, s).Login(&pb.LoginReq{Username: "alice", Password: "secret123"})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if reply.Username != "alice" || reply.UserId == 0 {
		t.Fatalf("unexpected reply: %+v", reply)
	}
}
