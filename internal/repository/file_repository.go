package repository

import (
	"context"
	"errors"
	"fmt"
	"github.com/JunBSer/FileManager/pkg/logger"
	"go.uber.org/zap"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	Open = iota
	CreateAndOpen
)

type FileRepository interface {
	GetFileHandle(ctx context.Context, path string) (*os.File, error)
	AppendData(ctx context.Context, file *os.File, data []byte, pos int64) (int, error)
	CopyFile(ctx context.Context, srcPath string, dstPath string) error
	DeleteFile(ctx context.Context, path string) error
	ReadFile(ctx context.Context, file *os.File, pos int64) ([]byte, int64, error)
}
type FileStorageRepo struct {
	storagePath string
	maxSize     int64
	readSize    int64
}

func New(storagePath string, maxSize int64) *FileStorageRepo {
	return &FileStorageRepo{storagePath: storagePath, maxSize: maxSize}
}

func (repo *FileStorageRepo) BuildPath(path string) string {
	path = filepath.Clean(path)
	path = filepath.Join(repo.storagePath, path)
	return path
}

func (repo *FileStorageRepo) ValidatePath(ctx context.Context, path string) error {

	lg := logger.GetLoggerFromContext(ctx)

	_, err := os.Stat(path)
	if err != nil {
		lg.Error(ctx, "File storage path does not exist", zap.String("path", path))
		return err
	}

	rel, err := filepath.Rel(repo.storagePath, path)
	if err != nil {
		lg.Error(ctx, "Cannot get relative path")
		return err
	}

	if !(!strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel)) {
		lg.Error(ctx, "Path is not valid", zap.String("path", path))
		return errors.New(fmt.Sprintf("Path is not valid, Path: %s", path))
	}

	return nil
}

func (repo *FileStorageRepo) GetFileHandle(ctx context.Context, path string, openOption int) (*os.File, error) {
	fullPath := repo.BuildPath(path)
	lg := logger.GetLoggerFromContext(ctx)

	err := repo.ValidatePath(ctx, fullPath)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(fullPath)
	if err != nil {
		lg.Error(ctx, "Error opening file", zap.String("path", fullPath), zap.Error(err))
		if openOption == Open {
			return nil, err
		}
	} else {
		lg.Info(ctx, "File was opened", zap.String("path", fullPath))
		return file, nil
	}

	file, err = os.Create(fullPath)
	if err != nil {
		lg.Error(ctx, "Error creating file", zap.String("path", fullPath), zap.Error(err))
		return nil, err
	}

	lg.Info(ctx, "File was created", zap.String("path", fullPath))
	return file, nil
}

func (repo *FileStorageRepo) AppendData(ctx context.Context, file *os.File, data []byte, pos int64) (int, error) {
	var err error
	lg := logger.GetLoggerFromContext(ctx)

	_, err = file.Seek(pos, 0)
	if err != nil {
		lg.Error(ctx, "Error seeking file", zap.Int64("position", pos), zap.Error(err))
	}

	wCnt, err := file.Write(data)
	if err != nil {
		lg.Error(ctx, "Error writing to file", zap.Int64("position", pos), zap.Error(err))
		return -1, err
	}

	lg.Info(ctx, fmt.Sprintf("Wrote %d bytes to file", wCnt))

	return wCnt, err
}

func (repo *FileStorageRepo) CopyFile(ctx context.Context, srcPath string, dstPath string) error {
	srcFullPath := repo.BuildPath(srcPath)
	dstFullPath := repo.BuildPath(dstPath)

	lg := logger.GetLoggerFromContext(ctx)

	err := repo.ValidatePath(ctx, dstFullPath)
	if err != nil {
		lg.Debug(ctx, "Error to copy file: dstPath path is inavalid")
		return err
	}

	err = repo.ValidatePath(ctx, srcFullPath)
	if err != nil {
		lg.Debug(ctx, "Error to copy file: srcPath path is inavalid")
		return err
	}

	srcFile, err := repo.GetFileHandle(ctx, srcFullPath, Open)
	if err != nil {
		lg.Debug(ctx, "Error to copy file: srcFile is not exist")
		return err
	}

	defer func() {
		err = srcFile.Close()
		if err != nil {
			lg.Error(ctx, "Error closing file", zap.String("path", srcFullPath), zap.Error(err))
		}
	}()

	dstFile, err := repo.GetFileHandle(ctx, dstFullPath, CreateAndOpen)
	if err != nil {
		lg.Debug(ctx, "Error to copy file: can not create and write file")
	}

	func() {
		err = dstFile.Close()
		if err != nil {
			lg.Error(ctx, "Error closing file", zap.String("path", srcFullPath), zap.Error(err))
		}
	}()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		lg.Error(ctx, "Error to copy file: can not copy file", zap.Error(err))
	}

	return err
}

func (repo *FileStorageRepo) DeleteFile(ctx context.Context, path string) error {
	lg := logger.GetLoggerFromContext(ctx)

	fullPath := repo.BuildPath(path)

	err := repo.ValidatePath(ctx, fullPath)
	if err != nil {
		lg.Debug(ctx, "Error to delete file: path is inavalid")
		return err
	}

	err = os.Remove(fullPath)
	if err != nil {
		lg.Error(ctx, "Error deleting file", zap.String("path", fullPath), zap.Error(err))
	}
	return err
}

func (repo *FileStorageRepo) ReadFile(ctx context.Context, file *os.File, pos int64) ([]byte, int64, error) {
	lg := logger.GetLoggerFromContext(ctx)

	_, err := file.Seek(pos, 0)
	if err != nil {
		lg.Error(ctx, "Error seeking file", zap.Int64("position", pos), zap.Error(err))
	}

	buf := make([]byte, repo.readSize)
	bRead, err := file.Read(buf)

	if err != nil {
		lg.Error(ctx, "Error reading file", zap.Int64("position", pos), zap.Error(err))
		return nil, 0, err
	}

	lg.Debug(ctx, fmt.Sprintf("Read %d bytes from file", bRead))

	return buf, int64(bRead), err
}
