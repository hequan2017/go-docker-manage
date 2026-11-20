package logic

import (
	"context"

	containerpb "github.com/hequan2017/go-docker-manage/app/container/rpc/pb"
	"github.com/hequan2017/go-docker-manage/app/gateway/internal/svc"
	"github.com/hequan2017/go-docker-manage/app/gateway/internal/types"
	imagepb "github.com/hequan2017/go-docker-manage/app/image/rpc/pb"
	nodepb "github.com/hequan2017/go-docker-manage/app/node/rpc/pb"
	"github.com/zeromicro/go-zero/core/logx"
)

// DashboardLogic 聚合各微服务数据出总览
type DashboardLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDashboardLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DashboardLogic {
	return &DashboardLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *DashboardLogic) Stats() (*types.DashboardResp, error) {
	resp, err := l.svcCtx.NodeRpc.ListNodes(l.ctx, &nodepb.ListNodesReq{})
	if err != nil {
		return nil, err
	}
	out := &types.DashboardResp{NodeTotal: len(resp.Nodes)}
	for _, n := range resp.Nodes {
		if n.Status == "healthy" {
			out.NodeHealthy++
		}
		brief := types.NodeBrief{Id: n.Id, Name: n.Name, Status: n.Status}

		if containers, err := l.svcCtx.ContainerRpc.ListContainers(l.ctx, &containerpb.ListContainersReq{NodeId: n.Id, All: true}); err == nil {
			brief.Containers = len(containers.Containers)
			for _, c := range containers.Containers {
				if c.State == "running" {
					brief.Running++
				}
			}
			out.ContainerTotal += brief.Containers
			out.ContainerRunning += brief.Running
		} else {
			brief.Error = "容器查询失败"
		}
		if images, err := l.svcCtx.ImageRpc.ListImages(l.ctx, &imagepb.ListImagesReq{NodeId: n.Id}); err == nil {
			brief.Images = len(images.Images)
			out.ImageTotal += brief.Images
		} else if brief.Error == "" {
			brief.Error = "镜像查询失败"
		}
		out.Nodes = append(out.Nodes, brief)
	}
	return out, nil
}
