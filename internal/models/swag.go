package models

// ErrorResponse error
type ErrorResponse struct {
	Code    int    `json:"code" example:"400"`
	Message string `json:"message" example:"invalid request parameters"`
}

// FileEntry file list element
type FileEntry struct {
	Name        string `json:"name" example:"report.pdf"`
	IsDirectory bool   `json:"is_directory" example:"false"`
}
