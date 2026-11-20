package logic

import (
	"context"
	"errors"
	"testing"
	"time"

	containerpb "github.com/hequan2017/go-docker-manage/app/container/rpc/pb"
	"github.com/hequan2017/go-docker-manage/app/gateway/internal/svc"
	imagepb "github.com/hequan2017/go-docker-manage/app/image/rpc/pb"
	nodepb "github.com/hequan2017/go-docker-manage/app/node/rpc/pb"
	"google.golang.org/grpc"
)

const fakeLatency = 100 * time.Millisecond

type fakeNodes struct {
	nodepb.NodeClient
	nodes []*nodepb.NodeInfo
}

func (f *fakeNodes) ListNodes(context.Context, *nodepb.ListNodesReq, ...grpc.CallOption) (*nodepb.ListNodesReply, error) {
	return &nodepb.ListNodesReply{Nodes: f.nodes}, nil
}

// fakeContainers 每个节点 2 个容器(1 个 running);failNode 上返回错误
type fakeContainers struct {
	containerpb.ContainerClient
	failNode int64
}

func (f *fakeContainers) ListContainers(_ context.Context, in *containerpb.ListContainersReq, _ ...grpc.CallOption) (*containerpb.ListContainersReply, error) {
	time.Sleep(fakeLatency)
	if in.NodeId == f.failNode {
		return nil, errors.New("docker down")
	}
	return &containerpb.ListContainersReply{Containers: []*containerpb.ContainerSummary{
		{Id: "a", State: "running"}, {Id: "b", State: "exited"},
	}}, nil
}

// fakeImages 每个节点 3 个镜像
type fakeImages struct{ imagepb.ImageClient }

func (fakeImages) ListImages(context.Context, *imagepb.ListImagesReq, ...grpc.CallOption) (*imagepb.ListImagesReply, error) {
	time.Sleep(fakeLatency)
	return &imagepb.ListImagesReply{Images: []*imagepb.ImageInfo{{Id: "1"}, {Id: "2"}, {Id: "3"}}}, nil
}

func TestDashboardStats(t *testing.T) {
	nodes := []*nodepb.NodeInfo{
		{Id: 1, Name: "n1", Status: "healthy"},
		{Id: 2, Name: "n2", Status: "healthy"},
		{Id: 3, Name: "n3", Status: "unhealthy"},
		{Id: 4, Name: "n4", Status: "healthy"},
	}
	svcCtx := &svc.ServiceContext{
		NodeRpc:      &fakeNodes{nodes: nodes},
		ContainerRpc: &fakeContainers{failNode: 3},
		ImageRpc:     fakeImages{},
	}

	start := time.Now()
	got, err := NewDashboardLogic(context.Background(), svcCtx).Stats()
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}

	// 串行需 4 节点 × 2 次调用 × 100ms = 800ms;并发应接近单次调用耗时
	if elapsed > 4*fakeLatency {
		t.Errorf("Stats took %v, want < %v (queries should run concurrently)", elapsed, 4*fakeLatency)
	}
	if got.NodeTotal != 4 || got.NodeHealthy != 3 {
		t.Errorf("nodes = %d/%d healthy, want 4/3", got.NodeTotal, got.NodeHealthy)
	}
	if got.ContainerTotal != 6 || got.ContainerRunning != 3 {
		t.Errorf("containers = %d (%d running), want 6 (3 running)", got.ContainerTotal, got.ContainerRunning)
	}
	if got.ImageTotal != 12 {
		t.Errorf("images = %d, want 12", got.ImageTotal)
	}
	if len(got.Nodes) != 4 {
		t.Fatalf("len(Nodes) = %d, want 4", len(got.Nodes))
	}
	for i, n := range got.Nodes { // 保持节点原有顺序
		if n.Id != nodes[i].Id {
			t.Errorf("Nodes[%d].Id = %d, want %d", i, n.Id, nodes[i].Id)
		}
	}
	if got.Nodes[2].Error == "" {
		t.Error("node 3 should report container query error")
	}
}
