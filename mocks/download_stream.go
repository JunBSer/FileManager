// mocks/download_stream.go
package mocks

import (
	"context"
	"github.com/JunBSer/proto_fileManager/pkg/api/proto"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/metadata"
)

type MockDownloadStream struct {
	mock.Mock
	ctx context.Context
}

func NewMockDownloadStream(ctx context.Context) *MockDownloadStream {
	return &MockDownloadStream{ctx: ctx}
}

func (m *MockDownloadStream) Send(chunk *proto.FileChunk) error {
	return m.Called(chunk).Error(0)
}

func (m *MockDownloadStream) Context() context.Context {
	return m.ctx
}

func (m *MockDownloadStream) SetHeader(md metadata.MD) error {
	return m.Called(md).Error(0)
}

func (m *MockDownloadStream) SendHeader(md metadata.MD) error {
	return m.Called(md).Error(0)
}

func (m *MockDownloadStream) SetTrailer(md metadata.MD) {
	m.Called(md)
}

func (m *MockDownloadStream) SendMsg(msg interface{}) error {
	return m.Called(msg).Error(0)
}

func (m *MockDownloadStream) RecvMsg(msg interface{}) error {
	return m.Called(msg).Error(0)
}
