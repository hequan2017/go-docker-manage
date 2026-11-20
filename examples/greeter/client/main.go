// greeter 客户端:另一个微服务(或 gateway)调用 greeter-rpc 的方式
//
// 运行:go run ./examples/greeter/client -name 张三
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/hequan2017/go-docker-manage/examples/greeter/pb"
	"github.com/zeromicro/go-zero/zrpc"
)

func main() {
	name := flag.String("name", "world", "name to greet")
	flag.Parse()

	// 直连模式:写死服务端地址。
	// 接入 Nacos 后改成 Target: "nacos://greeter-rpc"(见 app/container/rpc/internal/svc)
	cli := zrpc.MustNewClient(zrpc.RpcClientConf{
		Endpoints: []string{"127.0.0.1:19100"},
		Timeout:   3000,
	})

	// 用生成的客户端桩代码发起调用,用法和调用本地函数一样
	greeter := pb.NewGreeterClient(cli.Conn())
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resp, err := greeter.SayHello(ctx, &pb.HelloReq{Name: *name})
	if err != nil {
		log.Fatalf("call SayHello: %v", err)
	}
	fmt.Println(resp.Message)
}
