package service

import (
	"github.com/JunBSer/FileManager/internal/repository"
	"github.com/JunBSer/proto_fileManager/pkg/api/proto"
)

type FileService struct {
	repo repository.FileRepository
}

func (srv *FileService) Upload(stream proto.FileService_UploadServer) error {
	file := srv.repo.GetFileHandle()
	return nil
}
