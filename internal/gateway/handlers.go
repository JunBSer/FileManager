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
)

type Handler struct {
	gw *Gateway
}

func NewGatewayHandler(gw *Gateway) *Handler {
	return &Handler{gw: gw}
}

func (h Handler) HandleFilePath(requiredPath string, w http.ResponseWriter, r *http.Request) (string, error) {
	fileName := r.URL.Query().Get(requiredPath)

	logger.GetLoggerFromContext(r.Context()).Debug(r.Context(), "Request path: ", zap.String("fileName", fileName))

	if fileName == "" {
		http.Error(w, requiredPath+" parameter is required", http.StatusBadRequest)
		return fileName, myErr.ParamError{ParamName: requiredPath}
	}
	return fileName, nil
}

func (h Handler) ProcessDownloadFile(w http.ResponseWriter, stream proto.FileService_DownloadClient) (int64, error) {
	bufWriter := bufio.NewWriterSize(w, int(h.gw.maxSize)<<10)
	defer bufWriter.Flush()
	cnt := int64(0)
	var n int

	for {
		res, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return cnt, myErr.ReadError{Err: err, Src: "stream"}

		}
		if n, err = bufWriter.Write(res.Content); err != nil {
			return cnt, myErr.WriteError{Err: err, Src: "response"}
		}
		cnt += int64(n)
	}
	return cnt, nil
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

// Upload uploads a file
// @Summary Uploads a file
// @Description Accepts a multipart file upload
// @Tags uploading
// @Accept multipart/form-data
// @Produce text/plain
// @Param file formData file true "File to upload"
// @Param file_path query string true "Path to save the file" example("/documents/report.pdf")
// @Success 200 {string} string "Status: {status}"
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /upload [post]
func (h Handler) Upload(w http.ResponseWriter, r *http.Request) {
	lg := logger.GetLoggerFromContext(r.Context())

	if r.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, h.gw.maxSize<<20)
	defer r.Body.Close()

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

// Download retrieves a file
// @Summary Download a file
// @Description Retrieves a file based on the provided path
// @Tags downloading
// @Accept application/json
// @Produce application/octet-stream
// @Param file_path query string true "Path to the file"
// @Success 200 {file} file "The requested file"
// @Failure 404 {object} models.ErrorResponse "File not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /download [get]
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

	cnt, err := h.ProcessDownloadFile(w, stream)
	if err != nil {
		lg.Error(r.Context(), "Error processing file", zap.Error(err))
		if cnt == 0 {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}

}

// Read retrieves the content of a file
// @Summary Read a file
// @Description Returns the content of a specific file
// @Tags reading
// @Accept application/json
// @Produce application/octet-stream
// @Param file_path query string true "Path to the file"
// @Success 200 {file} file "Content of the file"
// @Failure 404 {object} models.ErrorResponse "File not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /read [get]
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

	cnt, err := h.ProcessDownloadFile(w, stream)
	if err != nil {
		lg.Error(r.Context(), "Error processing file", zap.Error(err))
		if cnt == 0 {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}

// Append appends data to a file
// @Summary Append data to a file
// @Description Appends data to an existing file
// @Tags appending
// @Accept multipart/form-data
// @Produce text/plain
// @Param file formData file true "File to append"
// @Param file_path query string true "Path to the file"
// @Success 200 {string} string "Status: {status}"
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /append [post]
func (h Handler) Append(w http.ResponseWriter, r *http.Request) {
	lg := logger.GetLoggerFromContext(r.Context())

	if r.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, h.gw.maxSize<<20)
	defer r.Body.Close()

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

// Overwrite replaces a file with a new one
// @Summary Overwrite a file
// @Description Replaces an existing file with a new one
// @Tags overwriting
// @Accept multipart/form-data
// @Produce text/plain
// @Param file formData file true "File to upload"
// @Param file_path query string true "Path to save the file"
// @Success 200 {string} string "Status: {status}"
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router/overwrite [put]
func (h Handler) Overwrite(w http.ResponseWriter, r *http.Request) {
	lg := logger.GetLoggerFromContext(r.Context())

	if r.Method != "PUT" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, h.gw.maxSize<<20)
	defer r.Body.Close()

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

// Delete removes a file
// @Summary Delete a file
// @Description Deletes a file based on the provided path
// @Tags deleting
// @Accept application/json
// @Produce text/plain
// @Param file_path query string true "Path to the file"
// @Success 200 {string} string "Status: {status}"
// @Failure 404 {object} models.ErrorResponse "File not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /delete [delete]
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

// MoveFile moves a file from one location to another
// @Summary Move a file
// @Description Moves a file to a new location
// @Tags moving
// @Accept application/json
// @Produce text/plain
// @Param src_path query string true "Source path"
// @Param dst_path query string true "Destination path"
// @Success 200 {string} string "Status: {status}"
// @Failure 404 {object} models.ErrorResponse "File not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /move [post]
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

// ListDir lists files in a directory
// @Summary List directory contents
// @Description Returns a list of files and directories in the specified path
// @Tags listing
// @Accept application/json
// @Produce application/json
// @Param path query string true "Directory path"
// @Success 200 {array} models.FileEntry "List of directory entries"
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /list [get]
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

// Handlers with a bit of large logic :) <3
