// Package dockertest 提供测试用的最小 Docker Engine API 服务器。
package dockertest

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

var versionPrefix = regexp.MustCompile(`^/v\d+\.\d+`)

// NewServer 启动假 Docker API,返回 tcp://host:port 地址。
// routes 的 key 为 "METHOD /path"(不含 /v1.xx 版本前缀);/_ping 已内置,
// 未匹配的请求返回 Docker 风格的 404 {"message": ...}。
func NewServer(t *testing.T, routes map[string]http.HandlerFunc) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := versionPrefix.ReplaceAllString(r.URL.Path, "")
		if path == "/_ping" {
			w.Header().Set("API-Version", "1.47")
			_, _ = w.Write([]byte("OK"))
			return
		}
		if h, ok := routes[r.Method+" "+path]; ok {
			h(w, r)
			return
		}
		Error(w, http.StatusNotFound, "not found: "+r.Method+" "+path)
	}))
	t.Cleanup(srv.Close)
	return "tcp://" + strings.TrimPrefix(srv.URL, "http://")
}

// Error 写 Docker 风格错误响应
func Error(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, _ = w.Write([]byte(`{"message":` + quote(msg) + `}`))
}

func quote(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}
