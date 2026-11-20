package server

import (
	"context"

	"github.com/hequan2017/go-docker-manage/app/image/rpc/internal/logic"
	"github.com/hequan2017/go-docker-manage/app/image/rpc/internal/svc"
	"github.com/hequan2017/go-docker-manage/app/image/rpc/pb"
)

type ImageServer struct {
	pb.UnimplementedImageServer
	svcCtx *svc.ServiceContext
}

func NewImageServer(svcCtx *svc.ServiceContext) *ImageServer {
	return &ImageServer{svcCtx: svcCtx}
}

func (s *ImageServer) ListImages(ctx context.Context, in *pb.ListImagesReq) (*pb.ListImagesReply, error) {
	return logic.NewImageLogic(ctx, s.svcCtx).ListImages(in)
}

func (s *ImageServer) PullImage(ctx context.Context, in *pb.PullImageReq) (*pb.PullImageReply, error) {
	return logic.NewImageLogic(ctx, s.svcCtx).PullImage(in)
}

func (s *ImageServer) RemoveImage(ctx context.Context, in *pb.RemoveImageReq) (*pb.RemoveImageReply, error) {
	return logic.NewImageLogic(ctx, s.svcCtx).RemoveImage(in)
}
