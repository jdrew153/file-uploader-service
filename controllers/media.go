package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/jdrew153/services"
)

type MediaController struct {
	Service *services.MediaService
}

func NewMediaController(s *services.MediaService) *MediaController {
	return &MediaController{
		Service: s,
	}
}

func (c *MediaController) ServeContent(w http.ResponseWriter, r *http.Request) {
	filePath := r.URL.Path[1:]

	if filePath == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	
	totalPath := fmt.Sprintf("./%s", filePath)

	buffer := c.Service.Get(totalPath)

	if buffer == nil {
		
		c.Service.Set(totalPath)

		log.Println("Serving content from disk")

		c.Service.CalculateCacheWeight()

		http.ServeFile(w, r, "./" + filePath)
		
		return
	}

	log.Println("Serving content from cache")

	http.ServeContent(w, r, filePath, time.Now(), bytes.NewReader(buffer))

}

func (c *MediaController) DownloadContent(w http.ResponseWriter, r *http.Request) {

	log.Println("starting download request...")

	err := r.ParseMultipartForm(2 << 30) // 32 MB max memory limit for parsing the form
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	file, header, err := r.FormFile("file")

	uploadId := r.FormValue("uploadId")

	ext := filepath.Ext(header.Filename)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	defer file.Close()

	err = os.MkdirAll("./media", os.ModePerm)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	out, err := os.Create(fmt.Sprintf("./media/%s", uploadId + ext))

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	defer out.Close()

	bufferSize := 64 * 1024

	buffer := make([]byte, bufferSize)
	var written int64

	for {
		n, err := file.Read(buffer)

		if err != nil && err != io.EOF {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if n == 0 {
			break
		}

		nWritten, err := out.Write(buffer[:n])

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		written += int64(nWritten)

		if written%(10<<20) == 0 {
			log.Printf("Uploaded: %.2f MB", float64(written)/(1<<20))
		}
	}

	log.Printf("File uploaded successfully! Size: %.2f MB", float64(written)/(1<<20))

	w.WriteHeader(http.StatusCreated)
}

type DownloadMediaRequest struct {
	Url string `json:"url"`
	FileName string `json:"fileName"`
}

func (c *MediaController) DownloadMediaFromUrl(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var body DownloadMediaRequest

	err := json.NewDecoder(r.Body).Decode(&body)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = c.Service.MediaFromURLFileWriter(body.Url, body.FileName)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
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

