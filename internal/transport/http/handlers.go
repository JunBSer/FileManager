package http

import (
	"github.com/JunBSer/proto_fileManager/pkg/api/proto"
	"net/http"
)

type Config struct {
	Host string `env:"HTTP_HOST" envDefault:"localhost"`
	Port int    `env:"HTTP_PORT" envDefault:"8080"`
}

func (s *FileService) Upload(w *http.ResponseWriter, r *http.Request) {

	return
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
