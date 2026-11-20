package server

import (
	"context"

	"github.com/hequan2017/go-docker-manage/app/container/rpc/internal/logic"
	"github.com/hequan2017/go-docker-manage/app/container/rpc/internal/svc"
	"github.com/hequan2017/go-docker-manage/app/container/rpc/pb"
)

type ContainerServer struct {
	pb.UnimplementedContainerServer
	svcCtx *svc.ServiceContext
}

func NewContainerServer(svcCtx *svc.ServiceContext) *ContainerServer {
	return &ContainerServer{svcCtx: svcCtx}
}

func (s *ContainerServer) ListContainers(ctx context.Context, in *pb.ListContainersReq) (*pb.ListContainersReply, error) {
	return logic.NewContainerLogic(ctx, s.svcCtx).ListContainers(in)
}

func (s *ContainerServer) ContainerAction(ctx context.Context, in *pb.ContainerActionReq) (*pb.ContainerActionReply, error) {
	return logic.NewContainerLogic(ctx, s.svcCtx).ContainerAction(in)
}

func (s *ContainerServer) RemoveContainer(ctx context.Context, in *pb.RemoveContainerReq) (*pb.RemoveContainerReply, error) {
	return logic.NewContainerLogic(ctx, s.svcCtx).RemoveContainer(in)
}

func (s *ContainerServer) CreateContainer(ctx context.Context, in *pb.CreateContainerReq) (*pb.CreateContainerReply, error) {
	return logic.NewContainerLogic(ctx, s.svcCtx).CreateContainer(in)
}

func (s *ContainerServer) ContainerLogs(ctx context.Context, in *pb.ContainerLogsReq) (*pb.ContainerLogsReply, error) {
	return logic.NewContainerLogic(ctx, s.svcCtx).ContainerLogs(in)
}

func (s *ContainerServer) ContainerStats(ctx context.Context, in *pb.ContainerStatsReq) (*pb.ContainerStatsReply, error) {
	return logic.NewContainerLogic(ctx, s.svcCtx).ContainerStats(in)
}
