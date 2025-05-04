package main

import (
	_ "github.com/JunBSer/FileManager/docs"
	"github.com/JunBSer/FileManager/internal/app/GW"
	"github.com/JunBSer/FileManager/internal/config"
)

//	@title			Swagger Example API
//	@version		1.0
//	@description	Api for file management

//	@license.name	Apache 2.0
//	@license.url	http://www.apache.org/licenses/LICENSE-2.0.html

// @host		localhost:8080
// @BasePath    /api/v1/files
func main() {
	cfg, err := config.New()
	if err != nil {
		panic(err)
	}

	GW.MustRun(cfg)
}
