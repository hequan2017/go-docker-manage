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
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
	"github.com/hequan2017/go-docker-manage/app/container/rpc/internal/svc"
	"github.com/hequan2017/go-docker-manage/app/container/rpc/pb"
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

// nodeOf 查询节点信息
func (l *ContainerLogic) nodeOf(nodeId int64) (*nodepb.NodeInfo, error) {
	resp, err := l.svcCtx.NodeCli.GetNode(l.ctx, &nodepb.GetNodeReq{Id: nodeId})
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "节点不存在: %s", grpcMessage(err))
	}
	return resp.Node, nil
}

func grpcMessage(err error) string {
	if st, ok := status.FromError(err); ok {
		return st.Message()
	}
	return err.Error()
}

func (l *ContainerLogic) ListContainers(in *pb.ListContainersReq) (*pb.ListContainersReply, error) {
	resp, err := l.nodeOf(in.NodeId)
	if err != nil {
		return nil, err
	}
	cli, err := dockerx.New(resp.DockerApiUrl)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Docker API 地址无效: %v", err)
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
	node, err := l.nodeOf(in.NodeId)
	if err != nil {
		return nil, err
	}
	cli, err := dockerx.New(node.DockerApiUrl)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Docker API 地址无效: %v", err)
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
		return nil, wrapDockerErr(actionErr)
	}
	return &pb.ContainerActionReply{Ok: true, Message: "容器 " + in.ContainerId[:12] + " " + in.Action + " 完成"}, nil
}

func (l *ContainerLogic) RemoveContainer(in *pb.RemoveContainerReq) (*pb.RemoveContainerReply, error) {
	node, err := l.nodeOf(in.NodeId)
	if err != nil {
		return nil, err
	}
	cli, err := dockerx.New(node.DockerApiUrl)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Docker API 地址无效: %v", err)
	}
	if err := cli.ContainerRemove(l.ctx, in.ContainerId, container.RemoveOptions{Force: in.Force}); err != nil {
		return nil, wrapDockerErr(err)
	}
	return &pb.RemoveContainerReply{}, nil
}

func (l *ContainerLogic) CreateContainer(in *pb.CreateContainerReq) (*pb.CreateContainerReply, error) {
	node, err := l.nodeOf(in.NodeId)
	if err != nil {
		return nil, err
	}
	cli, err := dockerx.New(node.DockerApiUrl)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Docker API 地址无效: %v", err)
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
		return nil, wrapDockerErr(err)
	}
	l.Infof("container %s created on node %s (image=%s)", resp.ID[:12], node.Name, in.Image)
	return &pb.CreateContainerReply{ContainerId: resp.ID}, nil
}

func (l *ContainerLogic) ContainerLogs(in *pb.ContainerLogsReq) (*pb.ContainerLogsReply, error) {
	node, err := l.nodeOf(in.NodeId)
	if err != nil {
		return nil, err
	}
	cli, err := dockerx.New(node.DockerApiUrl)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Docker API 地址无效: %v", err)
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
		return nil, wrapDockerErr(err)
	}
	defer reader.Close()

	var buf bytes.Buffer
	if _, err := stdcopy.StdCopy(&buf, &buf, reader); err != nil && buf.Len() == 0 {
		return nil, status.Errorf(codes.Internal, "读取日志失败: %v", err)
	}
	return &pb.ContainerLogsReply{Logs: buf.String()}, nil
}

func (l *ContainerLogic) ContainerStats(in *pb.ContainerStatsReq) (*pb.ContainerStatsReply, error) {
	node, err := l.nodeOf(in.NodeId)
	if err != nil {
		return nil, err
	}
	cli, err := dockerx.New(node.DockerApiUrl)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Docker API 地址无效: %v", err)
	}
	resp, err := cli.ContainerStats(l.ctx, in.ContainerId, false)
	if err != nil {
		return nil, wrapDockerErr(err)
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

// wrapDockerErr 将 docker SDK 错误(404/409 等)转成 gRPC 状态码
func wrapDockerErr(err error) error {
	msg := err.Error()
	// docker SDK 错误已带 "Error response from daemon: ..." 前缀
	switch {
	case isDockerNotFound(err):
		return status.Error(codes.NotFound, msg)
	case isDockerConflict(err):
		return status.Error(codes.FailedPrecondition, msg)
	default:
		return status.Error(codes.Unavailable, msg)
	}
}

func isDockerNotFound(err error) bool {
	type notFounds interface{ NotFound() bool }
	if e, ok := err.(notFounds); ok {
		return e.NotFound()
	}
	return false
}

func isDockerConflict(err error) bool {
	type conflicts interface{ Conflict() bool }
	if e, ok := err.(conflicts); ok {
		return e.Conflict()
	}
	return false
}
