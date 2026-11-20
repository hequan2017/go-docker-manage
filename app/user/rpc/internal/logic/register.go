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

type RegisterLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterLogic {
	return &RegisterLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *RegisterLogic) Register(in *pb.RegisterReq) (*pb.RegisterReply, error) {
	if len(in.Username) < 3 || len(in.Username) > 64 {
		return nil, status.Error(codes.InvalidArgument, "用户名长度需在 3~64 之间")
	}
	if len(in.Password) < 6 {
		return nil, status.Error(codes.InvalidArgument, "密码至少 6 位")
	}

	var count int64
	if err := l.svcCtx.DB.Model(&model.User{}).Where("username = ?", in.Username).Count(&count).Error; err != nil {
		return nil, status.Error(codes.Internal, "查询用户失败")
	}
	if count > 0 {
		return nil, status.Errorf(codes.AlreadyExists, "用户名 %s 已存在", in.Username)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, status.Error(codes.Internal, "密码加密失败")
	}
	nickname := in.Nickname
	if nickname == "" {
		nickname = in.Username
	}
	u := model.User{Username: in.Username, Password: string(hash), Nickname: nickname}
	if err := l.svcCtx.DB.Create(&u).Error; err != nil {
		return nil, status.Error(codes.Internal, "创建用户失败")
	}
	l.Infof("user %s registered, id=%d", in.Username, u.ID)
	return &pb.RegisterReply{UserId: u.ID}, nil
}
