package svc

import (
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"github.com/hequan2017/go-docker-manage/app/user/rpc/internal/config"
	"github.com/hequan2017/go-docker-manage/app/user/rpc/internal/model"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type ServiceContext struct {
	Config config.Config
	DB     *gorm.DB
}

func NewServiceContext(c config.Config) *ServiceContext {
	if dir := filepath.Dir(c.DB.Path); dir != "" && dir != "." {
		_ = os.MkdirAll(dir, 0o755)
	}
	db, err := gorm.Open(sqlite.Open(c.DB.Path), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	logx.Must(err)
	if err := db.AutoMigrate(&model.User{}); err != nil {
		logx.Must(err)
	}
	return &ServiceContext{Config: c, DB: db}
}
