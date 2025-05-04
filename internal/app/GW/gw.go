package GW

import (
	"context"
	"github.com/JunBSer/FileManager/internal/config"
	"github.com/JunBSer/FileManager/internal/gateway"
	"github.com/JunBSer/FileManager/pkg/logger"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"syscall"
)

func MustRun(cfg *config.Config) {
	ctx := context.Background()

	mainLogger := logger.New("Gateway", cfg.Logger.LogLvl)
	ctx = context.WithValue(ctx, logger.Key, mainLogger)

	mainLogger.Info(ctx, "Starting gateway...")

	gw, err := gateway.New(ctx, &cfg.GRPc, &cfg.Http, &cfg.Gw)
	if err != nil {
		mainLogger.Error(ctx, "Error occurred while creating gateway", zap.Error(err))
		panic(err)
	}

	graceCh := make(chan os.Signal, 2)
	signal.Notify(graceCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		err = gw.Start(ctx)
		if err != nil {
			mainLogger.Error(ctx, "Error occurred while running GRPC server", zap.Error(err))
		}
	}()

	sig := <-graceCh
	mainLogger.Info(ctx, "Shutting down...", zap.String("signal", sig.String()))
	gw.Stop(ctx)
}
