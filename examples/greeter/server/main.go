// greeter 服务端:最小的 go-zero zrpc 微服务
//
// 运行:cd examples/greeter/server && go run . -f etc/greeter.yaml
package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/hequan2017/go-docker-manage/examples/greeter/pb"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Config 对应 etc/greeter.yaml;内嵌 RpcServerConf 获得 Name/ListenOn/日志等通用配置
type Config struct {
	zrpc.RpcServerConf
}

// greeterServer 实现 proto 生成的 pb.GreeterServer 接口
type greeterServer struct {
	pb.UnimplementedGreeterServer
}

func (s *greeterServer) SayHello(ctx context.Context, in *pb.HelloReq) (*pb.HelloReply, error) {
	if in.Name == "" {
		// 用 gRPC status 返回错误码,客户端可以用 status.Code(err) 判断
		return nil, status.Error(codes.InvalidArgument, "name 不能为空")
	}
	return &pb.HelloReply{Message: "你好, " + in.Name}, nil
}

var configFile = flag.String("f", "etc/greeter.yaml", "the config file")

func main() {
	flag.Parse()

	var c Config
	conf.MustLoad(*configFile, &c)

	// 创建 gRPC 服务器并注册业务实现
	s := zrpc.MustNewServer(c.RpcServerConf, func(g *grpc.Server) {
		pb.RegisterGreeterServer(g, &greeterServer{})
	})
	defer s.Stop()

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
