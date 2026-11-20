// user-rpc —— 用户服务:注册/登录/用户信息,SQLite 持久化
package main

import (
	"flag"
	"fmt"

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
	c.Nacos.OverrideFromEnv()
	// Nacos 配置中心:远程 <Name>.yaml 覆盖本地配置(未启用时保持本地)
	nacosx.MustLoadRemote(c.Nacos, c.Name, &c)

	ctx := svc.NewServiceContext(c)
	s := zrpc.MustNewServer(c.RpcServerConf, func(g *grpc.Server) {
		pb.RegisterUserServer(g, server.NewUserServer(ctx))
	})
	defer s.Stop()

	// Nacos 注册中心:上报 gRPC 实例(未启用时为 no-op,便于本地直连调试)
	deregister := nacosx.MustRegister(c.Nacos, c.Name, c.ListenOn)
	defer deregister()

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
