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
	if err := srv.srv.Upload(stream); err != nil {
		stream.SendAndClose(&proto.StatusResponse{Status: proto.Status_STATUS_ERROR})
		return err
	}
	stream.SendAndClose(&proto.StatusResponse{Status: proto.Status_STATUS_SUCCESS})
	return nil
}

func (srv *FileService) Download(req *proto.FileRequest, stream proto.FileService_DownloadServer) error {
	if err := srv.srv.Download(req, stream); err != nil {
		stream.RecvMsg(&proto.StatusResponse{Status: proto.Status_STATUS_ERROR})
		return err
	}
	stream.RecvMsg(&proto.StatusResponse{Status: proto.Status_STATUS_SUCCESS})
	return nil
}

func (srv *FileService) Delete(ctx context.Context, req *proto.FileRequest) (*proto.StatusResponse, error) {
	err := srv.srv.Delete(ctx, req)
	if err != nil {
		return &proto.StatusResponse{Status: proto.Status_STATUS_ERROR}, err
	}
	return &proto.StatusResponse{Status: proto.Status_STATUS_SUCCESS}, nil
}

func (srv *FileService) Read(req *proto.FileRequest, stream proto.FileService_ReadServer) error {
	err := srv.srv.Read(req, stream)
	if err != nil {
		stream.RecvMsg(&proto.StatusResponse{Status: proto.Status_STATUS_ERROR})
		return err
	}
	stream.RecvMsg(&proto.StatusResponse{Status: proto.Status_STATUS_SUCCESS})
	return nil
}

func (srv *FileService) OverwriteFile(stream proto.FileService_OverwriteFileServer) error {
	if err := srv.srv.Overwrite(stream); err != nil {
		stream.SendAndClose(&proto.StatusResponse{Status: proto.Status_STATUS_ERROR})
		return err
	}
	stream.SendAndClose(&proto.StatusResponse{Status: proto.Status_STATUS_SUCCESS})
	return nil
}

func (srv *FileService) Append(stream proto.FileService_AppendServer) error {
	if err := srv.srv.Append(stream); err != nil {
		stream.SendAndClose(&proto.StatusResponse{Status: proto.Status_STATUS_ERROR})
		return err
	}
	stream.SendAndClose(&proto.StatusResponse{Status: proto.Status_STATUS_SUCCESS})
	return nil
}

func (srv *FileService) MoveFile(ctx context.Context, req *proto.OperationRequest) (*proto.StatusResponse, error) {
	err := srv.srv.MoveFile(ctx, req)
	if err != nil {
		return &proto.StatusResponse{Status: proto.Status_STATUS_ERROR}, err
	}
	return &proto.StatusResponse{Status: proto.Status_STATUS_SUCCESS}, nil
}

func (srv *FileService) ListDirectory(ctx context.Context, r *proto.DirectoryRequest) (*proto.DirectoryResponse, error) {
	res, err := srv.srv.ListDirectory(ctx, r)
	if err != nil {
		return nil, err
	}
	return &proto.DirectoryResponse{Entries: res}, nil
}
