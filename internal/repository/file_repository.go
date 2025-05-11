package repository

import (
	"context"
	"fmt"
	"github.com/JunBSer/FileManager/pkg/logger"
	"go.uber.org/zap"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

const (
	Read       = os.O_RDONLY
	CreateAndW = os.O_CREATE | os.O_WRONLY
	Write      = os.O_RDWR
)

type FileStorageConfig struct {
	StoragePath string `env:"FILE_STORAGE_PATH" envDefault:"/var/tmp/storage"`
	MaxSize     int64  `env:"FILE_MAX_SIZE" envDefault:"10"`
	ReadSize    int64  `env:"FILE_READ_SIZE" envDefault:"2048"`
}

type FileRepository interface {
	GetFileHandle(ctx context.Context, path string, openOption int) (FileHandle, error)
	AppendData(ctx context.Context, file FileHandle, data []byte, pos int64) (int64, error)
	MoveFile(ctx context.Context, dstPath, srcPath string) error
	DeleteFile(ctx context.Context, path string) error
	ReadFile(ctx context.Context, file FileHandle, pos int64) ([]byte, int64, error)
	ListDir(ctx context.Context, path string) ([]DirectoryEntry, error)
	GetReadSize() int64
}

type FileStorageRepo struct {
	storagePath string
	maxSize     int64
	readSize    int64
}

type DirectoryEntry struct {
	Name  string
	IsDir bool
}

type FileHandle interface {
	Seek(offset int64, whence int) (ret int64, err error)
	Write(b []byte) (n int, err error)
	Close() error
	Read(b []byte) (n int, err error)
	Stat() (fs.FileInfo, error)
}

func New(storagePath string, maxSize int64, readSize int64) *FileStorageRepo {
	exePath, err := os.Executable()
	if err != nil {
		return nil
	}

	exeDir := filepath.Dir(exePath)

	fullPath := filepath.Join(exeDir, storagePath)

	if err := os.MkdirAll(fullPath, 0o755); err != nil {
		return nil
	}

	return &FileStorageRepo{storagePath: fullPath, maxSize: maxSize, readSize: readSize}
}

func (repo *FileStorageRepo) GetReadSize() int64 {
	return repo.readSize
}

func (repo *FileStorageRepo) BuildPath(path string) string {
	path = filepath.Join(repo.storagePath, path)
	path = filepath.Clean(path)
	return path
}

func (repo *FileStorageRepo) ValidatePath(ctx context.Context, userPath string) error {
	lg := logger.GetLoggerFromContext(ctx)

	if userPath == repo.storagePath {
		lg.Debug(ctx, "path is empty")
		return fmt.Errorf("path cannot be empty")
	}

	root, err := filepath.Abs(repo.storagePath)
	if err != nil {
		lg.Debug(ctx, "cannot make storagePath absolute", zap.Error(err))
		return fmt.Errorf("internal error")
	}

	abs, err := filepath.Abs(userPath)
	if err != nil {
		lg.Debug(ctx, "cannot make user path absolute", zap.String("userPath", userPath), zap.Error(err))
		return fmt.Errorf("invalid path")
	}
	invalidChars := `[*?"<>|]`
	if runtime.GOOS == "windows" {
		invalidChars = `[*?"<>|:]`
	}

	re := regexp.MustCompile("[" + regexp.QuoteMeta(invalidChars) + "]")
	if match := re.FindString(filepath.Base(abs)); match != "" {
		lg.Error(ctx, "invalid characters in path",
			zap.String("path", abs),
			zap.String("invalidChar", match),
		)
		return fmt.Errorf(
			"path contains invalid character %q",
			match,
		)
	}

	rel, err := filepath.Rel(root, abs)
	if err != nil {
		lg.Error(ctx, "cannot compute relative path", zap.String("abs", abs), zap.Error(err))
		return fmt.Errorf("invalid path")
	}

	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		lg.Debug(ctx, "path traversal detected", zap.String("abs", abs), zap.String("root", root))
		return fmt.Errorf("path %q is outside root directory", userPath)
	}

	return nil
}

