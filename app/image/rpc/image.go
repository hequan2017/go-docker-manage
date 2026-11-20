// image-rpc —— 镜像管理服务:列表/拉取/删除(无状态,实时查 Docker API)
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/hequan2017/go-docker-manage/app/image/rpc/internal/config"
	"github.com/hequan2017/go-docker-manage/app/image/rpc/internal/server"
	"github.com/hequan2017/go-docker-manage/app/image/rpc/internal/svc"
	"github.com/hequan2017/go-docker-manage/app/image/rpc/pb"
	"github.com/hequan2017/go-docker-manage/pkg/nacosx"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
)

var configFile = flag.String("f", "etc/image.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	c.Nacos.OverrideFromEnv()
	nacosx.MustLoadRemote(c.Nacos, c.Name, &c)

	ctx := svc.NewServiceContext(c)
	s := zrpc.MustNewServer(c.RpcServerConf, func(g *grpc.Server) {
		pb.RegisterImageServer(g, server.NewImageServer(ctx))
	})
	defer s.Stop()

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
