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
	r.ParseMultipartForm(10000 << 20)

	file, header, err := r.FormFile("file")

	uploadId := r.FormValue("uploadId")

	ext := filepath.Ext(header.Filename)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	defer file.Close()

	out, err := os.Create(fmt.Sprintf("./media/%s", uploadId + ext))

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	defer out.Close()

	_, err = io.Copy(out, file)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// //
	// // Remove temp file after 10 seconds - Real image to be uploaded later...
	// //

	// go func(filePath string) {
	// 	time.Sleep(60 * 2 * time.Second)
	// 	err := os.Remove(fmt.Sprintf("./media/%s", header.Filename))
		
	// 	if err != nil {
	// 		log.Println("Error removing file: ", err)
	// 		return
	// 	}
	// 	log.Println("Removed temp file: ", filePath)
	// }(header.Filename)

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

