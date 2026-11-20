package model

import "time"

// Node Docker 节点元数据
type Node struct {
	ID           int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Name         string    `gorm:"uniqueIndex;size:64" json:"name"`
	Host         string    `gorm:"size:128" json:"host"`
	DockerApiUrl string    `gorm:"size:256" json:"docker_api_url"` // tcp://host:2375 / unix:///var/run/docker.sock
	Status       string    `gorm:"size:32;default:unknown" json:"status"`
	Description  string    `gorm:"size:256" json:"description"`
	CreatedAt    time.Time `json:"created_at"`
}

func (Node) TableName() string { return "nodes" }
