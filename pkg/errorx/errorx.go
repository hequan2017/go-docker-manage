// Package errorx 提供统一业务错误码与错误类型,所有服务共用。
package errorx

import "fmt"

// 统一业务错误码
const (
	CodeOK           = 0
	CodeInvalidParam = 40001
	CodeUnauthorized = 40101
	CodeNotFound     = 40401
	CodeConflict     = 40901
	CodeInternal     = 50001
	CodeRpcError     = 50002
	CodeDockerError  = 50003
)

// StatusError 携带业务错误码的错误
type StatusError struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func (e *StatusError) Error() string { return fmt.Sprintf("[%d] %s", e.Code, e.Msg) }

// New 创建业务错误
func New(code int, msg string) error { return &StatusError{Code: code, Msg: msg} }

// Newf 创建带格式化信息的业务错误
func Newf(code int, format string, args ...any) error {
	return &StatusError{Code: code, Msg: fmt.Sprintf(format, args...)}
}
