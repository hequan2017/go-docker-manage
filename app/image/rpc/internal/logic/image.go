// 镜像管理逻辑:列表/拉取/删除,直接透传目标节点的 Docker API
package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/hequan2017/go-docker-manage/app/image/rpc/internal/svc"
	"github.com/hequan2017/go-docker-manage/app/image/rpc/pb"
	"github.com/hequan2017/go-docker-manage/app/node/rpc/nodeclient"
	nodepb "github.com/hequan2017/go-docker-manage/app/node/rpc/pb"
	"github.com/hequan2017/go-docker-manage/pkg/dockerx"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ImageLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewImageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ImageLogic {
	return &ImageLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

// docker 查询节点并返回其 Docker 客户端(错误已转为 gRPC status)
func (l *ImageLogic) docker(nodeId int64) (*nodepb.NodeInfo, *client.Client, error) {
	return nodeclient.Docker(l.ctx, l.svcCtx.NodeCli, nodeId)
}

func (l *ImageLogic) ListImages(in *pb.ListImagesReq) (*pb.ListImagesReply, error) {
	_, cli, err := l.docker(in.NodeId)
	if err != nil {
		return nil, err
	}
	images, err := cli.ImageList(l.ctx, image.ListOptions{})
	if err != nil {
		return nil, dockerx.ToStatus(err)
	}
	reply := &pb.ListImagesReply{}
	for _, img := range images {
		reply.Images = append(reply.Images, &pb.ImageInfo{
			Id:       img.ID,
			RepoTags: img.RepoTags,
			Created:  img.Created,
			Size:     img.Size,
		})
	}
	return reply, nil
}

// pullMessage Docker 拉取进度流中的一条消息;失败时 HTTP 仍为 200,错误放在 error 字段
type pullMessage struct {
	Status string `json:"status"`
	Error  string `json:"error"`
}

func (l *ImageLogic) PullImage(in *pb.PullImageReq) (*pb.PullImageReply, error) {
	if in.Ref == "" {
		return nil, status.Error(codes.InvalidArgument, "镜像名不能为空")
	}
	node, cli, err := l.docker(in.NodeId)
	if err != nil {
		return nil, err
	}
	reader, err := cli.ImagePull(l.ctx, in.Ref, image.PullOptions{})
	if err != nil {
		return nil, dockerx.ToStatus(err)
	}
	defer reader.Close()

	// 消费进度流直到 EOF,遇到 error 消息立即失败,否则取最后一条状态
	last := "拉取完成"
	dec := json.NewDecoder(reader)
	for {
		var msg pullMessage
		if err := dec.Decode(&msg); err != nil {
			if err == io.EOF {
				break
			}
			return nil, status.Errorf(codes.Internal, "读取拉取进度失败: %v", err)
		}
		if msg.Error != "" {
			return nil, status.Errorf(codes.Aborted, "拉取镜像 %s 失败: %s", in.Ref, msg.Error)
		}
		if msg.Status != "" {
			last = msg.Status
		}
	}
	l.Infof("image %s pulled on node %s: %s", in.Ref, node.Name, last)
	return &pb.PullImageReply{Status: fmt.Sprintf("%s: %s", in.Ref, last)}, nil
}

func (l *ImageLogic) RemoveImage(in *pb.RemoveImageReq) (*pb.RemoveImageReply, error) {
	_, cli, err := l.docker(in.NodeId)
	if err != nil {
		return nil, err
	}
	resp, err := cli.ImageRemove(l.ctx, in.ImageRef, image.RemoveOptions{Force: in.Force})
	if err != nil {
		st := status.Convert(dockerx.ToStatus(err))
		switch st.Code() {
		case codes.NotFound:
			return nil, status.Errorf(codes.NotFound, "镜像 %s 不存在", in.ImageRef)
		case codes.FailedPrecondition:
			return nil, status.Errorf(codes.FailedPrecondition, "镜像被容器使用,需强制删除: %s", st.Message())
		default:
			return nil, st.Err()
		}
	}
	reply := &pb.RemoveImageReply{}
	for _, r := range resp {
		if r.Untagged != "" {
			reply.Deleted = append(reply.Deleted, "Untagged: "+r.Untagged)
		}
		if r.Deleted != "" {
			reply.Deleted = append(reply.Deleted, "Deleted: "+r.Deleted)
		}
	}
	return reply, nil
}
