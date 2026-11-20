package server

import (
	"context"

	"github.com/hequan2017/go-docker-manage/app/user/rpc/internal/logic"
	"github.com/hequan2017/go-docker-manage/app/user/rpc/internal/svc"
	"github.com/hequan2017/go-docker-manage/app/user/rpc/pb"
)

type UserServer struct {
	pb.UnimplementedUserServer
	svcCtx *svc.ServiceContext
}

func NewUserServer(svcCtx *svc.ServiceContext) *UserServer {
	return &UserServer{svcCtx: svcCtx}
}

func (s *UserServer) Register(ctx context.Context, in *pb.RegisterReq) (*pb.RegisterReply, error) {
	return logic.NewRegisterLogic(ctx, s.svcCtx).Register(in)
}

func (s *UserServer) Login(ctx context.Context, in *pb.LoginReq) (*pb.LoginReply, error) {
	return logic.NewLoginLogic(ctx, s.svcCtx).Login(in)
}

func (s *UserServer) UserInfo(ctx context.Context, in *pb.UserInfoReq) (*pb.UserInfoReply, error) {
	return logic.NewUserInfoLogic(ctx, s.svcCtx).UserInfo(in)
}
