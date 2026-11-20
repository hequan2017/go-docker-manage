package logic

import (
	"context"

	"github.com/hequan2017/go-docker-manage/app/user/rpc/internal/model"
	"github.com/hequan2017/go-docker-manage/app/user/rpc/internal/svc"
	"github.com/hequan2017/go-docker-manage/app/user/rpc/pb"
	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type LoginLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *LoginLogic) Login(in *pb.LoginReq) (*pb.LoginReply, error) {
	var u model.User
	err := l.svcCtx.DB.Where("username = ?", in.Username).First(&u).Error
	if err != nil {
		return nil, status.Error(codes.NotFound, "用户名或密码错误")
	}
	if bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(in.Password)) != nil {
		return nil, status.Error(codes.NotFound, "用户名或密码错误")
	}
	return &pb.LoginReply{UserId: u.ID, Username: u.Username, Nickname: u.Nickname}, nil
}
