package service

import (
	"github.com/JunBSer/FileManager/internal/repository"
)

type FileService struct {
	repo repository.FileRepository
}

func (srv *FileService) Upload() error {

	return nil
}
