package gateway

import "github.com/gorilla/mux"

func (h *Handler) SetupRoutes(r *mux.Router) {
	filesRouter := r.PathPrefix("/api/v1/files").Subrouter()

	filesRouter.HandleFunc("/upload", h.Upload).Methods("POST")
	filesRouter.HandleFunc("/download", h.Download).Methods("GET")
	filesRouter.HandleFunc("/read", h.Read).Methods("GET")
	filesRouter.HandleFunc("/append", h.Append).Methods("POST")
	filesRouter.HandleFunc("/overwrite", h.Overwrite).Methods("PUT")
	filesRouter.HandleFunc("/delete", h.Delete).Methods("DELETE")
	filesRouter.HandleFunc("/move", h.MoveFile).Methods("POST")
}
