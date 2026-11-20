package logic

import (
	"context"
	"encoding/json"
	"testing"
)

// go-zero 的 JWT 中间件用 jwt.WithJSONNumber() 解析令牌,
// 写入 ctx 的数字 claim 是 json.Number
func TestUidFromCtxJSONNumber(t *testing.T) {
	ctx := context.WithValue(context.Background(), "uid", json.Number("42"))
	uid, err := uidFromCtx(ctx)
	if err != nil {
		t.Fatalf("uidFromCtx: %v", err)
	}
	if uid != 42 {
		t.Fatalf("uid = %d, want 42", uid)
	}
}

func TestUidFromCtxMissing(t *testing.T) {
	if _, err := uidFromCtx(context.Background()); err == nil {
		t.Fatal("expected error when uid claim is missing")
	}
}
