// container-rpc —— 容器管理服务:列表/启停/重启/删除/创建/日志/stats(无状态,实时查 Docker API)
package main

import (
	"flag"
	"fmt"

	"github.com/hequan2017/go-docker-manage/app/container/rpc/internal/config"
	"github.com/hequan2017/go-docker-manage/app/container/rpc/internal/server"
	"github.com/hequan2017/go-docker-manage/app/container/rpc/internal/svc"
	"github.com/hequan2017/go-docker-manage/app/container/rpc/pb"
	"github.com/hequan2017/go-docker-manage/pkg/nacosx"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
)

var configFile = flag.String("f", "etc/container.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	c.Nacos.OverrideFromEnv()
	nacosx.MustLoadRemote(c.Nacos, c.Name, &c)

	ctx := svc.NewServiceContext(c)
	s := zrpc.MustNewServer(c.RpcServerConf, func(g *grpc.Server) {
		pb.RegisterContainerServer(g, server.NewContainerServer(ctx))
	})
	defer s.Stop()

	// Nacos 注册中心:上报 gRPC 实例(未启用时为 no-op,便于本地直连调试)
	deregister := nacosx.MustRegister(c.Nacos, c.Name, c.ListenOn)
	defer deregister()

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
