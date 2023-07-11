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
		w.Header().Set("Content-Disposition", "inline")
		if fileBytes, ok := file.(*[]byte); ok {
			GenerateStream(w, r, fileBytes)
			return
		}
	}

	fmt.Println("Cache miss")
	file, err := os.ReadFile(filePath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	fmt.Println("Cache set")
	fileCache.Set(filePath, &file, cache.DefaultExpiration)

	fmt.Println("Cache set time: ", time.Since(now))
	w.Header().Set("Content-Disposition", "inline")
	GenerateStream(w, r, &file)
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

func main() {
	fileCache = cache.New(cache.NoExpiration, cache.NoExpiration)

	http.HandleFunc("/", HandleGetFile)

	http.HandleFunc("/download", func(w http.ResponseWriter, r *http.Request) {
		
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

	if port != nil {
		
		fmt.Printf("Listening on %s...", port)

		log.Fatal(http.ListenAndServe(port, nil))
	}

	log.Println("Listening on :3000...")

	log.Fatal(http.ListenAndServe(":3000", nil))

	
}
