package logic

import (
	"context"
	"errors"

	"github.com/hequan2017/go-docker-manage/app/user/rpc/internal/model"
	"github.com/hequan2017/go-docker-manage/app/user/rpc/internal/svc"
	"github.com/hequan2017/go-docker-manage/app/user/rpc/pb"
	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type LoginLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

// errBadCredentials 用户不存在与密码错误返回同一错误,避免泄露用户名是否存在
var errBadCredentials = status.Error(codes.Unauthenticated, "用户名或密码错误")

func (l *LoginLogic) Login(in *pb.LoginReq) (*pb.LoginReply, error) {
	var u model.User
	err := l.svcCtx.DB.Where("username = ?", in.Username).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errBadCredentials
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "查询用户失败")
	}
	if bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(in.Password)) != nil {
		return nil, errBadCredentials
	}
	return &pb.LoginReply{UserId: u.ID, Username: u.Username, Nickname: u.Nickname}, nil
}
