package app

import (
	"context"
	"github.com/JunBSer/FileManager/internal/config"
	"github.com/JunBSer/FileManager/pkg/logger"
)

func RunApp(cfg *config.Config) {
	ctx := context.Background()

	mainLogger := logger.New(cfg.App.ServiceName, cfg.Logger.LogLvl)
	ctx = context.WithValue(ctx, logger.Key, mainLogger)

	mainLogger.Info(ctx, "Starting file-service...")

}
