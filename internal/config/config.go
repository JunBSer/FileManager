package config

import (
	"github.com/JunBSer/FileManager/internal/gateway"
	"github.com/JunBSer/FileManager/internal/repository"
	"github.com/JunBSer/FileManager/internal/transport/grpc"
	"github.com/ilyakaznacheev/cleanenv"
)

type (
	Config struct {
		App     App
		Logger  Log
		GRPc    grpc.Config
		Http    gateway.Config
		Storage repository.FileStorageConfig
		gw      gateway.GwConfig
	}

	App struct {
		ServiceName string `env:"SERVICE_NAME" envDefault:"Unnamed_Service"`
		Version     string `env:"VERSION" envDefault:"1.0.0"`
	}

	Log struct {
		LogLvl string `env:"LOGGER_LEVEL" envDefault:"info"`
	}
)

func New() (*Config, error) {
	cfg := Config{}
	err := cleanenv.ReadConfig("./configs/local.env", &cfg)

	if err != nil {
		return nil, err
	}

	return &cfg, err
}
