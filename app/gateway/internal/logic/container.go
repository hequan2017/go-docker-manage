package logic

import (
	"context"
	"strconv"
	"strings"
	"time"

	containerpb "github.com/hequan2017/go-docker-manage/app/container/rpc/pb"
	"github.com/hequan2017/go-docker-manage/app/gateway/internal/svc"
	"github.com/hequan2017/go-docker-manage/app/gateway/internal/types"
	"github.com/hequan2017/go-docker-manage/pkg/errorx"
	"github.com/zeromicro/go-zero/core/logx"
)

type ContainerLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewContainerLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ContainerLogic {
	return &ContainerLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func containerItems(list []*containerpb.ContainerSummary) []types.ContainerItem {
	items := make([]types.ContainerItem, 0, len(list))
	for _, c := range list {
		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}
		item := types.ContainerItem{
			Id:        c.Id,
			Name:      name,
			Image:     c.Image,
			State:     c.State,
			Status:    c.Status,
			CreatedAt: time.Unix(c.Created, 0).Format("2006-01-02 15:04:05"),
			Command:   c.Command,
		}
		for _, p := range c.Ports {
			host := ""
			if p.PublicPort > 0 {
				host = p.Ip + ":" + strconv.Itoa(int(p.PublicPort))
			}
			item.Ports = append(item.Ports, types.PortItem{
				Host:      host,
				Container: strconv.Itoa(int(p.PrivatePort)),
				Protocol:  p.Type,
			})
		}
		items = append(items, item)
	}
	return items
}

func (l *ContainerLogic) List(nodeId int64, all bool) ([]types.ContainerItem, error) {
	resp, err := l.svcCtx.ContainerRpc.ListContainers(l.ctx, &containerpb.ListContainersReq{NodeId: nodeId, All: all})
	if err != nil {
		return nil, err
	}
	return containerItems(resp.Containers), nil
}

func (l *ContainerLogic) Action(nodeId int64, containerId, action string) (string, error) {
	switch action {
	case "start", "stop", "restart":
	default:
		return "", errorx.Newf(errorx.CodeInvalidParam, "不支持的动作 %q", action)
	}
	resp, err := l.svcCtx.ContainerRpc.ContainerAction(l.ctx, &containerpb.ContainerActionReq{
		NodeId: nodeId, ContainerId: containerId, Action: action,
	})
	if err != nil {
		return "", err
	}
	return resp.Message, nil
}

func (l *ContainerLogic) Create(in *types.CreateContainerReq) (*types.CreateContainerResp, error) {
	if in.Image == "" {
		return nil, errorx.New(errorx.CodeInvalidParam, "镜像名不能为空")
	}
	req := &containerpb.CreateContainerReq{
		NodeId: in.NodeId,
		Name:   in.Name,
		Image:  in.Image,
		Env:    in.Env,
	}
	for _, p := range in.Ports {
		proto := p.Protocol
		if proto == "" {
			proto = "tcp"
		}
		req.Ports = append(req.Ports, &containerpb.PortBinding{
			HostPort:      p.HostPort,
			ContainerPort: p.ContainerPort,
			Protocol:      proto,
		})
	}
	resp, err := l.svcCtx.ContainerRpc.CreateContainer(l.ctx, req)
	if err != nil {
		return nil, err
	}
	return &types.CreateContainerResp{ContainerId: resp.ContainerId}, nil
}

func (l *ContainerLogic) Remove(nodeId int64, containerId string, force bool) error {
	_, err := l.svcCtx.ContainerRpc.RemoveContainer(l.ctx, &containerpb.RemoveContainerReq{
		NodeId: nodeId, ContainerId: containerId, Force: force,
	})
	return err
}

func (l *ContainerLogic) Logs(nodeId int64, containerId string, tail int64, timestamps bool) (string, error) {
	resp, err := l.svcCtx.ContainerRpc.ContainerLogs(l.ctx, &containerpb.ContainerLogsReq{
		NodeId: nodeId, ContainerId: containerId, Tail: tail, Timestamps: timestamps,
	})
	if err != nil {
		return "", err
	}
	return resp.Logs, nil
}

func (l *ContainerLogic) Stats(nodeId int64, containerId string) (*types.ContainerStatsResp, error) {
	resp, err := l.svcCtx.ContainerRpc.ContainerStats(l.ctx, &containerpb.ContainerStatsReq{
		NodeId: nodeId, ContainerId: containerId,
	})
	if err != nil {
		return nil, err
	}
	return &types.ContainerStatsResp{
		CpuPercent: resp.CpuPercent,
		MemUsageMb: resp.MemUsageMb,
		MemLimitMb: resp.MemLimitMb,
		NetRxMb:    resp.NetRxMb,
		NetTxMb:    resp.NetTxMb,
		Pids:       resp.Pids,
	}, nil
}
