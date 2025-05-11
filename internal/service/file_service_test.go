package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/JunBSer/FileManager/internal/repository"
	"github.com/JunBSer/FileManager/mocks"
	"github.com/JunBSer/FileManager/pkg/logger"
	"github.com/JunBSer/proto_fileManager/pkg/api/proto"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"io"
	"testing"
)

func TestFileService_Upload(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	lg := logger.New("test_service", "debug")
	ctx = context.WithValue(ctx, logger.Key, lg)

	mockRepo := mocks.NewMockFileRepository(ctrl)
	svc := New(mockRepo)

	t.Run("successful upload", func(t *testing.T) {
		mockStream := mocks.NewMockUploadStream(ctx)
		fileMock := mocks.NewMockFileHandle(ctrl)

		mockRepo.EXPECT().
			GetFileHandle(gomock.Any(), "test.txt", repository.CreateAndW).
			Return(fileMock, nil).
			Times(1)

		mockRepo.EXPECT().
			AppendData(gomock.Any(), fileMock, []byte("chunk1"), int64(0)).
			Return(int64(6), nil).
			Times(1)

		fileMock.EXPECT().
			Close().
			Return(nil).
			Times(1)

		mockStream.On("Recv").Return(&proto.FileChunk{
			FileName: "test.txt",
			Content:  []byte("chunk1"),
		}, nil).Once()
		mockStream.On("Recv").Return((*proto.FileChunk)(nil), io.EOF).Once()

		err := svc.Upload(mockStream)

		assert.NoError(t, err)
		mockStream.AssertExpectations(t)
	})

	t.Run("error opening file", func(t *testing.T) {
		mockStream := mocks.NewMockUploadStream(ctx)

		mockRepo.EXPECT().
			GetFileHandle(gomock.Any(), "error.txt", repository.CreateAndW).
			Return(nil, errors.New("permission denied")).
			Times(1)

		mockStream.On("Recv").Return(&proto.FileChunk{
			FileName: "error.txt",
			Content:  []byte("data"),
		}, nil).Once()

		err := svc.Upload(mockStream)
		assert.Error(t, err)

		mockStream.AssertExpectations(t)
	})
}

func TestFileService_Download(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	lg := logger.New("test_service", "debug")

	ctx := context.WithValue(context.Background(), logger.Key, lg)
	repo := mocks.NewMockFileRepository(ctrl)
	svc := New(repo)

	t.Run("success download", func(t *testing.T) {
		stream := mocks.NewMockDownloadStream(ctx)
		file := mocks.NewMockFileHandle(ctrl)

		repo.EXPECT().GetFileHandle(gomock.Any(), "test.txt", repository.Read).Return(file, nil)
		repo.EXPECT().GetReadSize().Return(int64(4096))
		file.EXPECT().Read(gomock.Any()).Return(1024, io.EOF)
		file.EXPECT().Close().Return(nil)

		err := svc.Download(&proto.FileRequest{FileName: "test.txt"}, stream)
		assert.NoError(t, err)
		stream.AssertExpectations(t)
	})
}

func TestFileService_Delete(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	lg := logger.New("test_service", "debug")
	ctx := context.WithValue(context.Background(), logger.Key, lg)

	repo := mocks.NewMockFileRepository(ctrl)
	svc := New(repo)

	t.Run("success delete", func(t *testing.T) {
		repo.EXPECT().DeleteFile(gomock.Any(), "test.txt").Return(nil)
		err := svc.Delete(ctx, &proto.FileRequest{FileName: "test.txt"})
		assert.NoError(t, err)
	})

	t.Run("not found", func(t *testing.T) {
		repo.EXPECT().DeleteFile(gomock.Any(), "missing.txt").Return(fmt.Errorf("File is not exist"))
		err := svc.Delete(ctx, &proto.FileRequest{FileName: "missing.txt"})
		assert.Error(t, err)
	})
}

func TestFileService_MoveFile(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	lg := logger.New("test_service", "debug")
	ctx := context.WithValue(context.Background(), logger.Key, lg)

	repo := mocks.NewMockFileRepository(ctrl)
	svc := New(repo)

	t.Run("success move", func(t *testing.T) {
		repo.EXPECT().MoveFile(gomock.Any(), "/old.txt", "/new.txt").Return(nil)
		err := svc.MoveFile(ctx, &proto.OperationRequest{
			Source:      "/old.txt",
			Destination: "/new.txt",
		})
		assert.NoError(t, err)
	})
}

func TestFileService_ListDirectory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	lg := logger.New("test_service", "debug")
	ctx := context.WithValue(context.Background(), logger.Key, lg)

	repo := mocks.NewMockFileRepository(ctrl)
	svc := New(repo)

	t.Run("success list", func(t *testing.T) {
		entries := []repository.DirectoryEntry{
			{Name: "file1.txt", IsDir: false},
			{Name: "dir", IsDir: true},
		}

		repo.EXPECT().ListDir(gomock.Any(), "/path").Return(entries, nil)
		result, err := svc.ListDirectory(ctx, &proto.DirectoryRequest{Path: "/path"})
		assert.NoError(t, err)
		assert.Len(t, result, 2)
	})
}

func TestFileService_Append(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	lg := logger.New("test_service", "debug")
	ctx := context.WithValue(context.Background(), logger.Key, lg)

	repo := mocks.NewMockFileRepository(ctrl)
	svc := New(repo)

	t.Run("success append", func(t *testing.T) {
		stream := mocks.NewMockUploadStream(ctx)
		file := mocks.NewMockFileHandle(ctrl)

		repo.EXPECT().GetFileHandle(gomock.Any(), "test.txt", repository.Write).Return(file, nil)
		repo.EXPECT().AppendData(gomock.Any(), file, []byte("data"), int64(0)).Return(int64(4), nil)
		file.EXPECT().Stat().Return(mocks.MockFileInfo{SizeVal: 100}, nil)
		file.EXPECT().Close().Return(nil)

		stream.On("Recv").Return(&proto.FileChunk{FileName: "test.txt", Content: []byte("data")}, nil).Once()
		stream.On("Recv").Return((*proto.FileChunk)(nil), io.EOF).Once()

		err := svc.Append(stream)
		assert.NoError(t, err)
		stream.AssertExpectations(t)
	})
}
