package svc

import (
	"github.com/hequan2017/go-docker-manage/app/gateway/internal/config"
	containerpb "github.com/hequan2017/go-docker-manage/app/container/rpc/pb"
	imagepb "github.com/hequan2017/go-docker-manage/app/image/rpc/pb"
	nodepb "github.com/hequan2017/go-docker-manage/app/node/rpc/pb"
	userpb "github.com/hequan2017/go-docker-manage/app/user/rpc/pb"
	"github.com/hequan2017/go-docker-manage/pkg/zrpcx"
)

// ServiceContext 持有全部下游微服务的 gRPC 客户端(经 Nacos 发现或直连)
type ServiceContext struct {
	Config       config.Config
	UserRpc      userpb.UserClient
	NodeRpc      nodepb.NodeClient
	ContainerRpc containerpb.ContainerClient
	ImageRpc     imagepb.ImageClient
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:       c,
		UserRpc:      userpb.NewUserClient(zrpcx.NewClient(c.Upstreams.UserRpc, c.Nacos).Conn()),
		NodeRpc:      nodepb.NewNodeClient(zrpcx.NewClient(c.Upstreams.NodeRpc, c.Nacos).Conn()),
		ContainerRpc: containerpb.NewContainerClient(zrpcx.NewClient(c.Upstreams.ContainerRpc, c.Nacos).Conn()),
		ImageRpc:     imagepb.NewImageClient(zrpcx.NewClient(c.Upstreams.ImageRpc, c.Nacos).Conn()),
	}
}
