package logic

import (
	"context"
	"net/http"
	"testing"

	"github.com/hequan2017/go-docker-manage/app/image/rpc/internal/svc"
	"github.com/hequan2017/go-docker-manage/app/image/rpc/pb"
	"github.com/hequan2017/go-docker-manage/app/node/rpc/nodeclient/nodetest"
	nodepb "github.com/hequan2017/go-docker-manage/app/node/rpc/pb"
	"github.com/hequan2017/go-docker-manage/pkg/dockerx/dockertest"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func newLogic(nodeCli nodepb.NodeClient) *ImageLogic {
	return NewImageLogic(context.Background(), &svc.ServiceContext{NodeCli: nodeCli})
}

// Docker 拉取失败时 HTTP 仍是 200,错误写在进度流的 {"error": ...} 里
func TestPullImageReportsStreamError(t *testing.T) {
	addr := dockertest.NewServer(t, map[string]http.HandlerFunc{
		"POST /images/create": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"Pulling from library/nope"}` + "\n" +
				`{"errorDetail":{"message":"manifest unknown"},"error":"manifest unknown"}` + "\n"))
		},
	})
	l := newLogic(nodetest.At(addr))

	_, err := l.PullImage(&pb.PullImageReq{NodeId: 1, Ref: "nope:latest"})
	if err == nil {
		t.Fatal("expected error when pull stream reports an error")
	}
}

func TestPullImageSuccess(t *testing.T) {
	addr := dockertest.NewServer(t, map[string]http.HandlerFunc{
		"POST /images/create": func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"status":"Pulling"}` + "\n" + `{"status":"Status: Downloaded newer image for nginx:latest"}` + "\n"))
		},
	})
	l := newLogic(nodetest.At(addr))

	reply, err := l.PullImage(&pb.PullImageReq{NodeId: 1, Ref: "nginx:latest"})
	if err != nil {
		t.Fatalf("PullImage: %v", err)
	}
	if reply.Status != "nginx:latest: Status: Downloaded newer image for nginx:latest" {
		t.Fatalf("status = %q", reply.Status)
	}
}

func TestImageNodeRpcUnavailableIsNotReportedAsNotFound(t *testing.T) {
	l := newLogic(&nodetest.Client{Err: status.Error(codes.Unavailable, "connection refused")})

	_, err := l.ListImages(&pb.ListImagesReq{NodeId: 1})
	if got := status.Code(err); got != codes.Unavailable {
		t.Fatalf("code = %v, want Unavailable (err=%v)", got, err)
	}
}

func TestRemoveImageErrorCodes(t *testing.T) {
	addr := dockertest.NewServer(t, map[string]http.HandlerFunc{
		"DELETE /images/busy:latest": func(w http.ResponseWriter, _ *http.Request) {
			dockertest.Error(w, http.StatusConflict, "unable to delete busy:latest (must be forced) - image is being used by running container abc")
		},
	})
	l := newLogic(nodetest.At(addr))

	cases := map[string]codes.Code{
		"missing:latest": codes.NotFound,
		"busy:latest":    codes.FailedPrecondition,
	}
	for ref, want := range cases {
		_, err := l.RemoveImage(&pb.RemoveImageReq{NodeId: 1, ImageRef: ref})
		if got := status.Code(err); got != want {
			t.Errorf("%s: code = %v, want %v (err=%v)", ref, got, want, err)
		}
	}
}
