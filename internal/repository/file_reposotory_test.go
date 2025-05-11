package repository

import (
	"context"
	"github.com/JunBSer/FileManager/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const relPath = "test"

func CreateTempDir(t *testing.T) string {
	exePath, err := os.Executable()
	require.NoError(t, err)

	exeDir := filepath.Dir(exePath)

	fullPath := filepath.Join(exeDir, relPath)
	require.NoError(t, os.MkdirAll(fullPath, 0o755))

	return fullPath
}

func TestNewFileStorageRepo(t *testing.T) {
	fullPath := CreateTempDir(t)

	repo := New(relPath, 1024*1024, 2048)

	require.Equal(t, repo.storagePath, fullPath)

	defer os.RemoveAll(fullPath)
}

func TestFileStorageRepo_BuildPath(t *testing.T) {
	fullPath := CreateTempDir(t)
	repo := New(relPath, 1024*1024, 2048)

	tests := []struct {
		testName string
		path     string
	}{
		{"Clean path", "coolDir/das"},
		{"Bad test 1", "../asdas"},
		{"Bad test 2", "/asdas/../../.."},
		{"Bad test 3", ""},
		{"Bad test 4", "dasdas_das/ad|adawd\\"},
	}

	t.Run("Build path test", func(t *testing.T) {
		for _, test := range tests {
			resPath := repo.BuildPath(test.path)

			require.Equal(t, strings.Contains(resPath, ".."), false)
		}
	})

	defer os.RemoveAll(fullPath)
}

func TestFileStorageRepo_ValidatePath(t *testing.T) {
	fullPath := CreateTempDir(t)
	repo := New(relPath, 1024*1024, 2048)
	tests := []struct {
		testName string
		path     string
		isErr    bool
	}{
		{"Clean path", "coolDir/das", false},
		{"Bad test 1", "../asdas", true},
		{"Bad test 2", "/asdas/../../..", true},
		{"Bad test 3", "", true},
		{"Bad test 4", "asd  asda", false},
		{"Bad test 5", "/asdas\ndsd", false},
		{"Valid nested path", "dir/subdir/file.txt", false},
		{"Path traversal", "../../etc/passwd", true},
	}

	ctx := context.Background()
	lg := logger.New("test", "debug")
	ctx = context.WithValue(ctx, logger.Key, lg)

	t.Run("Validate path test", func(t *testing.T) {
		for _, test := range tests {
			fullPath := repo.BuildPath(test.path)

			err := repo.ValidatePath(ctx, fullPath)
			if test.isErr {
				require.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		}
	})

	defer os.RemoveAll(fullPath)
}

func TestFileStorageRepo_GetFileHandle(t *testing.T) {
	fullPath := CreateTempDir(t)
	repo := New(relPath, 1024*1024, 2048)

	ctx := context.Background()
	lg := logger.New("test", "debug")
	ctx = context.WithValue(ctx, logger.Key, lg)

	t.Run("Create new file", func(t *testing.T) {
		f, err := repo.GetFileHandle(ctx, "newfile.txt", CreateAndW)
		require.NoError(t, err)
		defer f.Close()

		info, err := f.Stat()
		assert.NoError(t, err)
		assert.Equal(t, "newfile.txt", info.Name())
	})

	t.Run("Open existing for read", func(t *testing.T) {

		f, _ := repo.GetFileHandle(ctx, "existing.txt", CreateAndW)
		f.Close()

		f, err := repo.GetFileHandle(ctx, "existing.txt", Read)
		require.NoError(t, err)
		defer f.Close()
	})

	t.Run("Fail on invalid path", func(t *testing.T) {
		_, err := repo.GetFileHandle(ctx, "../invalid.txt", CreateAndW)
		assert.Error(t, err)
	})

	defer os.RemoveAll(fullPath)
}

func TestFileStorageRepo_AppendData(t *testing.T) {
	fullPath := CreateTempDir(t)
	repo := New(relPath, 1024*1024, 2048)

	ctx := context.Background()
	lg := logger.New("test", "debug")
	ctx = context.WithValue(ctx, logger.Key, lg)

	t.Run("Append to new file", func(t *testing.T) {
		f, err := repo.GetFileHandle(ctx, "append_test.txt", CreateAndW)
		require.NoError(t, err)
		defer f.Close()

		data1 := []byte("hello ")
		written, err := repo.AppendData(ctx, f, data1, 0)
		require.NoError(t, err)
		require.Equal(t, int64(len(data1)), written)

		data2 := []byte("world")
		written, err = repo.AppendData(ctx, f, data2, int64(len(data1)))
		require.NoError(t, err)
		require.Equal(t, int64(len(data2)), written)

		f.Close()
		content, err := os.ReadFile(filepath.Join(fullPath, "append_test.txt"))
		require.NoError(t, err)
		require.Equal(t, "hello world", string(content))
	})

	t.Run("Append to closed file should fail", func(t *testing.T) {
		f, err := repo.GetFileHandle(ctx, "closed_file.txt", CreateAndW)
		require.NoError(t, err)
		f.Close()

		_, err = repo.AppendData(ctx, f, []byte("test"), 0)
		require.Error(t, err)
	})

	defer os.RemoveAll(fullPath)
}

func TestFileStorageRepo_ReadFile(t *testing.T) {
	fullPath := CreateTempDir(t)
	repo := New(relPath, 1024*1024, 2048)

	ctx := context.Background()
	lg := logger.New("test", "debug")
	ctx = context.WithValue(ctx, logger.Key, lg)

	testData := []byte("this is test data for reading")
	testFile := "read_test.txt"

	f, err := repo.GetFileHandle(ctx, testFile, CreateAndW)
	require.NoError(t, err)
	_, err = repo.AppendData(ctx, f, testData, 0)
	require.NoError(t, err)
	f.Close()

	t.Run("Read full content", func(t *testing.T) {
		f, err := repo.GetFileHandle(ctx, testFile, Read)
		require.NoError(t, err)
		defer f.Close()

		buf, n, err := repo.ReadFile(ctx, f, 0)
		require.NoError(t, err)
		require.Equal(t, int64(len(testData)), n)
		require.Equal(t, testData, buf[:n])
	})

	t.Run("Read with offset", func(t *testing.T) {
		f, err := repo.GetFileHandle(ctx, testFile, Read)
		require.NoError(t, err)
		defer f.Close()

		offset := int64(5)
		expected := testData[offset:]

		buf, n, err := repo.ReadFile(ctx, f, offset)
		require.NoError(t, err)
		require.Equal(t, int64(len(expected)), n)
		require.Equal(t, expected, buf[:n])
	})

	t.Run("Read beyond file end", func(t *testing.T) {
		f, err := repo.GetFileHandle(ctx, testFile, Read)
		require.NoError(t, err)
		defer f.Close()

		_, _, err = repo.ReadFile(ctx, f, int64(len(testData)+10))
		require.Error(t, err)
		require.Equal(t, io.EOF, err)
	})

	defer os.RemoveAll(fullPath)
}

func TestFileStorageRepo_DeleteFile(t *testing.T) {
	fullPath := CreateTempDir(t)
	repo := New(relPath, 1024*1024, 2048)
	ctx := context.Background()

	lg := logger.New("test", "debug")
	ctx = context.WithValue(ctx, logger.Key, lg)

	delFileName := "delete_file.txt"

	func() {
		f, err := os.Create(filepath.Join(fullPath, delFileName))
		require.NoError(t, err)
		defer f.Close()
	}()

	t.Run("Deletion of the empty file", func(t *testing.T) {
		err := repo.DeleteFile(ctx, delFileName)
		require.NoError(t, err)

		_, err = os.Stat(filepath.Join(fullPath, delFileName))
		require.True(t, os.IsNotExist(err))
	})

	t.Run("Deletion of non existing file", func(t *testing.T) {
		err := repo.DeleteFile(ctx, "non_existing_file.txt")
		require.Error(t, err)
		require.True(t, os.IsNotExist(err))
	})

	t.Run("Deletion of existing non-empty file", func(t *testing.T) {
		fileHandle, err := repo.GetFileHandle(ctx, delFileName, CreateAndW)
		require.NoError(t, err)

		_, err = repo.AppendData(ctx, fileHandle, []byte("Hello World!"), 0)
		require.NoError(t, err)
		fileHandle.Close()

		err = repo.DeleteFile(ctx, delFileName)
		require.NoError(t, err)

		_, err = os.Stat(filepath.Join(fullPath, delFileName))
		require.True(t, os.IsNotExist(err))
	})

	defer os.RemoveAll(fullPath)
}

func TestFileStorageRepo_ListDir(t *testing.T) {
	fullPath := CreateTempDir(t)
	repo := New(relPath, 1024*1024, 2048)

	ctx := context.Background()
	lg := logger.New("test", "debug")
	ctx = context.WithValue(ctx, logger.Key, lg)

	testDir := "test_list_dir"
	testFiles := []string{"file1.txt", "file2.jpg"}
	testSubdir := "subdir"

	t.Run("Prepare test data", func(t *testing.T) {
		require.NoError(t, os.MkdirAll(filepath.Join(fullPath, testDir), 0755))

		for _, f := range testFiles {
			require.NoError(t, os.WriteFile(
				filepath.Join(fullPath, testDir, f),
				[]byte("content"),
				0644,
			))
		}

		require.NoError(t, os.MkdirAll(
			filepath.Join(fullPath, testDir, testSubdir),
			0755,
		))
	})

	t.Run("List directory with content", func(t *testing.T) {
		entries, err := repo.ListDir(ctx, testDir)
		require.NoError(t, err)
		assert.Len(t, entries, 3)

		expected := map[string]bool{
			"file1.txt": false,
			"file2.jpg": false,
			"subdir":    true,
		}

		for _, entry := range entries {
			isDir, exists := expected[entry.Name]
			require.True(t, exists, "Unexpected entry: %s", entry.Name)
			assert.Equal(t, isDir, entry.IsDir)
			delete(expected, entry.Name)
		}
		assert.Empty(t, expected)
	})

	t.Run("List empty directory", func(t *testing.T) {
		emptyDir := "empty_dir"
		require.NoError(t, os.Mkdir(filepath.Join(fullPath, emptyDir), 0755))

		entries, err := repo.ListDir(ctx, emptyDir)
		require.NoError(t, err)
		assert.Empty(t, entries)
	})

	t.Run("List non-existent directory", func(t *testing.T) {
		_, err := repo.ListDir(ctx, "non_existent_dir")
		require.Error(t, err)
		assert.True(t, os.IsNotExist(err))
	})

	defer os.RemoveAll(fullPath)
}

func TestFileStorageRepo_MoveFile(t *testing.T) {
	fullPath := CreateTempDir(t)
	repo := New(relPath, 1024*1024, 2048)

	ctx := context.Background()
	lg := logger.New("test", "debug")
	ctx = context.WithValue(ctx, logger.Key, lg)

	t.Run("Move existing file", func(t *testing.T) {
		src := "source.txt"
		dst := "dest.txt"
		content := []byte("test content")

		require.NoError(t, os.WriteFile(
			filepath.Join(fullPath, src),
			content,
			0644,
		))

		err := repo.MoveFile(ctx, src, dst)
		require.NoError(t, err)

		_, err = os.Stat(filepath.Join(fullPath, src))
		assert.True(t, os.IsNotExist(err))

		data, err := os.ReadFile(filepath.Join(fullPath, dst))
		require.NoError(t, err)
		assert.Equal(t, content, data)
	})

	t.Run("Move to nested directory", func(t *testing.T) {
		src := "file.txt"
		dst := "nested/destination.txt"

		require.NoError(t, os.WriteFile(
			filepath.Join(fullPath, src),
			[]byte("data"),
			0644,
		))

		require.NoError(t, os.Mkdir(filepath.Join(fullPath, "nested"), 0755))

		err := repo.MoveFile(ctx, src, dst)
		require.NoError(t, err)

		_, err = os.Stat(filepath.Join(fullPath, "nested"))
		require.NoError(t, err)

		_, err = os.Stat(filepath.Join(fullPath, dst))
		require.NoError(t, err)
	})

	t.Run("Move non-existent file", func(t *testing.T) {
		err := repo.MoveFile(ctx, "missing.txt", "new.txt")
		require.Error(t, err)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("Move to invalid path", func(t *testing.T) {
		src := "valid.txt"
		dst := "../../etc/passwd"

		require.NoError(t, os.WriteFile(
			filepath.Join(fullPath, src),
			[]byte("data"),
			0644,
		))

		err := repo.MoveFile(ctx, src, dst)
		require.Error(t, err)
	})

	defer os.RemoveAll(fullPath)
}
