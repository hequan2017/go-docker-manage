// Package zrpcx 构造 zrpc 客户端:
// 启用 Nacos 时走 nacos:// 服务发现,否则 Endpoints 直连(本地调试友好)。
package zrpcx

import (
	"sync"

	"github.com/hequan2017/go-docker-manage/pkg/nacosx"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/zrpc"
)

// Upstream 上游服务配置
type Upstream struct {
	Target    string   `json:",optional"` // nacos://service-name
	Endpoints []string `json:",optional"` // 直连地址(未启用 Nacos 时使用)
	Timeout   int64    `json:",default=5000"`
}

// NewClient 按 Nacos 启用状态选择服务发现或直连。
// NonBlock:启动时不等待下游就绪(懒连接),避免服务启动顺序依赖。
// nacos resolver 只需注册一次,进程内重复调用安全。
func NewClient(u Upstream, nacos nacosx.Config) zrpc.Client {
	if !nacos.Empty() && u.Target != "" {
		initNacosResolver(nacos)
		return zrpc.MustNewClient(zrpc.RpcClientConf{Target: u.Target, Timeout: u.Timeout, NonBlock: true})
	}
	return zrpc.MustNewClient(zrpc.RpcClientConf{Endpoints: u.Endpoints, Timeout: u.Timeout, NonBlock: true})
}

// resolverOnce 保证进程内只注册一次 nacos:// resolver(并发调用 NewClient 也安全)
var resolverOnce sync.Once

func initNacosResolver(nacos nacosx.Config) {
	resolverOnce.Do(func() {
		nc, err := nacosx.NewNamingClient(nacos)
		logx.Must(err)
		nacosx.RegisterResolver(nc, nacos)
	})
}
