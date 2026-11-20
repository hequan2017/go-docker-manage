// gateway —— API 网关:JWT 鉴权 + REST 路由,聚合 user/node/container/image 微服务
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"

	"github.com/hequan2017/go-docker-manage/app/gateway/internal/config"
	"github.com/hequan2017/go-docker-manage/app/gateway/internal/handler"
	"github.com/hequan2017/go-docker-manage/app/gateway/internal/svc"
	"github.com/hequan2017/go-docker-manage/pkg/errorx"
	"github.com/hequan2017/go-docker-manage/pkg/nacosx"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var configFile = flag.String("f", "etc/gateway.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	c.Nacos.OverrideFromEnv()
	nacosx.MustLoadRemote(c.Nacos, c.Name, &c)

	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	// 统一错误响应:业务码 + HTTP 状态(gRPC 错误自动映射)
	httpx.SetErrorHandlerCtx(func(_ context.Context, err error) (int, any) {
		code, msg := classify(err)
		return httpStatusOf(code), map[string]any{"code": code, "msg": msg}
	})

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx, c.Auth.AccessSecret)

	fmt.Printf("Starting gateway at %s:%d...\n", c.Host, c.Port)
	server.Start()
}

// httpStatusOf 业务码 → HTTP 状态码
func httpStatusOf(code int) int {
	switch code {
	case errorx.CodeInvalidParam:
		return http.StatusBadRequest
	case errorx.CodeUnauthorized:
		return http.StatusUnauthorized
	case errorx.CodeNotFound:
		return http.StatusNotFound
	case errorx.CodeConflict:
		return http.StatusConflict
	case errorx.CodeRpcError:
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
}

func httpStatus(err error) int {
	code, _ := classify(err)
	switch code {
	case errorx.CodeInvalidParam:
		return http.StatusBadRequest
	case errorx.CodeUnauthorized:
		return http.StatusUnauthorized
	case errorx.CodeNotFound:
		return http.StatusNotFound
	case errorx.CodeConflict:
		return http.StatusConflict
	case errorx.CodeRpcError:
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
}

// classify 错误 → (业务码, 消息)
func classify(err error) (int, string) {
	if se, ok := err.(*errorx.StatusError); ok {
		return se.Code, se.Msg
	}
	if st, ok := status.FromError(err); ok {
		switch st.Code() {
		case codes.NotFound:
			return errorx.CodeNotFound, st.Message()
		case codes.InvalidArgument:
			return errorx.CodeInvalidParam, st.Message()
		case codes.AlreadyExists, codes.FailedPrecondition:
			return errorx.CodeConflict, st.Message()
		case codes.Unauthenticated:
			return errorx.CodeUnauthorized, st.Message()
		case codes.Unavailable, codes.DeadlineExceeded:
			return errorx.CodeRpcError, "下游服务不可用: " + st.Message()
		default:
			return errorx.CodeInternal, st.Message()
		}
	}
	return errorx.CodeInternal, err.Error()
}
