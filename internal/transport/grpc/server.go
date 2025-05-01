package grpc

import (
	"context"
	"fmt"
	"github.com/JunBSer/FileManager/internal/service"
	"github.com/JunBSer/FileManager/pkg/logger"
	pb "github.com/JunBSer/proto_fileManager/pkg/api/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"net"
)

type Config struct {
	GRPCHost string `env:"GRPC_HOST" envDefault:"localhost"`
	GRPCPort int    `env:"GRPC_PORT" envDefault:"50051"`
}

type Server struct {
	Grpc     *grpc.Server
	Listener net.Listener
}

func New(ctx context.Context, grpcConfig *Config, srv service.FileService) (*Server, error) {
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

	return &Server{Grpc: grpcServer, Listener: lis}, nil
}

func (s *Server) Start(ctx context.Context) error {
	logger.GetLoggerFromContext(ctx).Info(ctx, "Starting gRPC server", zap.String("host", s.Listener.Addr().(*net.TCPAddr).IP.String()), zap.Int("port", s.Listener.Addr().(*net.TCPAddr).Port))
	return s.Grpc.Serve(s.Listener)
}

func (s *Server) Stop(ctx context.Context) {
	lg := logger.GetLoggerFromContext(ctx)
	lg.Info(ctx, "Stopping gRPC server")

	s.Grpc.GracefulStop()
	lg.Info(ctx, "Grpc server stopped")
}
