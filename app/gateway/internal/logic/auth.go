// 认证逻辑:注册/登录(签发 JWT)/当前用户
package logic

import (
	"context"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/hequan2017/go-docker-manage/app/gateway/internal/svc"
	"github.com/hequan2017/go-docker-manage/app/gateway/internal/types"
	userpb "github.com/hequan2017/go-docker-manage/app/user/rpc/pb"
	"github.com/hequan2017/go-docker-manage/pkg/errorx"
	"github.com/zeromicro/go-zero/core/logx"
)

type AuthLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAuthLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AuthLogic {
	return &AuthLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *AuthLogic) Register(in *types.RegisterReq) (*types.UserBrief, error) {
	resp, err := l.svcCtx.UserRpc.Register(l.ctx, &userpb.RegisterReq{
		Username: in.Username, Password: in.Password, Nickname: in.Nickname,
	})
	if err != nil {
		return nil, err
	}
	return &types.UserBrief{UserId: resp.UserId, Username: in.Username, Nickname: in.Nickname}, nil
}

func (l *AuthLogic) Login(in *types.LoginReq) (*types.LoginResp, error) {
	u, err := l.svcCtx.UserRpc.Login(l.ctx, &userpb.LoginReq{
		Username: in.Username, Password: in.Password,
	})
	if err != nil {
		return nil, err
	}
	secret := l.svcCtx.Config.Auth.AccessSecret
	expire := l.svcCtx.Config.Auth.AccessExpire
	now := time.Now().Unix()
	claims := jwt.MapClaims{
		"uid":      u.UserId,
		"username": u.Username,
		"nickname": u.Nickname,
		"iat":      now,
		"exp":      now + expire,
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		return nil, errorx.New(errorx.CodeInternal, "签发令牌失败")
	}
	return &types.LoginResp{
		Token:     token,
		ExpiresAt: now + expire,
		User:      types.UserBrief{UserId: u.UserId, Username: u.Username, Nickname: u.Nickname},
	}, nil
}

func (l *AuthLogic) UserInfo() (*types.UserInfoResp, error) {
	uid, err := uidFromCtx(l.ctx)
	if err != nil {
		return nil, err
	}
	u, err := l.svcCtx.UserRpc.UserInfo(l.ctx, &userpb.UserInfoReq{UserId: uid})
	if err != nil {
		return nil, err
	}
	return &types.UserInfoResp{
		UserBrief: types.UserBrief{UserId: u.UserId, Username: u.Username, Nickname: u.Nickname},
		CreatedAt: u.CreatedAt,
	}, nil
}

// uidFromCtx 从 JWT claims(由 rest 中间件写入 ctx)取用户 ID
func uidFromCtx(ctx context.Context) (int64, error) {
	switch v := ctx.Value("uid").(type) {
	case float64:
		return int64(v), nil
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	default:
		return 0, errorx.New(errorx.CodeUnauthorized, "令牌中缺少用户信息")
	}
}
