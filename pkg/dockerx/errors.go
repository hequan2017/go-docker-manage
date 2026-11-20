package dockerx

import (
	"github.com/docker/docker/errdefs"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ToStatus 把 Docker SDK 返回的错误转成 gRPC status。
// SDK 会把 daemon 的 HTTP 错误包装成 errdefs 类型(可能多层),必须用 errdefs.IsXxx 判断。
func ToStatus(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	switch {
	case errdefs.IsNotFound(err):
		return status.Error(codes.NotFound, msg)
	case errdefs.IsConflict(err):
		return status.Error(codes.FailedPrecondition, msg)
	case errdefs.IsInvalidParameter(err):
		return status.Error(codes.InvalidArgument, msg)
	case errdefs.IsUnauthorized(err), errdefs.IsForbidden(err):
		return status.Error(codes.PermissionDenied, msg)
	case errdefs.IsNotImplemented(err):
		return status.Error(codes.Unimplemented, msg)
	case errdefs.IsDeadline(err):
		return status.Error(codes.DeadlineExceeded, msg)
	default:
		return status.Error(codes.Unavailable, msg)
	}
}
