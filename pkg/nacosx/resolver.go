package nacosx

import (
	"fmt"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"google.golang.org/grpc/resolver"
)

// RegisterResolver 注册 gRPC resolver,使 zrpc 客户端可用
// Target: "nacos://service-name" 走 Nacos 服务发现。
// 进程内调用一次即可,必须在创建 zrpc 客户端之前调用。
func RegisterResolver(nc naming_client.INamingClient, c Config) {
	resolver.Register(&nacosResolverBuilder{naming: nc, group: c.Group})
}

type nacosResolverBuilder struct {
	naming naming_client.INamingClient
	group  string
}

func (b *nacosResolverBuilder) Scheme() string { return Scheme }

func (b *nacosResolverBuilder) Build(target resolver.Target, cc resolver.ClientConn, _ resolver.BuildOptions) (resolver.Resolver, error) {
	serviceName := endpointOf(target)
	if serviceName == "" {
		return nil, fmt.Errorf("nacosx: empty service name in target %q", target.String())
	}
	r := &nacosResolver{
		naming:      b.naming,
		group:       b.group,
		serviceName: serviceName,
		cc:          cc,
	}
	if err := r.start(); err != nil {
		return nil, err
	}
	return r, nil
}

// endpointOf 兼容不同 grpc-go 版本对 target 的解析:
// nacos://user-rpc(host 形式)与 nacos:///user-rpc(path 形式)都支持
func endpointOf(target resolver.Target) string {
	if e := target.Endpoint(); e != "" {
		return e
	}
	if h := target.URL.Host; h != "" {
		return h
	}
	ep := target.URL.Path
	for len(ep) > 0 && ep[0] == '/' {
		ep = ep[1:]
	}
	return ep
}

type nacosResolver struct {
	naming      naming_client.INamingClient
	group       string
	serviceName string
	cc          resolver.ClientConn
}

func (r *nacosResolver) start() error {
	// 先主动查一次,拿到初始地址列表
	instances, err := r.naming.SelectInstances(vo.SelectInstancesParam{
		ServiceName: r.serviceName,
		GroupName:   r.group,
		HealthyOnly: true,
	})
	if err == nil && len(instances) > 0 {
		r.update(toAddresses(instances))
	}
	// 订阅变更(实例上下线时回调)
	return r.naming.Subscribe(&vo.SubscribeParam{
		ServiceName: r.serviceName,
		GroupName:   r.group,
		SubscribeCallback: func(services []model.Instance, err error) {
			if err != nil {
				return
			}
			r.update(toAddresses(services))
		},
	})
}

func toAddresses(instances []model.Instance) []resolver.Address {
	addrs := make([]resolver.Address, 0, len(instances))
	for _, ins := range instances {
		addrs = append(addrs, resolver.Address{Addr: fmt.Sprintf("%s:%d", ins.Ip, ins.Port)})
	}
	return addrs
}

func (r *nacosResolver) update(addrs []resolver.Address) {
	_ = r.cc.UpdateState(resolver.State{Addresses: addrs})
}

func (r *nacosResolver) ResolveNow(resolver.ResolveNowOptions) {}

func (r *nacosResolver) Close() {
	_ = r.naming.Unsubscribe(&vo.SubscribeParam{
		ServiceName: r.serviceName,
		GroupName:   r.group,
	})
}
