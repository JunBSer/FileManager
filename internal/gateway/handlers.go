package gateway

import (
	"bufio"
	"context"
	"encoding/json"
	myErr "github.com/JunBSer/FileManager/internal/gateway/error"
	"github.com/JunBSer/FileManager/pkg/logger"
	"github.com/JunBSer/proto_fileManager/pkg/api/proto"
	"go.uber.org/zap"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
)

type Handler struct {
	gw Gateway
}

func NewGatewayHandler(gw Gateway) *Handler {
	return &Handler{gw: gw}
}

func (h Handler) HandleFilePath(requiredPath string, w http.ResponseWriter, r *http.Request) (string, error) {
	fileName := r.URL.Query().Get(requiredPath)
	fileName = url.QueryEscape(fileName)
	logger.GetLoggerFromContext(r.Context()).Debug(r.Context(), "Request path: ", zap.String("fileName", fileName))

	if fileName == "" {
		http.Error(w, requiredPath+" parameter is required", http.StatusBadRequest)
		return fileName, myErr.ParamError{ParamName: requiredPath}
	}
	return fileName, nil
}

func (h Handler) ProcessDownloadFile(w http.ResponseWriter, stream proto.FileService_DownloadClient) error {
	bufWriter := bufio.NewWriterSize(w, int(h.gw.maxSize)<<10)
	defer bufWriter.Flush()

	for {
		res, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return myErr.ReadError{Err: err, Src: "stream"}

		}
		if _, err := bufWriter.Write(res.Content); err != nil {
			return myErr.WriteError{Err: err, Src: "response"}
		}
	}
	return nil
}

func (h Handler) ProcessUploadFile(fileName string, file multipart.File, stream proto.FileService_UploadClient) error {
	buf := make([]byte, h.gw.maxSize<<10)

	for {
		bytesRead, err := file.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		req := proto.FileChunk{FileName: fileName, Content: buf[:bytesRead]}
		if err := stream.Send(&req); err != nil {
			return err
		}
	}

	return nil
}

func (h Handler) Upload(w http.ResponseWriter, r *http.Request) {
	lg := logger.GetLoggerFromContext(r.Context())

	if r.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseMultipartForm(h.gw.maxSize << 20); err != nil {
		http.Error(w, "Invalid file upload: "+err.Error(), http.StatusBadRequest)
		return
	}

	fileName, err := h.HandleFilePath("file_path", w, r)
	if err != nil {
		lg.Error(r.Context(), "Error handling file path", zap.String("fileName", fileName))
		return
	}

	file, _, err := r.FormFile("file")
	defer func() {
		err = file.Close()
		if err != nil {
			log.Println(err)
		}
	}()

	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		lg.Error(r.Context(), "Error reading file", zap.Error(err))
		return
	}

	stream, err := h.gw.client.Cl.Upload(r.Context())
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		lg.Error(r.Context(), "Error getting stream", zap.Error(err))
		return
	}
	defer stream.CloseSend()

	err = h.ProcessUploadFile(fileName, file, stream)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		lg.Error(r.Context(), "Error processing file", zap.Error(err))
		return
	}

	res, err := stream.CloseAndRecv()
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		lg.Error(r.Context(), "Error receiving response", zap.Error(err))
		return
	}

	_, err = w.Write([]byte("Status: " + res.GetStatus().String()))
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}

}

// Download Add header middleware
func (h Handler) Download(w http.ResponseWriter, r *http.Request) {
	lg := logger.GetLoggerFromContext(r.Context())
	if r.Method != "GET" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	fileName, err := h.HandleFilePath("file_path", w, r)
	if err != nil {
		lg.Error(r.Context(), "Error handling file path", zap.String("fileName", fileName))
		return
	}

	stream, err := h.gw.client.Cl.Download(r.Context(), &proto.FileRequest{FileName: fileName})
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		lg.Error(r.Context(), "Error getting stream", zap.Error(err))
		return
	}

	defer stream.CloseSend()
	w.Header().Set("Content-Disposition", "attachment; filename*=UTF-8''"+fileName)

	err = h.ProcessDownloadFile(w, stream)
	if err != nil {
		lg.Error(r.Context(), "Error processing file", zap.Error(err))
	}

}

func (h Handler) Read(w http.ResponseWriter, r *http.Request) {
	lg := logger.GetLoggerFromContext(r.Context())
	if r.Method != "GET" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	fileName, err := h.HandleFilePath("file_path", w, r)
	if err != nil {
		lg.Error(r.Context(), "Error handling file path", zap.String("fileName", fileName))
		return
	}

	stream, err := h.gw.client.Cl.Read(r.Context(), &proto.FileRequest{FileName: fileName})
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		lg.Error(r.Context(), "Error getting stream", zap.Error(err))
		return
	}

	defer stream.CloseSend()

	err = h.ProcessDownloadFile(w, stream)
	if err != nil {
		lg.Error(r.Context(), "Error processing file", zap.Error(err))
	}
}

