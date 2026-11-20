package logic

import (
	"context"

	containerpb "github.com/hequan2017/go-docker-manage/app/container/rpc/pb"
	"github.com/hequan2017/go-docker-manage/app/gateway/internal/svc"
	"github.com/hequan2017/go-docker-manage/app/gateway/internal/types"
	imagepb "github.com/hequan2017/go-docker-manage/app/image/rpc/pb"
	nodepb "github.com/hequan2017/go-docker-manage/app/node/rpc/pb"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/mr"
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

// Stats 并发查询每个节点的容器与镜像(单个节点失败只标记该节点,不影响整体)
func (l *DashboardLogic) Stats() (*types.DashboardResp, error) {
	resp, err := l.svcCtx.NodeRpc.ListNodes(l.ctx, &nodepb.ListNodesReq{})
	if err != nil {
		return nil, err
	}

	briefs := make([]types.NodeBrief, len(resp.Nodes))
	containerErr := make([]bool, len(resp.Nodes))
	imageErr := make([]bool, len(resp.Nodes))
	tasks := make([]func(), 0, 2*len(resp.Nodes))
	for i, n := range resp.Nodes {
		briefs[i] = types.NodeBrief{Id: n.Id, Name: n.Name, Status: n.Status}
		tasks = append(tasks,
			func() {
				containers, err := l.svcCtx.ContainerRpc.ListContainers(l.ctx, &containerpb.ListContainersReq{NodeId: n.Id, All: true})
				if err != nil {
					containerErr[i] = true
					return
				}
				briefs[i].Containers = len(containers.Containers)
				for _, c := range containers.Containers {
					if c.State == "running" {
						briefs[i].Running++
					}
				}
			},
			func() {
				images, err := l.svcCtx.ImageRpc.ListImages(l.ctx, &imagepb.ListImagesReq{NodeId: n.Id})
				if err != nil {
					imageErr[i] = true
					return
				}
				briefs[i].Images = len(images.Images)
			},
		)
	}
	mr.FinishVoid(tasks...)

	out := &types.DashboardResp{NodeTotal: len(resp.Nodes), Nodes: briefs}
	for i := range briefs {
		b := &briefs[i]
		if b.Status == "healthy" {
			out.NodeHealthy++
		}
		switch {
		case containerErr[i]:
			b.Error = "容器查询失败"
		case imageErr[i]:
			b.Error = "镜像查询失败"
		}
		out.ContainerTotal += b.Containers
		out.ContainerRunning += b.Running
		out.ImageTotal += b.Images
	}
	return out, nil
}
