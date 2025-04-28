package grpc

import (
	"context"
	"fmt"
	"github.com/JunBSer/FileManager/internal/service"
	rest "github.com/JunBSer/FileManager/internal/transport/http"
	"github.com/JunBSer/FileManager/pkg/logger"
	pb "github.com/JunBSer/proto_fileManager/pkg/api/proto"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"net"
	"net/http"
)

type Config struct {
	GRPCHost string `env:"GRPC_HOST" envDefault:"localhost"`
	GRPCPort int    `env:"GRPC_PORT" envDefault:"50051"`
}

type Server struct {
	Grpc     *grpc.Server
	Rest     *http.Server
	Listener net.Listener
}

func New(ctx context.Context, grpcConfig *Config, httpConfig *rest.Config, srv service.FileService) (*Server, error) {
	lg := logger.GetLoggerFromContext(ctx)

	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", (*grpcConfig).GRPCHost, (*grpcConfig).GRPCPort))
	if err != nil {
		lg.Error(ctx, fmt.Sprintf("Grpc server: Failed to listen: %v", err))
		return nil, err
	}
	lg.Info(ctx, fmt.Sprintf("Created grpc server listening on %s:%d", (*grpcConfig).GRPCHost, (*grpcConfig).GRPCPort))

	var opts []grpc.ServerOption

	grpcServer := grpc.NewServer(opts...)
	lg.Info(ctx, "Created grpc server")

	pb.RegisterFileServiceServer(grpcServer, NewService(srv))
	lg.Info(ctx, "GRPC service has been registered")

	var gwmux = runtime.NewServeMux()
	if err = pb.RegisterFileServiceHandlerServer(context.Background(), gwmux, NewService(srv)); err != nil {
		lg.Error(ctx, fmt.Sprintf("Rest server: failed to regigter grpc gateway %v", err))
		return nil, err
	}

	httpServer := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", (*httpConfig).Host, (*httpConfig).Port),
		Handler: gwmux,
	}

	return &Server{Grpc: grpcServer, Rest: httpServer, Listener: lis}, nil
}

func (s *Server) Start(ctx context.Context) error {
	eg := errgroup.Group{}

	eg.Go(func() error {
		logger.GetLoggerFromContext(ctx).Info(ctx, "Starting gRPC server", zap.String("host", s.Listener.Addr().(*net.TCPAddr).IP.String()), zap.Int("port", s.Listener.Addr().(*net.TCPAddr).Port))
		return s.Grpc.Serve(s.Listener)
	})
	eg.Go(func() error {
		logger.GetLoggerFromContext(ctx).Info(ctx, "Starting http server", zap.String("addr", s.Rest.Addr))
		return s.Rest.ListenAndServe()
	})

	return eg.Wait()
}

func (s *Server) Stop(ctx context.Context) error {
	lg := logger.GetLoggerFromContext(ctx)
	lg.Info(ctx, "Stopping gRPC server")

	s.Grpc.GracefulStop()
	lg.Info(ctx, "Grpc server stopped")

	err := s.Rest.Shutdown(ctx)
	if err != nil {
		lg.Error(ctx, "Error to stop http server", zap.Error(err))
	}
	lg.Info(ctx, "Rest server stopped")
	return err
}