func (h Handler) Append(w http.ResponseWriter, r *http.Request) {
	lg := logger.GetLoggerFromContext(r.Context())

	if r.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseMultipartForm(h.gw.maxSize << 20); err != nil {
		http.Error(w, "Invalid file upload: "+err.Error(), http.StatusBadRequest)
		return
	}

	fileName, err := h.HandleFilePath("file_path", w, r)
	if err != nil {
		lg.Error(r.Context(), "Error handling file path", zap.String("fileName", fileName))
		return
	}

	file, _, err := r.FormFile("file")
	defer func() {
		err = file.Close()
		if err != nil {
			log.Println(err)
		}
	}()

	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		lg.Error(r.Context(), "Error reading file", zap.Error(err))
		return
	}

	stream, err := h.gw.client.Cl.Append(r.Context())
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		lg.Error(r.Context(), "Error getting stream", zap.Error(err))
		return
	}
	defer stream.CloseSend()

	err = h.ProcessUploadFile(fileName, file, stream)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		lg.Error(r.Context(), "Error processing file", zap.Error(err))
		return
	}

	res, err := stream.CloseAndRecv()
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		lg.Error(r.Context(), "Error receiving response and closing stream", zap.Error(err))
		return
	}

	_, err = w.Write([]byte("Status: " + res.GetStatus().String()))
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (h Handler) Overwrite(w http.ResponseWriter, r *http.Request) {
	lg := logger.GetLoggerFromContext(r.Context())

	if r.Method != "PUT" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseMultipartForm(h.gw.maxSize << 20); err != nil {
		http.Error(w, "Invalid file upload: "+err.Error(), http.StatusBadRequest)
		return
	}

	fileName, err := h.HandleFilePath("file_path", w, r)
	if err != nil {
		lg.Error(r.Context(), "Error handling file path", zap.String("fileName", fileName))
		return
	}

	file, _, err := r.FormFile("file")
	defer func() {
		err = file.Close()
		if err != nil {
			log.Println(err)
		}
	}()

	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		lg.Error(r.Context(), "Error reading file", zap.Error(err))
		return
	}

	stream, err := h.gw.client.Cl.OverwriteFile(r.Context())
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		lg.Error(r.Context(), "Error getting stream", zap.Error(err))
		return
	}
	defer stream.CloseSend()

	err = h.ProcessUploadFile(fileName, file, stream)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		lg.Error(r.Context(), "Error processing file", zap.Error(err))
		return
	}

	res, err := stream.CloseAndRecv()
	if err != nil {
		lg.Error(r.Context(), "Error closing stream", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	_, err = w.Write([]byte("Status: " + res.GetStatus().String()))
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (h Handler) Delete(w http.ResponseWriter, r *http.Request) {
	lg := logger.GetLoggerFromContext(r.Context())
	if r.Method != "DELETE" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	fileName, err := h.HandleFilePath("file_path", w, r)
	if err != nil {
		lg.Debug(r.Context(), "Error handling file path", zap.String("fileName", fileName))
		return
	}

	res, err := h.gw.client.Cl.Delete(r.Context(), &proto.FileRequest{FileName: fileName})
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		lg.Error(r.Context(), "Error getting stream", zap.Error(err))
		return
	}

	_, err = w.Write([]byte("Status: " + res.GetStatus().String()))
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func (h Handler) MoveFile(w http.ResponseWriter, r *http.Request) {
	lg := logger.GetLoggerFromContext(r.Context())
	if r.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	dstFileName, err := h.HandleFilePath("dst_path", w, r)
	if err != nil {
		lg.Debug(r.Context(), "Error handling file path", zap.String("fileName", dstFileName))
		return
	}

	srcFileName, err := h.HandleFilePath("src_path", w, r)
	if err != nil {
		lg.Debug(r.Context(), "Error handling file path", zap.String("fileName", srcFileName))
		return
	}

	res, err := h.gw.client.Cl.MoveFile(r.Context(), &proto.OperationRequest{Destination: dstFileName, Source: srcFileName})
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		lg.Debug(r.Context(), "Error moving file", zap.Error(err))
		return
	}

	_, err = w.Write([]byte("Status: " + res.GetStatus().String()))
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}

}

func (h Handler) EncodeDirectoryResponse(
	w http.ResponseWriter,
	entries []*proto.DirectoryEntry,
	dirPath string,
	ctx context.Context,
) {
	lg := logger.GetLoggerFromContext(ctx)
	type Entry struct {
		Name        string `json:"name"`
		IsDirectory bool   `json:"is_directory"`
	}

	jsonEntries := make([]Entry, 0, len(entries))
	for _, e := range entries {
		jsonEntries = append(jsonEntries, Entry{
			Name:        e.Name,
			IsDirectory: e.IsDir,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(jsonEntries); err != nil {
		lg.Error(ctx, "Error encoding JSON response",
			zap.String("path", dirPath),
			zap.Error(err))
	}
}

func (h Handler) ListDir(w http.ResponseWriter, r *http.Request) {
	lg := logger.GetLoggerFromContext(r.Context())

	if r.Method != "GET" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	dirPath, err := h.HandleFilePath("path", w, r)
	if err != nil {
		lg.Debug(r.Context(), "Error handling file path", zap.String("fileName", dirPath))
		return
	}

	res, err := h.gw.client.Cl.ListDirectory(r.Context(),
		&proto.DirectoryRequest{Path: dirPath})
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
		lg.Error(r.Context(), "Error getting directory listing",
			zap.String("path", dirPath),
			zap.Error(err))
		return
	}

	h.EncodeDirectoryResponse(w, res.Entries, dirPath, r.Context())
}
