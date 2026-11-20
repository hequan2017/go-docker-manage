// Package dockerx 封装 Docker Engine API 客户端的创建与复用。
// 地址格式与 docker CLI 一致:tcp://host:2375 或 unix:///var/run/docker.sock,
// 指向 mock-dockerd 或真实 daemon 均可,业务层无感知。
package dockerx

import (
	"sync"

	dc "github.com/docker/docker/client"
)

var (
	mu      sync.Mutex
	clients sync.Map // addr -> *dc.Client
)

// New 返回指向 addr 的 Docker API 客户端(按地址缓存复用连接)
func New(addr string) (*dc.Client, error) {
	if v, ok := clients.Load(addr); ok {
		return v.(*dc.Client), nil
	}
	mu.Lock()
	defer mu.Unlock()
	if v, ok := clients.Load(addr); ok {
		return v.(*dc.Client), nil
	}
	c, err := dc.NewClientWithOpts(
		dc.WithHost(addr),
		dc.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, err
	}
	clients.Store(addr, c)
	return c, nil
}
