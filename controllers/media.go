package controllers

import (
	"net/http"
	"path/filepath"
)

type MediaController struct {
}

func NewMediaController() *MediaController {
	return &MediaController{}
}

func (c *MediaController) ServeContent(w http.ResponseWriter, r *http.Request) {
	filePath := r.URL.Path[1:]
	SetCacheHeaders(w, r, filePath)
	http.ServeFile(w, r, "./"+filePath)
}


func SetCacheHeaders(w http.ResponseWriter, r *http.Request, filePath string) {
	ext := filepath.Ext(filePath)

	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif":
		w.Header().Set("Cache-Control", "public, max-age=86400")
	case ".mp4", ".mov", ".avi", ".webm":
		w.Header().Set("Cache-Control", "public, max-age=604800")
	default:
		w.Header().Set("Cache-Control", "public, max-age=3600")
	}
}