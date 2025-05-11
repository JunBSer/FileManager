package gateway

import (
	"context"
	myErr "github.com/JunBSer/FileManager/internal/gateway/error"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JunBSer/FileManager/pkg/logger"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestHandler_HandleFilePath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	h := &Handler{
		gw: &Gateway{
			maxSize: 1024,
		},
	}

	t.Run("valid path parameter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test?file_path=test.txt", nil)
		req = req.WithContext(context.WithValue(context.Background(), logger.Key, logger.New("gw test", "debug")))
		w := httptest.NewRecorder()

		fileName, err := h.HandleFilePath("file_path", w, req)

		assert.NoError(t, err)
		assert.Equal(t, "test.txt", fileName)
	})

	t.Run("missing path parameter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req = req.WithContext(context.WithValue(context.Background(), logger.Key, logger.New("gw test", "debug")))
		w := httptest.NewRecorder()

		_, err := h.HandleFilePath("file_path", w, req)

		assert.Error(t, err)
		assert.IsType(t, myErr.ParamError{}, err)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
