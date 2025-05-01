package grpc

import (
	"context"
	"fmt"
	"github.com/JunBSer/FileManager/pkg/logger"
	"github.com/JunBSer/proto_fileManager/pkg/api/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type Client struct {
	Conn *grpc.ClientConn
	Cl   proto.FileServiceClient
}

func NewClient(ctx context.Context, host string, port int) (*Client, error) {
	var opts []grpc.DialOption

	conn, err := grpc.NewClient(fmt.Sprintf("%s:%d", host, port), opts...)
	if err != nil {
		logger.GetLoggerFromContext(ctx).Error(ctx, "Error to create conn", zap.Error(err))
		return nil, err
	}

	logger.GetLoggerFromContext(ctx).Info(ctx, "Grpc client connection has been created")

	cl := proto.NewFileServiceClient(conn)
	return &Client{Conn: conn,
		Cl: cl}, nil
}

func (c *Client) Close(ctx context.Context) {
	err := c.Conn.Close()
	if err != nil {
		logger.GetLoggerFromContext(ctx).Error(ctx, "Error to close conn", zap.Error(err))
		panic(err)
	}
	logger.GetLoggerFromContext(ctx).Info(ctx, "Grpc client connection has been closed")
}
