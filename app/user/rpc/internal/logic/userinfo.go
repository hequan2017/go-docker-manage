package logic

import (
	"context"

	"github.com/hequan2017/go-docker-manage/app/user/rpc/internal/model"
	"github.com/hequan2017/go-docker-manage/app/user/rpc/internal/svc"
	"github.com/hequan2017/go-docker-manage/app/user/rpc/pb"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UserInfoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUserInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UserInfoLogic {
	return &UserInfoLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *UserInfoLogic) UserInfo(in *pb.UserInfoReq) (*pb.UserInfoReply, error) {
	var u model.User
	if err := l.svcCtx.DB.First(&u, in.UserId).Error; err != nil {
		return nil, status.Errorf(codes.NotFound, "用户 %d 不存在", in.UserId)
	}
	return &pb.UserInfoReply{
		UserId:    u.ID,
		Username:  u.Username,
		Nickname:  u.Nickname,
		CreatedAt: u.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}
