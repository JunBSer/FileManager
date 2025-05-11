package mocks

import (
	"io/fs"
	"time"
)

type MockFileInfo struct {
	NameVal    string
	SizeVal    int64
	ModeVal    fs.FileMode
	ModTimeVal time.Time
	IsDirVal   bool
}

func (m MockFileInfo) Name() string       { return m.NameVal }
func (m MockFileInfo) Size() int64        { return m.SizeVal }
func (m MockFileInfo) Mode() fs.FileMode  { return m.ModeVal }
func (m MockFileInfo) ModTime() time.Time { return m.ModTimeVal }
func (m MockFileInfo) IsDir() bool        { return m.IsDirVal }
func (m MockFileInfo) Sys() interface{}   { return nil }
