package nacosx

import (
	"log"
	"strings"

	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/zeromicro/go-zero/core/conf"
)

// MustLoadRemote 从 Nacos 配置中心拉取 dataId 内容(YAML)并覆盖合并到 conf。
// 典型用法:本地 etc/*.yaml 只放 Nacos 地址等 bootstrap 信息,
// 业务配置(Nacos 上的 <service-name>.yaml)启动时远程拉取覆盖。
// dataId 为空或 Nacos 未启用时保持本地配置。
func MustLoadRemote(c Config, dataId string, out any) {
	if c.Empty() || dataId == "" {
		return
	}
	cc, err := NewConfigClient(c)
	if err != nil {
		log.Fatalf("[nacosx] create config client failed: %v", err)
	}
	content, err := cc.GetConfig(vo.ConfigParam{DataId: dataId, Group: c.Group})
	if err != nil {
		log.Fatalf("[nacosx] load remote config %q failed: %v", dataId, err)
	}
	if strings.TrimSpace(content) == "" {
		log.Printf("[nacosx] remote config %q is empty, keep local config", dataId)
		return
	}
	// 覆盖合并:远程存在的字段覆盖本地,其余保留本地值
	if err := conf.LoadFromYamlBytes([]byte(content), out); err != nil {
		log.Fatalf("[nacosx] parse remote config %q failed: %v", dataId, err)
	}
	log.Printf("[nacosx] loaded remote config %q (group=%s)", dataId, c.Group)
}
