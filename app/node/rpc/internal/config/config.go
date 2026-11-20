package config

import (
	"github.com/hequan2017/go-docker-manage/pkg/nacosx"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	Nacos nacosx.Config `json:",optional"`
	DB    struct {
		Path string `json:",default=data/node.db"`
	} `json:",optional"`
}
