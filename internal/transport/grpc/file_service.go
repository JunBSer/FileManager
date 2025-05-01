package grpc

import (
	"context"
	"github.com/JunBSer/FileManager/internal/service"
	"github.com/JunBSer/proto_fileManager/pkg/api/proto"
)

type FileService struct {
	srv service.FileService
	proto.UnimplementedFileServiceServer
}

func NewService(srv service.FileService) *FileService {
	return &FileService{srv: srv}
}

func (srv *FileService) Upload(stream proto.FileService_UploadServer) error {
	return nil
}

func (srv *FileService) Download(req *proto.FileRequest, stream proto.FileService_DownloadServer) error {
	return nil
}

func (srv *FileService) Delete(ctx context.Context, req *proto.FileRequest) (*proto.StatusResponse, error) {
	return nil, nil
}

func (srv *FileService) Read(req *proto.FileRequest, stream proto.FileService_ReadServer) error {
	return nil
}

func (srv *FileService) OverwriteFile(stream proto.FileService_OverwriteFileServer) error {
	return nil
}

func (srv *FileService) Append(stream proto.FileService_AppendServer) error {
	return nil
}

func (srv *FileService) MoveFile(ctx context.Context, req *proto.OperationRequest) (*proto.StatusResponse, error) {
	return nil, nil
}

func (srv *FileService) ListDirectory(ctx context.Context, r *proto.DirectoryRequest) (*proto.DirectoryResponse, error) {
	return nil, nil
}
