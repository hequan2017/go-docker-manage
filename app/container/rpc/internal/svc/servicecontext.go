package svc

import (
	"github.com/hequan2017/go-docker-manage/app/container/rpc/internal/config"
	nodepb "github.com/hequan2017/go-docker-manage/app/node/rpc/pb"
	"github.com/hequan2017/go-docker-manage/pkg/zrpcx"
)

type ServiceContext struct {
	Config  config.Config
	NodeCli nodepb.NodeClient // 查询节点 Docker API 地址
}

func NewServiceContext(c config.Config) *ServiceContext {
	nodeCli := zrpcx.NewClient(c.NodeRpc, c.Nacos)
	return &ServiceContext{Config: c, NodeCli: nodepb.NewNodeClient(nodeCli.Conn())}
}
