package server

import (
	"context"

	"github.com/hequan2017/go-docker-manage/app/node/rpc/internal/logic"
	"github.com/hequan2017/go-docker-manage/app/node/rpc/internal/svc"
	"github.com/hequan2017/go-docker-manage/app/node/rpc/pb"
)

type NodeServer struct {
	pb.UnimplementedNodeServer
	svcCtx *svc.ServiceContext
}

func NewNodeServer(svcCtx *svc.ServiceContext) *NodeServer {
	return &NodeServer{svcCtx: svcCtx}
}

func (s *NodeServer) CreateNode(ctx context.Context, in *pb.CreateNodeReq) (*pb.CreateNodeReply, error) {
	return logic.NewNodeLogic(ctx, s.svcCtx).CreateNode(in)
}

func (s *NodeServer) ListNodes(ctx context.Context, in *pb.ListNodesReq) (*pb.ListNodesReply, error) {
	return logic.NewNodeLogic(ctx, s.svcCtx).ListNodes(in)
}

func (s *NodeServer) GetNode(ctx context.Context, in *pb.GetNodeReq) (*pb.GetNodeReply, error) {
	return logic.NewNodeLogic(ctx, s.svcCtx).GetNode(in)
}

func (s *NodeServer) DeleteNode(ctx context.Context, in *pb.DeleteNodeReq) (*pb.DeleteNodeReply, error) {
	return logic.NewNodeLogic(ctx, s.svcCtx).DeleteNode(in)
}

func (s *NodeServer) HealthCheck(ctx context.Context, in *pb.HealthCheckReq) (*pb.HealthCheckReply, error) {
	return logic.NewNodeLogic(ctx, s.svcCtx).HealthCheck(in)
}
