package repository

import (
	"context"
	"github.com/JunBSer/FileManager/pkg/logger"
	"os"
	"path/filepath"
)

type FileRepository interface {
	AppendData(path string, data []byte, pos int64) error
	CopyFile(srcPath string, dstPath string) error
	DeleteFile(path string) error
	ReadFile(path string, pos int64) ([]byte, error)
	CreateFile(path string, size int64) error
}
type FileStorageRepo struct {
	storagePath string
	maxSize     int64
}

func (repo *FileStorageRepo) ValidatePath(ctx context.Context, path string) error {
	lg := logger.GetLoggerFromContext(ctx)
	lg.Info(zap)

	path = filepath.Clean(path)

	path = filepath.Join(repo.storagePath, path)

	_, err := os.Stat(path)
	if err != nil {
		lg.Error(ctx, "File storage path does not exist")
		return err
	}

	rel, err := filepath.Rel(repo.storagePath, path)
	if err != nil {
		lg.Error(ctx, "Cannot get relative path")

	}
}

func New(storagePath string, maxSize int64) *FileStorageRepo {
	return &FileStorageRepo{storagePath: storagePath, maxSize: maxSize}
}

func (repo *FileStorageRepo) AppendData(path string, data []byte, pos int64) error {

}
