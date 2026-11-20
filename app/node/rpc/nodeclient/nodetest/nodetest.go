// Package nodetest 提供测试用的 node-rpc 假客户端。
package nodetest

import (
	"context"

	"github.com/hequan2017/go-docker-manage/app/node/rpc/pb"
	"google.golang.org/grpc"
)

// Client 只实现 GetNode:Err 非空时返回 Err,否则返回 Node。其余方法未实现(调用会 panic)。
type Client struct {
	pb.NodeClient
	Node *pb.NodeInfo
	Err  error
}

func (c *Client) GetNode(context.Context, *pb.GetNodeReq, ...grpc.CallOption) (*pb.GetNodeReply, error) {
	if c.Err != nil {
		return nil, c.Err
	}
	return &pb.GetNodeReply{Node: c.Node}, nil
}

// At 返回一个指向 dockerAddr 的节点 1
func At(dockerAddr string) *Client {
	return &Client{Node: &pb.NodeInfo{Id: 1, Name: "test-node", DockerApiUrl: dockerAddr}}
}
