package gateway

import (
	"context"
	"fmt"
	"github.com/JunBSer/FileManager/internal/transport/grpc"
	"github.com/JunBSer/FileManager/pkg/logger"
	"github.com/gorilla/mux"
	"net/http"
)

type Config struct {
	Host string `env:"HTTP_HOST" envDefault:"localhost"`
	Port int    `env:"HTTP_PORT" envDefault:"8080"`
}

type GwConfig struct {
	maxSize int64 `env:"MAX_SIZE" envDefault:"32"`
}
type Gateway struct {
	client  *grpc.Client
	srv     *http.Server
	maxSize int64
}

func New(ctx *context.Context, grpcConfig *grpc.Config, httpConfig Config, gwConf GwConfig) (*Gateway, error) {
	client, err := grpc.NewClient(*ctx, grpcConfig.GRPCHost, grpcConfig.GRPCPort)
	if err != nil {
		return nil, err
	}

	logger.GetLoggerFromContext(*ctx).Info(*ctx, "Gateway created successfully")

	router := mux.NewRouter()
	rest := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", httpConfig.Host, httpConfig.Port),
		Handler: LoggerMiddleware(*ctx, CorsMiddleware(router)),
	}

	return &Gateway{client: client, srv: rest, maxSize: gwConf.maxSize}, nil
}

func (gw *Gateway) Start(ctx context.Context) error {
	logger.GetLoggerFromContext(ctx).Info(ctx, "Starting HTTP server __ gateway__")
	return gw.srv.ListenAndServe()
}

func (gw *Gateway) Stop(ctx context.Context) error {
	logger.GetLoggerFromContext(ctx).Info(ctx, "Stopping gRPC server")
	err := gw.srv.Shutdown(context.Background())
	if err != nil {
		logger.GetLoggerFromContext(ctx).Info(ctx, "Failed to shutdown HTTP server")
	}
	return err
}
