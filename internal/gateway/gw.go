package gateway

import (
	"context"
	"github.com/JunBSer/FileManager/internal/transport/grpc"
	"github.com/JunBSer/FileManager/pkg/logger"
)

type Gateway struct {
	client *grpc.Client
}

func New(ctx context.Context, host string, port int) (*Gateway, error) {
	client, err := grpc.NewClient(ctx, host, port)
	if err != nil {
		return nil, err
	}

	logger.GetLoggerFromContext(ctx).Info(ctx, "Gateway created successfully")

	return &Gateway{client: client}, nil
}

func (gw *Gateway) Setup() error {

}
