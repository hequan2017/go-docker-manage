package svc

import (
	"github.com/hequan2017/go-docker-manage/app/image/rpc/internal/config"
	nodepb "github.com/hequan2017/go-docker-manage/app/node/rpc/pb"
	"github.com/hequan2017/go-docker-manage/pkg/zrpcx"
)

type ServiceContext struct {
	Config  config.Config
	NodeCli nodepb.NodeClient
}

func NewServiceContext(c config.Config) *ServiceContext {
	nodeCli := zrpcx.NewClient(c.NodeRpc, c.Nacos)
	return &ServiceContext{Config: c, NodeCli: nodepb.NewNodeClient(nodeCli.Conn())}
}
