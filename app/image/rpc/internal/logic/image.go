// 镜像管理逻辑:列表/拉取/删除,直接透传目标节点的 Docker API
package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/docker/docker/api/types/image"
	"github.com/hequan2017/go-docker-manage/app/image/rpc/internal/svc"
	"github.com/hequan2017/go-docker-manage/app/image/rpc/pb"
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

func (l *ImageLogic) nodeOf(nodeId int64) (*nodepb.NodeInfo, error) {
	resp, err := l.svcCtx.NodeCli.GetNode(l.ctx, &nodepb.GetNodeReq{Id: nodeId})
	if err != nil {
		if st, ok := status.FromError(err); ok {
			return nil, status.Errorf(codes.NotFound, "节点不存在: %s", st.Message())
		}
		return nil, err
	}
	return resp.Node, nil
}

func (l *ImageLogic) ListImages(in *pb.ListImagesReq) (*pb.ListImagesReply, error) {
	node, err := l.nodeOf(in.NodeId)
	if err != nil {
		return nil, err
	}
	cli, err := dockerx.New(node.DockerApiUrl)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Docker API 地址无效: %v", err)
	}
	images, err := cli.ImageList(l.ctx, image.ListOptions{})
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "查询镜像列表失败: %v", err)
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

func (l *ImageLogic) PullImage(in *pb.PullImageReq) (*pb.PullImageReply, error) {
	if in.Ref == "" {
		return nil, status.Error(codes.InvalidArgument, "镜像名不能为空")
	}
	node, err := l.nodeOf(in.NodeId)
	if err != nil {
		return nil, err
	}
	cli, err := dockerx.New(node.DockerApiUrl)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Docker API 地址无效: %v", err)
	}
	reader, err := cli.ImagePull(l.ctx, in.Ref, image.PullOptions{})
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "拉取镜像失败: %v", err)
	}
	defer reader.Close()

	// 消费进度流直到 EOF,取最后一条状态
	var statuses []string
	dec := json.NewDecoder(reader)
	for {
		var msg struct {
			Status string `json:"status"`
		}
		if err := dec.Decode(&msg); err != nil {
			if err == io.EOF {
				break
			}
			return nil, status.Errorf(codes.Internal, "读取拉取进度失败: %v", err)
		}
		if msg.Status != "" {
			statuses = append(statuses, msg.Status)
		}
	}
	last := "拉取完成"
	if len(statuses) > 0 {
		last = statuses[len(statuses)-1]
	}
	l.Infof("image %s pulled on node %s: %s", in.Ref, node.Name, last)
	return &pb.PullImageReply{Status: fmt.Sprintf("%s: %s", in.Ref, last)}, nil
}

func (l *ImageLogic) RemoveImage(in *pb.RemoveImageReq) (*pb.RemoveImageReply, error) {
	node, err := l.nodeOf(in.NodeId)
	if err != nil {
		return nil, err
	}
	cli, err := dockerx.New(node.DockerApiUrl)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Docker API 地址无效: %v", err)
	}
	resp, err := cli.ImageRemove(l.ctx, in.ImageRef, image.RemoveOptions{Force: in.Force})
	if err != nil {
		if strings.Contains(err.Error(), "No such image") {
			return nil, status.Errorf(codes.NotFound, "镜像 %s 不存在", in.ImageRef)
		}
		if strings.Contains(err.Error(), "conflict") {
			return nil, status.Errorf(codes.FailedPrecondition, "镜像被容器使用,需强制删除: %v", err)
		}
		return nil, status.Errorf(codes.Unavailable, "删除镜像失败: %v", err)
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
