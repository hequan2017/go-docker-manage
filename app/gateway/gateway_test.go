package main

import (
	"errors"
	"net/http"
	"testing"

	"github.com/hequan2017/go-docker-manage/pkg/errorx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestErrorMapping(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantHTTP int
	}{
		{"business error", errorx.New(errorx.CodeInvalidParam, "bad"), errorx.CodeInvalidParam, http.StatusBadRequest},
		{"login failed", status.Error(codes.Unauthenticated, "用户名或密码错误"), errorx.CodeUnauthorized, http.StatusUnauthorized},
		{"not found", status.Error(codes.NotFound, "节点不存在"), errorx.CodeNotFound, http.StatusNotFound},
		{"conflict", status.Error(codes.FailedPrecondition, "容器运行中"), errorx.CodeConflict, http.StatusConflict},
		{"downstream down", status.Error(codes.Unavailable, "refused"), errorx.CodeRpcError, http.StatusBadGateway},
		{"plain error", errors.New("boom"), errorx.CodeInternal, http.StatusInternalServerError},
	}
	for _, tc := range cases {
		code, _ := classify(tc.err)
		if code != tc.wantCode {
			t.Errorf("%s: code = %d, want %d", tc.name, code, tc.wantCode)
		}
		if got := httpStatusOf(code); got != tc.wantHTTP {
			t.Errorf("%s: http = %d, want %d", tc.name, got, tc.wantHTTP)
		}
	}
}
