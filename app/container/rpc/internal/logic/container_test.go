package logic

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hequan2017/go-docker-manage/app/container/rpc/internal/svc"
	"github.com/hequan2017/go-docker-manage/app/container/rpc/pb"
	"github.com/hequan2017/go-docker-manage/app/node/rpc/nodeclient/nodetest"
	nodepb "github.com/hequan2017/go-docker-manage/app/node/rpc/pb"
	"github.com/hequan2017/go-docker-manage/pkg/dockerx/dockertest"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func newLogic(nodeCli nodepb.NodeClient) *ContainerLogic {
	return NewContainerLogic(context.Background(), &svc.ServiceContext{NodeCli: nodeCli})
}

func TestContainerActionAcceptsShortID(t *testing.T) {
	addr := dockertest.NewServer(t, map[string]http.HandlerFunc{
		"POST /containers/web/start": func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		},
	})
	l := newLogic(nodetest.At(addr))

	reply, err := l.ContainerAction(&pb.ContainerActionReq{NodeId: 1, ContainerId: "web", Action: "start"})
	if err != nil {
		t.Fatalf("ContainerAction: %v", err)
	}
	if !reply.Ok || !strings.Contains(reply.Message, "web") {
		t.Fatalf("unexpected reply: %+v", reply)
	}
}

func TestNodeRpcUnavailableIsNotReportedAsNotFound(t *testing.T) {
	l := newLogic(&nodetest.Client{Err: status.Error(codes.Unavailable, "connection refused")})

	_, err := l.ListContainers(&pb.ListContainersReq{NodeId: 1})
	if got := status.Code(err); got != codes.Unavailable {
		t.Fatalf("code = %v, want Unavailable (err=%v)", got, err)
	}
}

func TestNodeMissingIsNotFound(t *testing.T) {
	l := newLogic(&nodetest.Client{Err: status.Error(codes.NotFound, "节点 9 不存在")})

	_, err := l.ListContainers(&pb.ListContainersReq{NodeId: 9})
	if got := status.Code(err); got != codes.NotFound {
		t.Fatalf("code = %v, want NotFound (err=%v)", got, err)
	}
}

func TestDockerNotFoundMapsToNotFound(t *testing.T) {
	l := newLogic(nodetest.At(dockertest.NewServer(t, nil))) // 所有容器端点都返回 404

	_, err := l.RemoveContainer(&pb.RemoveContainerReq{NodeId: 1, ContainerId: "nope"})
	if got := status.Code(err); got != codes.NotFound {
		t.Fatalf("code = %v, want NotFound (err=%v)", got, err)
	}
}

func TestDockerConflictMapsToFailedPrecondition(t *testing.T) {
	addr := dockertest.NewServer(t, map[string]http.HandlerFunc{
		"DELETE /containers/web": func(w http.ResponseWriter, _ *http.Request) {
			dockertest.Error(w, http.StatusConflict, "You cannot remove a running container")
		},
	})
	l := newLogic(nodetest.At(addr))

	_, err := l.RemoveContainer(&pb.RemoveContainerReq{NodeId: 1, ContainerId: "web"})
	if got := status.Code(err); got != codes.FailedPrecondition {
		t.Fatalf("code = %v, want FailedPrecondition (err=%v)", got, err)
	}
}
