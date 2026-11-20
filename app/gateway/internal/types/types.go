package types

import "fmt"

// Body 统一响应体
type Body struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data,omitempty"`
}

// ---------- auth / user ----------

type RegisterReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Nickname string `json:"nickname,optional"`
}

type LoginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type UserBrief struct {
	UserId   int64  `json:"userId"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
}

type LoginResp struct {
	Token     string    `json:"token"`
	ExpiresAt int64     `json:"expiresAt"`
	User      UserBrief `json:"user"`
}

type UserInfoResp struct {
	UserBrief
	CreatedAt string `json:"createdAt"`
}

// ---------- node ----------

type CreateNodeReq struct {
	Name         string `json:"name"`
	Host         string `json:"host,optional"`
	DockerApiUrl string `json:"dockerApiUrl"`
	Description  string `json:"description,optional"`
}

type Node struct {
	Id           int64  `json:"id"`
	Name         string `json:"name"`
	Host         string `json:"host"`
	DockerApiUrl string `json:"dockerApiUrl"`
	Status       string `json:"status"`
	Description  string `json:"description"`
	CreatedAt    string `json:"createdAt"`
}

type NodeIDPathReq struct {
	NodeId int64 `path:"nodeId"`
}

type HealthResp struct {
	Ok            bool   `json:"ok"`
	Message       string `json:"message"`
	DockerVersion string `json:"dockerVersion"`
}

// ---------- container ----------

type ContainerListReq struct {
	NodeId int64 `path:"nodeId"`
	All    bool  `form:"all,optional"`
}

type PortItem struct {
	Host      string `json:"host"`
	Container string `json:"container"`
	Protocol  string `json:"protocol"`
}

type ContainerItem struct {
	Id        string     `json:"id"`
	Name      string     `json:"name"`
	Image     string     `json:"image"`
	State     string     `json:"state"`
	Status    string     `json:"status"`
	CreatedAt string     `json:"createdAt"`
	Command   string     `json:"command"`
	Ports     []PortItem `json:"ports"`
}

type CreateContainerReq struct {
	NodeId int64         `path:"nodeId"`
	Name   string        `json:"name,optional"`
	Image  string        `json:"image"`
	Env    []string      `json:"env,optional"`
	Ports  []PortBinding `json:"ports,optional"`
}

type PortBinding struct {
	HostPort      string `json:"hostPort"`
	ContainerPort string `json:"containerPort"`
	Protocol      string `json:"protocol,optional"`
}

type CreateContainerResp struct {
	ContainerId string `json:"containerId"`
}

type ContainerActionReq struct {
	NodeId      int64  `path:"nodeId"`
	ContainerId string `path:"containerId"`
	Action      string `path:"action"` // start|stop|restart
}

type ContainerLogsReq struct {
	NodeId      int64  `path:"nodeId"`
	ContainerId string `path:"containerId"`
	Tail        int64  `form:"tail,optional"`
	Timestamps  bool   `form:"timestamps,optional"`
}

type ContainerStatsReq struct {
	NodeId      int64  `path:"nodeId"`
	ContainerId string `path:"containerId"`
}

type ContainerStatsResp struct {
	CpuPercent float64 `json:"cpuPercent"`
	MemUsageMb float64 `json:"memUsageMb"`
	MemLimitMb float64 `json:"memLimitMb"`
	NetRxMb    float64 `json:"netRxMb"`
	NetTxMb    float64 `json:"netTxMb"`
	Pids       int64   `json:"pids"`
}

type RemoveContainerReq struct {
	NodeId      int64  `path:"nodeId"`
	ContainerId string `path:"containerId"`
	Force       bool   `form:"force,optional"`
}

// ---------- image ----------

type ImageItem struct {
	Id       string   `json:"id"`
	RepoTags []string `json:"repoTags"`
	Size     int64    `json:"size"`
	SizeText string   `json:"sizeText"`
	Created  string   `json:"created"`
}

type PullImageReq struct {
	NodeId int64  `path:"nodeId"`
	Ref    string `json:"ref"`
}

type PullImageResp struct {
	Status string `json:"status"`
}

type RemoveImageReq struct {
	NodeId int64  `path:"nodeId"`
	Ref    string `form:"ref"`
	Force  bool   `form:"force,optional"`
}

// ---------- dashboard ----------

type NodeBrief struct {
	Id         int64  `json:"id"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	Containers int    `json:"containers"`
	Running    int    `json:"running"`
	Images     int    `json:"images"`
	Error      string `json:"error,omitempty"`
}

type DashboardResp struct {
	NodeTotal        int         `json:"nodeTotal"`
	NodeHealthy      int         `json:"nodeHealthy"`
	ContainerTotal   int         `json:"containerTotal"`
	ContainerRunning int         `json:"containerRunning"`
	ImageTotal       int         `json:"imageTotal"`
	Nodes            []NodeBrief `json:"nodes"`
}

// HumanSize 字节数转可读文本
func HumanSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}
