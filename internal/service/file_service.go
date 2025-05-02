package service

import (
	"context"
	"github.com/JunBSer/FileManager/internal/repository"
	"github.com/JunBSer/FileManager/pkg/logger"
	"github.com/JunBSer/proto_fileManager/pkg/api/proto"
	"go.uber.org/zap"
	"io"
)

type FileService struct {
	repo repository.FileRepository
}

func (srv *FileService) ProcessUpload(
	ctx context.Context,
	stream proto.FileService_UploadServer,
	file repository.FileHandle,
	lg logger.Logger,
	pos int64) error {

	for {
		data, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				lg.Info(ctx, "EOF received")
				return nil
			}
			lg.Error(ctx, "Error to read data", zap.Error(err))
			return err
		}

		n, err := srv.repo.AppendData(ctx, file, data.Content, pos)
		if err != nil {
			lg.Error(ctx, "Error to append data", zap.Error(err))
			return err
		}
		pos += n
	}
}

func (srv *FileService) ProcessDownload(file repository.FileHandle, stream proto.FileService_DownloadServer, fileName string) error {
	bufLen := srv.repo.GetReadSize()
	for {
		buf := make([]byte, bufLen)
		_, err := file.Read(buf)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		err = stream.Send(&proto.FileChunk{FileName: fileName, Content: buf})
		if err != nil {
			return err
		}
	}
}

func (srv *FileService) Upload(stream proto.FileService_UploadServer) error {
	ctx := stream.Context()
	lg := logger.GetLoggerFromContext(ctx)

	lg.Info(ctx, "Upload is in process")

	data, err := stream.Recv()
	if err != nil {
		lg.Error(ctx, "Error to request data", zap.Error(err))
		return err
	}

	file, err := srv.repo.GetFileHandle(ctx, data.FileName, repository.CreateAndW)
	if err != nil {
		lg.Error(ctx, "Error to open file", zap.Error(err))
		return err
	}

	defer file.Close()

	pos, err := srv.repo.AppendData(ctx, file, data.Content, 0)
	if err != nil {
		lg.Error(ctx, "Error to append data", zap.Error(err))
		return err
	}

	return srv.ProcessUpload(ctx, stream, file, lg, pos)
}

func (srv *FileService) Append(stream proto.FileService_AppendServer) error {
	ctx := stream.Context()
	lg := logger.GetLoggerFromContext(ctx)

	lg.Info(ctx, "Append is in process")

	data, err := stream.Recv()
	if err != nil {
		lg.Error(ctx, "Error to request data", zap.Error(err))
		return err
	}

	file, err := srv.repo.GetFileHandle(ctx, data.FileName, repository.CreateAndW)
	if err != nil {
		lg.Error(ctx, "Error to open file", zap.Error(err))
		return err
	}

	defer file.Close()

	_, err = srv.repo.AppendData(ctx, file, data.Content, 0)
	if err != nil {
		lg.Error(ctx, "Error to append data", zap.Error(err))
		return err
	}

	info, err := file.Stat()
	if err != nil {
		lg.Error(ctx, "Error to get file info", zap.Error(err))
		return err
	}

	return srv.ProcessUpload(ctx, stream, file, lg, info.Size())
}

func (srv *FileService) Overwrite(stream proto.FileService_OverwriteFileServer) error {
	ctx := stream.Context()
	lg := logger.GetLoggerFromContext(ctx)

	lg.Info(ctx, "Overwrite is in process")

	data, err := stream.Recv()
	if err != nil {
		lg.Error(ctx, "Error to request data", zap.Error(err))
		return err
	}

	file, err := srv.repo.GetFileHandle(ctx, data.FileName, repository.CreateAndW)
	if err != nil {
		lg.Error(ctx, "Error to open file", zap.Error(err))
		return err
	}

	defer file.Close()

	pos, err := srv.repo.AppendData(ctx, file, data.Content, 0)
	if err != nil {
		lg.Error(ctx, "Error to append data", zap.Error(err))
		return err
	}

	return srv.ProcessUpload(ctx, stream, file, lg, pos)
}

func (srv *FileService) Download(req *proto.FileRequest, stream proto.FileService_DownloadServer) error {
	ctx := stream.Context()
	lg := logger.GetLoggerFromContext(ctx)

	lg.Info(ctx, "Download is in process")

	fileName := req.FileName
	file, err := srv.repo.GetFileHandle(ctx, fileName, repository.CreateAndW)
	if err != nil {
		lg.Error(ctx, "Error to open file", zap.Error(err))
		return err
	}

	defer file.Close()

	err = srv.ProcessDownload(file, stream, req.FileName)
	if err != nil {
		lg.Error(ctx, "Error to download file", zap.Error(err))
		return err
	}

	return nil
}

func (srv *FileService) Read(req *proto.FileRequest, stream proto.FileService_ReadServer) error {
	ctx := stream.Context()
	lg := logger.GetLoggerFromContext(ctx)

	lg.Info(ctx, "Read is in process")

	fileName := req.FileName

	file, err := srv.repo.GetFileHandle(ctx, fileName, repository.CreateAndW)
	if err != nil {
		lg.Error(ctx, "Error to open file", zap.Error(err))
		return err
	}
	defer file.Close()

	err = srv.ProcessDownload(file, stream, req.FileName)
	if err != nil {
		lg.Error(ctx, "Error to read file", zap.Error(err))
		return err
	}

	return nil
}

func (srv *FileService) Delete(ctx context.Context, req *proto.FileRequest) error {
	lg := logger.GetLoggerFromContext(ctx)

	lg.Info(ctx, "Delete is in process")
	fileName := req.FileName

	err := srv.repo.DeleteFile(ctx, fileName)
	if err != nil {
		lg.Error(ctx, "Error to delete file", zap.Error(err))
		return err
	}
	return nil
}

func (srv *FileService) MoveFile(ctx context.Context, req *proto.OperationRequest) error {
	lg := logger.GetLoggerFromContext(ctx)

	lg.Info(ctx, "MoveFile is in process")
	destPath, srcPath := req.Destination, req.Source

	err := srv.repo.CopyFile(ctx, srcPath, destPath)
	if err != nil {
		lg.Error(ctx, "Error to move file", zap.Error(err))
		return err
	}

	err = srv.repo.DeleteFile(ctx, srcPath)
	if err != nil {
		lg.Error(ctx, "Error to delete file while moving", zap.Error(err))
		srv.repo.DeleteFile(ctx, destPath)

		return err
	}

	return nil
}

func (srv *FileService) ListDirectory(ctx context.Context, r *proto.DirectoryRequest) ([]*proto.DirectoryEntry, error) {
	lg := logger.GetLoggerFromContext(ctx)

	lg.Info(ctx, "ListDirectory is in process")
	res, err := srv.repo.ListDir(ctx, r.Path)
	if err != nil {
		lg.Error(ctx, "Error to list dir")
		return nil, err
	}

	var protoRes []*proto.DirectoryEntry
	for _, entry := range res {
		protoRes = append(protoRes, &proto.DirectoryEntry{Name: entry.Name, IsDir: entry.IsDir})
	}

	return protoRes, nil
}
