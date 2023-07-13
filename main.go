package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/jpeg"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
	"path/filepath"

	"github.com/nfnt/resize"
	cache "github.com/patrickmn/go-cache"
)

var fileCache *cache.Cache

type DownloadRequest struct {
	URL      string `json:"url"`
	FileName string `json:"fileName"`
}

func DownloadFile(URL, fileName string) error {
	response, err := http.Get(URL)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", response.Status)
	}

	file, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, response.Body)
	if err != nil {
		return err
	}

	ResizeFile(fileName, 1280, 720)

	return nil
}

func HandleGetFile(w http.ResponseWriter, r *http.Request) {
	filePath := r.URL.Path[1:]

	now := time.Now()

	if file, found := fileCache.Get(filePath); found {
		fmt.Println("Cache hit")
		fmt.Println("Cache hit time: ", time.Since(now))
		SetCacheHeaders(w, r, filePath)
		if fileBytes, ok := file.(*[]byte); ok {
			GenerateStream(w, r, fileBytes)
			return
		}
	}

	fmt.Println("Cache miss")
	file, err := ReadFile(filePath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	fmt.Println("Cache set")
	fileCache.Set(filePath, &file, cache.DefaultExpiration)
	SetCacheHeaders(w, r, filePath)

	fmt.Println("Cache set time: ", time.Since(now))
	GenerateStream(w, r, &file)
}

func ReadFile(filePath string) ([]byte, error) {
	fileBytes := make(chan []byte)

	
	go func() {
		readBytes, err := os.ReadFile(filePath)
		fmt.Println("File size after read: ", len(readBytes))
		if err != nil {
			fmt.Println("Error reading file: ", err)
		}
		fileBytes <- readBytes
		close(fileBytes)
	}()

	fileBytesResult := <-fileBytes
	
	
	return fileBytesResult, nil
}

func GenerateStream(w http.ResponseWriter, r *http.Request, fileBytes *[]byte) {
	// Set the appropriate headers for the video file
	w.Header().Set("Accept-Ranges", "bytes")

	// Serve the file using http.ServeContent to handle range requests and send in chunks
	http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(*fileBytes))
}

func ResizeFile(fileName string, width, height int) error {
	if !(strings.Contains(fileName, ".jpg") || strings.Contains(fileName, ".jpeg")) {
		fmt.Println("Not a jpg file")
		return nil
	} else {
		file, err := os.Open(fileName)

		if err != nil {
			return err
		}

		defer file.Close()

		img, err := jpeg.Decode(file)

		if err != nil {
			return err
		}

		m := resize.Resize(uint(width), uint(height), img, resize.Lanczos3)
		basePath := strings.Split(".", fileName)[0]
		newFileName := fmt.Sprintf("%s_thumb_%dx%d.jpeg", basePath, width, height)

		out, err := os.Create(newFileName)

		if err != nil {
			return err
		}
		defer out.Close()

		jpeg.Encode(out, m, nil)

		return nil
	}

}

func SetCacheHeaders(w http.ResponseWriter, r *http.Request, filePath string) {
	ext := filepath.Ext(filePath)

	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif":
		w.Header().Set("Cache-Control", "public max-age=86400")
	case ".mp4", ".mov", ".avi", ".webm":
		w.Header().Set("Cache-Control", "public max-age=604800")
	default:
		w.Header().Set("Cache-Control", "public max-age=3600")
	}
}


func main() {
	fileCache = cache.New(cache.NoExpiration, cache.NoExpiration)
	
        http.Handle("/", http.FileServer(http.Dir("./")))
	//http.HandleFunc("/", HandleGetFile)

	http.HandleFunc("/download", func(w http.ResponseWriter, r *http.Request) {

			// Enable CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// Handle preflight OPTIONS request
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		var body DownloadRequest

		fmt.Println("hello from download endpoint")
		
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		err := DownloadFile(body.URL, body.FileName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		fmt.Printf("Downloaded: %s from %s\n", body.FileName, body.URL)

		w.WriteHeader(http.StatusCreated)
	})

	port := os.Getenv("PORT")

	if port != "" {
		
		fmt.Printf("Listening on %s...", port)

		log.Fatal(http.ListenAndServe("0.0.0.0:" + port, nil))
	}

	log.Println("Listening on :3000...")

	log.Fatal(http.ListenAndServe("0.0.0.0" + ":3000", nil))

	
}
