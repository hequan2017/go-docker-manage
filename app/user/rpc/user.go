// user-rpc —— 用户服务:注册/登录/用户信息,SQLite 持久化
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/hequan2017/go-docker-manage/app/user/rpc/internal/config"
	"github.com/hequan2017/go-docker-manage/app/user/rpc/internal/server"
	"github.com/hequan2017/go-docker-manage/app/user/rpc/internal/svc"
	"github.com/hequan2017/go-docker-manage/app/user/rpc/pb"
	"github.com/hequan2017/go-docker-manage/pkg/nacosx"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
)

var configFile = flag.String("f", "etc/user.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	// Nacos 配置中心:远程 <Name>.yaml 覆盖本地配置(未启用时保持本地)
	nacosx.MustLoadRemote(c.Nacos, c.Name, &c)

	ctx := svc.NewServiceContext(c)
	s := zrpc.MustNewServer(c.RpcServerConf, func(g *grpc.Server) {
		pb.RegisterUserServer(g, server.NewUserServer(ctx))
	})
	defer s.Stop()

	// Nacos 注册中心:上报 gRPC 实例(未启用时跳过,便于本地直连调试)
	if !c.Nacos.Empty() {
		port, err := nacosx.ParseListenOn(c.ListenOn)
		if err != nil {
			log.Fatalf("parse ListenOn: %v", err)
		}
		nc, err := nacosx.NewNamingClient(c.Nacos)
		if err != nil {
			log.Fatalf("create nacos naming client: %v", err)
		}
		if err := nacosx.RegisterService(nc, c.Nacos, c.Name, nacosx.LocalIP(), port); err != nil {
			log.Fatalf("register to nacos: %v", err)
		}
		defer nacosx.DeregisterService(nc, c.Nacos, c.Name, nacosx.LocalIP(), port)
	}

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
