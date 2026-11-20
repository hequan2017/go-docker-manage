package logic

import (
	"context"

	"github.com/hequan2017/go-docker-manage/app/gateway/internal/svc"
	"github.com/hequan2017/go-docker-manage/app/gateway/internal/types"
	nodepb "github.com/hequan2017/go-docker-manage/app/node/rpc/pb"
	"github.com/zeromicro/go-zero/core/logx"
)

type NodeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewNodeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *NodeLogic {
	return &NodeLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func toNode(n *nodepb.NodeInfo) types.Node {
	return types.Node{
		Id:           n.Id,
		Name:         n.Name,
		Host:         n.Host,
		DockerApiUrl: n.DockerApiUrl,
		Status:       n.Status,
		Description:  n.Description,
		CreatedAt:    n.CreatedAt,
	}
}

func (l *NodeLogic) Create(in *types.CreateNodeReq) (*types.Node, error) {
	resp, err := l.svcCtx.NodeRpc.CreateNode(l.ctx, &nodepb.CreateNodeReq{
		Name: in.Name, Host: in.Host, DockerApiUrl: in.DockerApiUrl, Description: in.Description,
	})
	if err != nil {
		return nil, err
	}
	return &types.Node{Id: resp.Id, Name: in.Name, Host: in.Host, DockerApiUrl: in.DockerApiUrl, Status: "unknown", Description: in.Description}, nil
}

func (l *NodeLogic) List() ([]types.Node, error) {
	resp, err := l.svcCtx.NodeRpc.ListNodes(l.ctx, &nodepb.ListNodesReq{})
	if err != nil {
		return nil, err
	}
	nodes := make([]types.Node, 0, len(resp.Nodes))
	for _, n := range resp.Nodes {
		nodes = append(nodes, toNode(n))
	}
	return nodes, nil
}

func (l *NodeLogic) Health(nodeId int64) (*types.HealthResp, error) {
	resp, err := l.svcCtx.NodeRpc.HealthCheck(l.ctx, &nodepb.HealthCheckReq{Id: nodeId})
	if err != nil {
		return nil, err
	}
	return &types.HealthResp{Ok: resp.Ok, Message: resp.Message, DockerVersion: resp.DockerVersion}, nil
}

func (l *NodeLogic) Delete(nodeId int64) error {
	_, err := l.svcCtx.NodeRpc.DeleteNode(l.ctx, &nodepb.DeleteNodeReq{Id: nodeId})
	return err
}
