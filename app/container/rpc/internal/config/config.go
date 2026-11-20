package config

import (
	"github.com/hequan2017/go-docker-manage/pkg/nacosx"
	"github.com/hequan2017/go-docker-manage/pkg/zrpcx"
	"github.com/zeromicro/go-zero/zrpc"
)

// Upstream 上游服务地址(复用公共定义)
type Upstream = zrpcx.Upstream

type Config struct {
	zrpc.RpcServerConf
	Nacos  nacosx.Config `json:",optional"`
	NodeRpc Upstream     `json:",optional"`
}
