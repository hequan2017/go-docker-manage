package nacosx

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

// LocalIP 返回本机第一个非 loopback 的 IPv4 地址(容器内即容器 IP)
func LocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}
	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() && ipNet.IP.To4() != nil {
			return ipNet.IP.String()
		}
	}
	return "127.0.0.1"
}

// ParseListenOn 从 ListenOn(如 0.0.0.0:9000 / :9000)提取端口
func ParseListenOn(listenOn string) (int, error) {
	_, portStr, err := net.SplitHostPort(strings.TrimSpace(listenOn))
	if err != nil {
		return 0, fmt.Errorf("invalid ListenOn %q: %w", listenOn, err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 {
		return 0, fmt.Errorf("invalid port in ListenOn %q", listenOn)
	}
	return port, nil
}

// RegisterService 将 gRPC 服务实例注册到 Nacos(临时实例,gRPC 连接保活)
func RegisterService(nc naming_client.INamingClient, c Config, serviceName, ip string, port int) error {
	ok, err := nc.RegisterInstance(vo.RegisterInstanceParam{
		Ip:          ip,
		Port:        uint64(port),
		ServiceName: serviceName,
		GroupName:   c.Group,
		Weight:      10,
		Enable:      true,
		Healthy:     true,
		Ephemeral:   true,
		Metadata:    map[string]string{"protocol": "grpc"},
	})
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("nacos register %q returned false", serviceName)
	}
	log.Printf("[nacosx] registered service %q group=%s addr=%s:%d", serviceName, c.Group, ip, port)
	return nil
}

// DeregisterService 从 Nacos 注销实例
func DeregisterService(nc naming_client.INamingClient, c Config, serviceName, ip string, port int) {
	_, _ = nc.DeregisterInstance(vo.DeregisterInstanceParam{
		Ip:          ip,
		Port:        uint64(port),
		ServiceName: serviceName,
		GroupName:   c.Group,
	})
}
