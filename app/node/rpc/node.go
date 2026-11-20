// node-rpc —— Docker 节点管理服务:注册/列表/删除/健康检查,SQLite 持久化
package main

import (
	"flag"
	"fmt"

	"github.com/hequan2017/go-docker-manage/app/node/rpc/internal/config"
	"github.com/hequan2017/go-docker-manage/app/node/rpc/internal/server"
	"github.com/hequan2017/go-docker-manage/app/node/rpc/internal/svc"
	"github.com/hequan2017/go-docker-manage/app/node/rpc/pb"
	"github.com/hequan2017/go-docker-manage/pkg/nacosx"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
)

var configFile = flag.String("f", "etc/node.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	c.Nacos.OverrideFromEnv()
	nacosx.MustLoadRemote(c.Nacos, c.Name, &c)

	ctx := svc.NewServiceContext(c)
	s := zrpc.MustNewServer(c.RpcServerConf, func(g *grpc.Server) {
		pb.RegisterNodeServer(g, server.NewNodeServer(ctx))
	})
	defer s.Stop()

	// Nacos 注册中心:上报 gRPC 实例(未启用时为 no-op,便于本地直连调试)
	deregister := nacosx.MustRegister(c.Nacos, c.Name, c.ListenOn)
	defer deregister()

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
