// 节点管理逻辑:CRUD + Docker API 健康检查
package logic

import (
	"context"
	"fmt"
	"time"

	"github.com/hequan2017/go-docker-manage/app/node/rpc/internal/model"
	"github.com/hequan2017/go-docker-manage/app/node/rpc/internal/svc"
	"github.com/hequan2017/go-docker-manage/app/node/rpc/pb"
	"github.com/hequan2017/go-docker-manage/pkg/dockerx"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type NodeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewNodeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *NodeLogic {
	return &NodeLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func toPb(n *model.Node) *pb.NodeInfo {
	return &pb.NodeInfo{
		Id:           n.ID,
		Name:         n.Name,
		Host:         n.Host,
		DockerApiUrl: n.DockerApiUrl,
		Status:       n.Status,
		Description:  n.Description,
		CreatedAt:    n.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

func (l *NodeLogic) CreateNode(in *pb.CreateNodeReq) (*pb.CreateNodeReply, error) {
	if in.Name == "" || in.DockerApiUrl == "" {
		return nil, status.Error(codes.InvalidArgument, "节点名称与 Docker API 地址不能为空")
	}
	var count int64
	l.svcCtx.DB.Model(&model.Node{}).Where("name = ?", in.Name).Count(&count)
	if count > 0 {
		return nil, status.Errorf(codes.AlreadyExists, "节点 %s 已存在", in.Name)
	}
	n := model.Node{Name: in.Name, Host: in.Host, DockerApiUrl: in.DockerApiUrl, Description: in.Description}
	if err := l.svcCtx.DB.Create(&n).Error; err != nil {
		return nil, status.Error(codes.Internal, "保存节点失败")
	}
	l.Infof("node %s created, id=%d, dockerApi=%s", n.Name, n.ID, n.DockerApiUrl)
	return &pb.CreateNodeReply{Id: n.ID}, nil
}

func (l *NodeLogic) ListNodes(_ *pb.ListNodesReq) (*pb.ListNodesReply, error) {
	var nodes []model.Node
	if err := l.svcCtx.DB.Order("id").Find(&nodes).Error; err != nil {
		return nil, status.Error(codes.Internal, "查询节点失败")
	}
	reply := &pb.ListNodesReply{}
	for i := range nodes {
		reply.Nodes = append(reply.Nodes, toPb(&nodes[i]))
	}
	return reply, nil
}

func (l *NodeLogic) GetNode(in *pb.GetNodeReq) (*pb.GetNodeReply, error) {
	var n model.Node
	if err := l.svcCtx.DB.First(&n, in.Id).Error; err != nil {
		return nil, status.Errorf(codes.NotFound, "节点 %d 不存在", in.Id)
	}
	return &pb.GetNodeReply{Node: toPb(&n)}, nil
}

func (l *NodeLogic) DeleteNode(in *pb.DeleteNodeReq) (*pb.DeleteNodeReply, error) {
	res := l.svcCtx.DB.Delete(&model.Node{}, in.Id)
	if res.Error != nil || res.RowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound, "节点 %d 不存在", in.Id)
	}
	return &pb.DeleteNodeReply{}, nil
}

// healthCheckTimeout 单次健康检查(_ping + version)的总超时
const healthCheckTimeout = 3 * time.Second

// HealthCheck 用 Docker SDK 探测节点(Ping + ServerVersion),并回写状态。
// 走 SDK 而不是手拼 HTTP,tcp:// 与 unix:// 地址都支持。
func (l *NodeLogic) HealthCheck(in *pb.HealthCheckReq) (*pb.HealthCheckReply, error) {
	var n model.Node
	if err := l.svcCtx.DB.First(&n, in.Id).Error; err != nil {
		return nil, status.Errorf(codes.NotFound, "节点 %d 不存在", in.Id)
	}

	ctx, cancel := context.WithTimeout(l.ctx, healthCheckTimeout)
	defer cancel()

	newStatus, message, version := "unhealthy", "", ""
	cli, err := dockerx.New(n.DockerApiUrl)
	if err != nil {
		message = fmt.Sprintf("Docker API 地址无效: %v", err)
	} else if _, err := cli.Ping(ctx); err != nil {
		message = fmt.Sprintf("连接失败: %v", err)
	} else {
		newStatus, message = "healthy", "ok"
		if v, err := cli.ServerVersion(ctx); err == nil {
			version = v.Version
		}
	}
	if err := l.svcCtx.DB.Model(&n).Update("status", newStatus).Error; err != nil {
		l.Errorf("update node %d status: %v", n.ID, err)
	}
	return &pb.HealthCheckReply{Ok: newStatus == "healthy", Message: message, DockerVersion: version}, nil
}
