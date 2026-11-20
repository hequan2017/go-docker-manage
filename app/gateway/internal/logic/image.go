package logic

import (
	"context"
	"time"

	"github.com/hequan2017/go-docker-manage/app/gateway/internal/svc"
	"github.com/hequan2017/go-docker-manage/app/gateway/internal/types"
	imagepb "github.com/hequan2017/go-docker-manage/app/image/rpc/pb"
	"github.com/zeromicro/go-zero/core/logx"
)

type ImageLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewImageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ImageLogic {
	return &ImageLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *ImageLogic) List(nodeId int64) ([]types.ImageItem, error) {
	resp, err := l.svcCtx.ImageRpc.ListImages(l.ctx, &imagepb.ListImagesReq{NodeId: nodeId})
	if err != nil {
		return nil, err
	}
	items := make([]types.ImageItem, 0, len(resp.Images))
	for _, img := range resp.Images {
		tags := img.RepoTags
		if len(tags) == 0 {
			tags = []string{"<none>:<none>"}
		}
		items = append(items, types.ImageItem{
			Id:       img.Id,
			RepoTags: tags,
			Size:     img.Size,
			SizeText: types.HumanSize(img.Size),
			Created:  time.Unix(img.Created, 0).Format("2006-01-02 15:04"),
		})
	}
	return items, nil
}

func (l *ImageLogic) Pull(nodeId int64, ref string) (*types.PullImageResp, error) {
	resp, err := l.svcCtx.ImageRpc.PullImage(l.ctx, &imagepb.PullImageReq{NodeId: nodeId, Ref: ref})
	if err != nil {
		return nil, err
	}
	return &types.PullImageResp{Status: resp.Status}, nil
}

func (l *ImageLogic) Remove(nodeId int64, ref string, force bool) error {
	_, err := l.svcCtx.ImageRpc.RemoveImage(l.ctx, &imagepb.RemoveImageReq{
		NodeId: nodeId, ImageRef: ref, Force: force,
	})
	return err
}
