// Package nodeclient 封装"按节点 ID 拿到 Docker 客户端"的公共流程,
// 供 container-rpc / image-rpc 共用。
package nodeclient

import (
	"context"

	dc "github.com/docker/docker/client"
	"github.com/hequan2017/go-docker-manage/app/node/rpc/pb"
	"github.com/hequan2017/go-docker-manage/pkg/dockerx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Docker 向 node-rpc 查询节点,返回节点信息和指向其 Docker API 的客户端。
// 返回的错误均为 gRPC status:节点不存在 → NotFound;
// node-rpc 自身故障(Unavailable / DeadlineExceeded 等)保留原状态码,不误报为"节点不存在"。
func Docker(ctx context.Context, cli pb.NodeClient, nodeId int64) (*pb.NodeInfo, *dc.Client, error) {
	resp, err := cli.GetNode(ctx, &pb.GetNodeReq{Id: nodeId})
	if err != nil {
		st := status.Convert(err)
		if st.Code() == codes.NotFound {
			return nil, nil, status.Errorf(codes.NotFound, "节点不存在: %s", st.Message())
		}
		return nil, nil, status.Errorf(st.Code(), "查询节点失败: %s", st.Message())
	}
	node := resp.Node
	c, err := dockerx.New(node.DockerApiUrl)
	if err != nil {
		return nil, nil, status.Errorf(codes.InvalidArgument, "节点 %s 的 Docker API 地址无效: %v", node.Name, err)
	}
	return node, c, nil
}
