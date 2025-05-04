package App

import (
	"context"
	"github.com/JunBSer/FileManager/internal/config"
	"github.com/JunBSer/FileManager/internal/repository"
	"github.com/JunBSer/FileManager/internal/service"
	"github.com/JunBSer/FileManager/internal/transport/grpc"
	"github.com/JunBSer/FileManager/pkg/logger"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"syscall"
)

func MustRun(cfg *config.Config) {
	ctx := context.Background()

	mainLogger := logger.New(cfg.App.ServiceName, cfg.Logger.LogLvl)
	ctx = context.WithValue(ctx, logger.Key, mainLogger)

	mainLogger.Info(ctx, "Starting file-service...")

	fileRepo := repository.New(cfg.Storage.StoragePath, cfg.Storage.MaxSize, cfg.Storage.ReadSize)
	fileService := service.New(fileRepo)

	grpcServer, err := grpc.New(ctx, &cfg.GRPc, fileService)
	if err != nil {
		panic(err)
	}

	graceCh := make(chan os.Signal, 2)
	signal.Notify(graceCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		err = grpcServer.Start(ctx)
		if err != nil {
			mainLogger.Error(ctx, "Error occurred while running GRPC server", zap.Error(err))
		}
	}()

	sig := <-graceCh
	mainLogger.Info(ctx, "Shutting down...", zap.String("signal", sig.String()))
	grpcServer.Stop(ctx)
}
