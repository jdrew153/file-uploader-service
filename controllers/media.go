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

	uploadId := r.FormValue("uploadId")

	ext := r.FormValue("ext")

	log.Println("ext", ext)
	log.Println("uploadId", uploadId)


	err := os.MkdirAll("./media", os.ModePerm)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	 // Create a temporary file to store the chunks
    tempFile, err := os.CreateTemp("", "temp-*")
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    defer tempFile.Close()

	bufferSize := 1024

	buffer := make([]byte, bufferSize)
	var written int64

	for {
		n, err := r.Body.Read(buffer)

		if err != nil && err != io.EOF {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if n == 0 {
			break
		}

		nWritten, err := tempFile.Write(buffer[:n])

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		written += int64(nWritten)

		if written%(10<<20) == 0 {
			log.Printf("Uploaded: %.2f MB", float64(written)/(1<<20))
		}
	}

	// Rename the temporary file to the final file name using uploadId and ext
    finalFileName := fmt.Sprintf("./media/%s%s", uploadID, ext)
    if err := os.Rename(tempFile.Name(), finalFileName); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
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

