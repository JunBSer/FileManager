package mocks

import (
	"context"
	"github.com/JunBSer/proto_fileManager/pkg/api/proto"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/metadata"
)

type MockUploadStream struct {
	mock.Mock
	ctx context.Context
}

func NewMockUploadStream(ctx context.Context) *MockUploadStream {
	return &MockUploadStream{
		ctx: ctx,
	}
}

func (m *MockUploadStream) SendAndClose(resp *proto.StatusResponse) error {
	return m.Called(resp).Error(0)
}

func (m *MockUploadStream) Recv() (*proto.FileChunk, error) {
	args := m.Called()
	return args.Get(0).(*proto.FileChunk), args.Error(1)
}

func (m *MockUploadStream) Context() context.Context {
	return m.ctx
}

func (m *MockUploadStream) SetHeader(md metadata.MD) error {
	return m.Called(md).Error(0)
}

func (m *MockUploadStream) SendHeader(md metadata.MD) error {
	return m.Called(md).Error(0)
}

func (m *MockUploadStream) SetTrailer(md metadata.MD) {
	m.Called(md)
}

func (m *MockUploadStream) SendMsg(msg interface{}) error {
	return m.Called(msg).Error(0)
}

func (m *MockUploadStream) RecvMsg(msg interface{}) error {
	return m.Called(msg).Error(0)
}
