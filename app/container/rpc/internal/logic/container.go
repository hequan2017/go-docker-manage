// 容器管理逻辑:经 node-rpc 查节点 Docker API 地址,用官方 docker SDK 操作
// —— 指向 mock-dockerd 或真实 daemon 均可,业务层无感知
package logic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
	"github.com/hequan2017/go-docker-manage/app/container/rpc/internal/svc"
	"github.com/hequan2017/go-docker-manage/app/container/rpc/pb"
	"github.com/hequan2017/go-docker-manage/app/node/rpc/nodeclient"
	nodepb "github.com/hequan2017/go-docker-manage/app/node/rpc/pb"
	"github.com/hequan2017/go-docker-manage/pkg/dockerx"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ContainerLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewContainerLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ContainerLogic {
	return &ContainerLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

// docker 查询节点并返回其 Docker 客户端(错误已转为 gRPC status)
func (l *ContainerLogic) docker(nodeId int64) (*nodepb.NodeInfo, *client.Client, error) {
	return nodeclient.Docker(l.ctx, l.svcCtx.NodeCli, nodeId)
}

// shortID 截取容器 ID 前 12 位用于展示;入参也可能是容器名或短 ID,不足 12 位原样返回
func shortID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

func (l *ContainerLogic) ListContainers(in *pb.ListContainersReq) (*pb.ListContainersReply, error) {
	_, cli, err := l.docker(in.NodeId)
	if err != nil {
		return nil, err
	}
	list, err := cli.ContainerList(l.ctx, container.ListOptions{All: in.All})
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "查询容器列表失败: %v", err)
	}
	reply := &pb.ListContainersReply{}
	for _, c := range list {
		item := &pb.ContainerSummary{
			Id:      c.ID,
			Names:   c.Names,
			Image:   c.Image,
			ImageId: c.ImageID,
			Command: c.Command,
			Created: c.Created,
			State:   c.State,
			Status:  c.Status,
		}
		for _, p := range c.Ports {
			item.Ports = append(item.Ports, &pb.PortInfo{
				Ip:          p.IP,
				PrivatePort: uint32(p.PrivatePort),
				PublicPort:  uint32(p.PublicPort),
				Type:        p.Type,
			})
		}
		reply.Containers = append(reply.Containers, item)
	}
	return reply, nil
}

func (l *ContainerLogic) ContainerAction(in *pb.ContainerActionReq) (*pb.ContainerActionReply, error) {
	_, cli, err := l.docker(in.NodeId)
	if err != nil {
		return nil, err
	}
	var actionErr error
	switch in.Action {
	case "start":
		actionErr = cli.ContainerStart(l.ctx, in.ContainerId, container.StartOptions{})
	case "stop":
		actionErr = cli.ContainerStop(l.ctx, in.ContainerId, container.StopOptions{})
	case "restart":
		actionErr = cli.ContainerRestart(l.ctx, in.ContainerId, container.StopOptions{})
	default:
		return nil, status.Errorf(codes.InvalidArgument, "不支持的动作 %q(可选 start/stop/restart)", in.Action)
	}
	if actionErr != nil {
		return nil, dockerx.ToStatus(actionErr)
	}
	return &pb.ContainerActionReply{Ok: true, Message: "容器 " + shortID(in.ContainerId) + " " + in.Action + " 完成"}, nil
}

func (l *ContainerLogic) RemoveContainer(in *pb.RemoveContainerReq) (*pb.RemoveContainerReply, error) {
	_, cli, err := l.docker(in.NodeId)
	if err != nil {
		return nil, err
	}
	if err := cli.ContainerRemove(l.ctx, in.ContainerId, container.RemoveOptions{Force: in.Force}); err != nil {
		return nil, dockerx.ToStatus(err)
	}
	return &pb.RemoveContainerReply{}, nil
}

