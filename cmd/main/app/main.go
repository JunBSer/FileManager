package main

import (
	"github.com/JunBSer/FileManager/internal/app/App"
	"github.com/JunBSer/FileManager/internal/config"
)

func main() {
	cfg, err := config.New()
	if err != nil {
		panic(err)
	}

	App.MustRun(cfg)
}
