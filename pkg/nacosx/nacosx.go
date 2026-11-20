// Package nacosx 提供 go-zero 与 Nacos 的集成适配:
//   - 服务注册/注销(服务端)
//   - gRPC 服务发现解析器 nacos://scheme(客户端)
//   - 配置中心拉取(启动期合并远程配置)
package nacosx

import (
	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

// Scheme gRPC target 的 scheme,nacos://service-name
const Scheme = "nacos"

// Config Nacos 连接配置,只作为本地 bootstrap 配置存在,
// 服务自身的业务配置从 Nacos 配置中心的 dataId 拉取。
type Config struct {
	Host        string `json:",optional"`
	Port        uint64 `json:",default=8848"`
	NamespaceId string `json:",optional"`
	Group       string `json:",default=DEFAULT_GROUP"`
	Username    string `json:",optional"`
	Password    string `json:",optional"`
}

// Empty 未配置 Nacos(Host 为空)时,注册/发现/配置中心功能全部停用
func (c Config) Empty() bool { return c.Host == "" }

func (c Config) serverConfigs() []constant.ServerConfig {
	return []constant.ServerConfig{{
		IpAddr:      c.Host,
		Port:        c.Port,
		ContextPath: "/nacos",
		Scheme:      "http",
	}}
}

func (c Config) clientConfig() *constant.ClientConfig {
	return &constant.ClientConfig{
		NamespaceId:      c.NamespaceId,
		TimeoutMs:        5000,
		NotLoadCacheAtStart: true,
		LogLevel:         "error",
		Username:         c.Username,
		Password:         c.Password,
	}
}

// NewNamingClient 创建服务发现客户端(gRPC 长连接,保持 9848 端口可达)
func NewNamingClient(c Config) (naming_client.INamingClient, error) {
	return clients.NewNamingClient(vo.NacosClientParam{
		ClientConfig:  c.clientConfig(),
		ServerConfigs: c.serverConfigs(),
	})
}

// NewConfigClient 创建配置中心客户端
func NewConfigClient(c Config) (config_client.IConfigClient, error) {
	return clients.NewConfigClient(vo.NacosClientParam{
		ClientConfig:  c.clientConfig(),
		ServerConfigs: c.serverConfigs(),
	})
}