func (l *ContainerLogic) CreateContainer(in *pb.CreateContainerReq) (*pb.CreateContainerReply, error) {
	node, cli, err := l.docker(in.NodeId)
	if err != nil {
		return nil, err
	}

	cfg := &container.Config{
		Image: in.Image,
		Env:   in.Env,
	}
	hostCfg := &container.HostConfig{}
	if len(in.Ports) > 0 {
		exposed := nat.PortSet{}
		bindings := nat.PortMap{}
		for _, p := range in.Ports {
			proto := p.Protocol
			if proto == "" {
				proto = "tcp"
			}
			port := nat.Port(fmt.Sprintf("%s/%s", p.ContainerPort, proto))
			exposed[port] = struct{}{}
			bindings[port] = append(bindings[port], nat.PortBinding{HostIP: "0.0.0.0", HostPort: p.HostPort})
		}
		cfg.ExposedPorts = exposed
		hostCfg.PortBindings = bindings
	}

	resp, err := cli.ContainerCreate(l.ctx, cfg, hostCfg, &network.NetworkingConfig{}, nil, in.Name)
	if err != nil {
		return nil, dockerx.ToStatus(err)
	}
	l.Infof("container %s created on node %s (image=%s)", shortID(resp.ID), node.Name, in.Image)
	return &pb.CreateContainerReply{ContainerId: resp.ID}, nil
}

func (l *ContainerLogic) ContainerLogs(in *pb.ContainerLogsReq) (*pb.ContainerLogsReply, error) {
	_, cli, err := l.docker(in.NodeId)
	if err != nil {
		return nil, err
	}
	tail := "all"
	if in.Tail > 0 {
		tail = strconv.FormatInt(in.Tail, 10)
	}
	reader, err := cli.ContainerLogs(l.ctx, in.ContainerId, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       tail,
		Timestamps: in.Timestamps,
	})
	if err != nil {
		return nil, dockerx.ToStatus(err)
	}
	defer reader.Close()

	var buf bytes.Buffer
	if _, err := stdcopy.StdCopy(&buf, &buf, reader); err != nil && buf.Len() == 0 {
		return nil, status.Errorf(codes.Internal, "读取日志失败: %v", err)
	}
	return &pb.ContainerLogsReply{Logs: buf.String()}, nil
}

func (l *ContainerLogic) ContainerStats(in *pb.ContainerStatsReq) (*pb.ContainerStatsReply, error) {
	_, cli, err := l.docker(in.NodeId)
	if err != nil {
		return nil, err
	}
	resp, err := cli.ContainerStats(l.ctx, in.ContainerId, false)
	if err != nil {
		return nil, dockerx.ToStatus(err)
	}
	defer resp.Body.Close()
	var v container.StatsResponse
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return nil, status.Errorf(codes.Internal, "解析 stats 失败: %v", err)
	}

	const mb = 1024 * 1024
	reply := &pb.ContainerStatsReply{
		MemUsageMb: float64(v.MemoryStats.Usage) / mb,
		MemLimitMb: float64(v.MemoryStats.Limit) / mb,
		Pids:       int64(v.PidsStats.Current),
	}
	if eth0, ok := v.Networks["eth0"]; ok {
		reply.NetRxMb = float64(eth0.RxBytes) / mb
		reply.NetTxMb = float64(eth0.TxBytes) / mb
	}
	// CPU% = Δusage / Δsystem * online_cpus * 100
	cpuDelta := float64(v.CPUStats.CPUUsage.TotalUsage) - float64(v.PreCPUStats.CPUUsage.TotalUsage)
	sysDelta := float64(v.CPUStats.SystemUsage) - float64(v.PreCPUStats.SystemUsage)
	if sysDelta > 0 && cpuDelta >= 0 {
		cpus := v.CPUStats.OnlineCPUs
		if cpus == 0 {
			cpus = 1
		}
		reply.CpuPercent = cpuDelta / sysDelta * float64(cpus) * 100
	}
	return reply, nil
}
