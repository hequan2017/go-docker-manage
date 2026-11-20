package model

import "time"

// User 用户表(每个服务私有数据库,本服务使用 SQLite)
type User struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Username  string    `gorm:"uniqueIndex;size:64" json:"username"`
	Password  string    `gorm:"size:128" json:"-"` // bcrypt 哈希
	Nickname  string    `gorm:"size:64" json:"nickname"`
	CreatedAt time.Time `json:"created_at"`
}

func (User) TableName() string { return "users" }
