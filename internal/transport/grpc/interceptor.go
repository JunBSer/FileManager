package grpc

import (
	"context"
	"github.com/JunBSer/FileManager/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func unContextWithLogger(l logger.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		l.Info(ctx, "request started", zap.String("method", info.FullMethod))
		return handler(context.WithValue(ctx, logger.Key, l), req)
	}
}

func clStrContextWithLogger(l logger.Logger) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		l.Info(ctx, "request started", zap.String("method", method))
		return streamer(context.WithValue(ctx, logger.Key, l), desc, cc, method, opts...)
	}
}

func srvStrContextWithLogger(l logger.Logger) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		l.Info(ss.Context(), "request started", zap.String("method", info.FullMethod))
		ctx := context.WithValue(ss.Context(), logger.Key, l)
		wrappedStream := &wrappedStream{ServerStream: ss, ctx: ctx}

		return handler(srv, wrappedStream)
	}
}