func (repo *FileStorageRepo) GetFileHandle(ctx context.Context, path string, openOption int) (FileHandle, error) {
	fullPath := repo.BuildPath(path)
	lg := logger.GetLoggerFromContext(ctx)

	if err := repo.ValidatePath(ctx, fullPath); err != nil {
		return nil, err
	}

	if openOption == Read {
		if _, err := os.Stat(fullPath); err != nil {
			return nil, err
		}
		f, err := os.Open(fullPath)
		if err != nil {
			lg.Error(ctx, "Error opening file for read", zap.String("path", fullPath), zap.Error(err))
			return nil, err
		}
		return f, nil
	}

	file, err := os.OpenFile(fullPath, openOption, 0o777)
	if err != nil {
		if !os.IsNotExist(err) {
			lg.Error(ctx, "Error opening file for write", zap.String("path", fullPath), zap.Error(err))
			return nil, err
		}
		if mkErr := os.MkdirAll(filepath.Dir(fullPath), 0o755); mkErr != nil {
			lg.Error(ctx, "Error creating directory", zap.String("path", fullPath), zap.Error(mkErr))
			return nil, mkErr
		}
		file, err = os.Create(fullPath)
		if err != nil {
			lg.Error(ctx, "Error creating file", zap.String("path", fullPath), zap.Error(err))
			return nil, err
		}
		lg.Info(ctx, "File was created", zap.String("path", fullPath))
	}
	return file, nil
}

func (repo *FileStorageRepo) AppendData(ctx context.Context, file FileHandle, data []byte, pos int64) (int64, error) {
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

	return int64(wCnt), err
}

func (repo *FileStorageRepo) CopyFile(ctx context.Context, srcPath string, dstPath string) error {
	srcFullPath := repo.BuildPath(srcPath)
	dstFullPath := repo.BuildPath(dstPath)

	lg := logger.GetLoggerFromContext(ctx)

	err := repo.ValidatePath(ctx, dstFullPath)
	if err != nil {
		lg.Debug(ctx, "Error to copy file: dstPath path is invalid")
		return err
	}

	err = repo.ValidatePath(ctx, srcFullPath)
	if err != nil {
		lg.Debug(ctx, "Error to copy file: srcPath path is invalid")
		return err
	}

	srcFile, err := repo.GetFileHandle(ctx, srcFullPath, Read)
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

	dstFile, err := repo.GetFileHandle(ctx, dstFullPath, CreateAndW)
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
		lg.Debug(ctx, "Error to delete file: path is invalid")
		return err
	}

	err = os.Remove(fullPath)
	if err != nil {
		lg.Error(ctx, "Error deleting file", zap.String("path", fullPath), zap.Error(err))
	}
	return err
}

func (repo *FileStorageRepo) ReadFile(ctx context.Context, file FileHandle, pos int64) ([]byte, int64, error) {
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

func (repo *FileStorageRepo) ListDir(ctx context.Context, path string) ([]DirectoryEntry, error) {
	lg := logger.GetLoggerFromContext(ctx)

	fullPath := repo.BuildPath(path)
	err := repo.ValidatePath(ctx, fullPath)
	if err != nil {
		lg.Debug(ctx, "Error to list dir: path is invalid")
		return nil, err
	}

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		lg.Error(ctx, "Error listing dir", zap.String("path", fullPath), zap.Error(err))
		return nil, err
	}

	result := make([]DirectoryEntry, 0, len(entries))
	for _, entry := range entries {
		if err != nil {
			continue
		}

		result = append(result, DirectoryEntry{
			Name:  entry.Name(),
			IsDir: entry.IsDir(),
		})
	}

	return result, nil
}

func (repo *FileStorageRepo) MoveFile(ctx context.Context, srcPath, dstPath string) error {
	srcFullPath := repo.BuildPath(srcPath)
	dstFullPath := repo.BuildPath(dstPath)

	lg := logger.GetLoggerFromContext(ctx)

	err := repo.ValidatePath(ctx, dstFullPath)
	if err != nil {
		lg.Debug(ctx, "Error to copy file: dstPath path is invalid")
		return err
	}

	err = repo.ValidatePath(ctx, srcFullPath)
	if err != nil {
		lg.Debug(ctx, "Error to copy file: srcPath path is invalid")
		return err
	}

	err = os.Rename(srcFullPath, dstFullPath)
	if err != nil {
		lg.Debug(ctx, "Error to copy file: can not move file")
		return err
	}
	return nil
}
