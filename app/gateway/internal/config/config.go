package config

import (
	"github.com/hequan2017/go-docker-manage/pkg/nacosx"
	"github.com/hequan2017/go-docker-manage/pkg/zrpcx"
	"github.com/zeromicro/go-zero/rest"
)

type Config struct {
	rest.RestConf
	Auth struct {
		AccessSecret string
		AccessExpire int64 `json:",default=86400"`
	}
	Nacos     nacosx.Config `json:",optional"`
	Upstreams struct {
		UserRpc      zrpcx.Upstream `json:",optional"`
		NodeRpc      zrpcx.Upstream `json:",optional"`
		ContainerRpc zrpcx.Upstream `json:",optional"`
		ImageRpc     zrpcx.Upstream `json:",optional"`
	} `json:",optional"`
}
