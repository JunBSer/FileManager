package gateway

import (
	"context"
	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"
	"net/http"
)

func (h Handler) SetupRoutes(ctx context.Context, r *mux.Router) {
	r.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	filesRouter := r.PathPrefix("/api/v1/files").Subrouter()
	filesRouter.HandleFunc("/upload", h.Upload).Methods("POST")
	filesRouter.Handle("/download", http.HandlerFunc(h.Download)).Methods("GET")
	filesRouter.Handle("/read", http.HandlerFunc(h.Read)).Methods("GET")
	filesRouter.HandleFunc("/append", h.Append).Methods("POST")
	filesRouter.HandleFunc("/overwrite", h.Overwrite).Methods("PUT")
	filesRouter.HandleFunc("/delete", h.Delete).Methods("DELETE")
	filesRouter.HandleFunc("/move", h.MoveFile).Methods("POST")
	filesRouter.HandleFunc("/list", h.ListDir).Methods("GET")
}
