package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
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

	log.Println(filePath)

	if filePath == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if strings.Contains(filePath, ".mp4") || strings.Contains(filePath, ".m3u8") {

		http.ServeFile(w,r, "./" + filePath)
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
	log.SetOutput(os.Stderr)
	log.Println("Download request received")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	defer cancel()

	err := r.ParseMultipartForm(32 << 20)

	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	apiKey := r.MultipartForm.Value["apiKey"][0]

	var done = make(chan bool)

	go func(r *http.Request) {

		authModel, err := c.Service.APIKeyCheck(apiKey)

		if err != nil {
			log.Println(err)
		}
		
		log.Println("API Key: ", apiKey)

		ext := r.URL.Query().Get("ext")
		currChunk, _ := strconv.Atoi(r.URL.Query().Get("currChunk"))
		totalChunks, _ := strconv.Atoi(r.URL.Query().Get("totalChunks"))
		baseFileName := r.URL.Query().Get("fileName")
		fileId := r.URL.Query().Get("fileId")
		dirtyRemote := r.URL.Query().Get("remote")

		remote := strings.Trim(dirtyRemote, "\"")

		log.Println("Remote:", remote)

		sentFileSize, _ := strconv.Atoi(r.URL.Query().Get("totalSize"))

		// header needs to be random
		file, header, err := r.FormFile("file")

		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		defer file.Close()

		os.Mkdir("./media", os.ModePerm)

		dir := fmt.Sprintf("./%s", fileId)
		os.MkdirAll(dir, os.ModePerm)

		out, err := os.Create(fmt.Sprintf("%s/%s", dir, strconv.Itoa(currChunk)+"_"+header.Filename+"."+ext))

		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		defer out.Close()

		_, err = io.CopyN(out, file, r.ContentLength)

		if err != nil && err != io.EOF {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if currChunk < totalChunks {

			w.WriteHeader(http.StatusPartialContent)
			progress := strconv.Itoa(int(r.ContentLength / int64(sentFileSize)))
			w.Write([]byte(progress))

		} else {
			// Create the final file
			finalFileName := fmt.Sprintf("%s.%s", baseFileName, ext)

			finalFile, err := os.Create(fmt.Sprintf("./media/%s", finalFileName))

			if err != nil {
				log.Println(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer finalFile.Close()

			files, err := os.ReadDir(dir)

			if err != nil {
				log.Println(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			totalSize := int64(0)

			sorted := SortFiles(files)

			for _, entry := range sorted {
				tempFileName := fmt.Sprintf("%s/%s", dir, entry.Name)
				tempData, err := os.Open(tempFileName)

				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				totalSize += entry.Size

				tempData.Seek(0, 0)

				written, err := io.Copy(finalFile, tempData)

				if err != nil && err != io.EOF {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				log.Printf("written %d from file %s to final file %s\n", written, entry.Name, finalFileName)

				tempData.Close()

			}

			if totalSize != int64(sentFileSize) {
				log.Printf("size mistmatch total written: %d vs total expected %d", totalSize, sentFileSize)

				err := os.RemoveAll(dir)
				if err != nil {
					return
				}
			}

			fmt.Printf("complete final file size %d\n", totalSize)

			err = os.RemoveAll(dir)
			if err != nil {
				return
			}


			remoteBool, _ := strconv.ParseBool(remote)

			log.Printf("remote bool: %t", remoteBool)

			if remoteBool {

				log.Println("Remote upload detected")

				newUploadModel := services.NewUploadModel{
					Url: fmt.Sprintf("https://kaykatjd.com/media/joshie_%s.%s", fileId, ext),
					FileType: ext,
					Size: strconv.Itoa(int(totalSize)),
					ApplicationId: authModel.ApplicationId,
					UserId: authModel.UserId,
				}
	
				err = c.Service.WriteNewUploadsToDB([]services.NewUploadModel{newUploadModel})

				if err != nil {
					log.Println(err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			}


			w.WriteHeader(http.StatusCreated)
		}

	
	done <- true

	}(r)

	select {
		case <-ctx.Done():
			log.Println("Upload timed out")
			w.WriteHeader(http.StatusRequestTimeout)
			return
	case <-done:
		log.Println("Upload completed")
		w.WriteHeader(http.StatusCreated)
		return
	}
}

type MyFile struct {
	Name        string
	NumericPart int64
	Size        int64
}

func SortFiles(files []os.DirEntry) []MyFile {

	var myFiles []MyFile
	for _, file := range files {
		numericPart, err := extractNumericPartOfFileName(file.Name())
		if err != nil {
			fmt.Println("Error extracting numeric part:", err)
			continue
		}
		info, _ := file.Info()
		myFiles = append(myFiles, MyFile{Name: file.Name(), NumericPart: numericPart, Size: info.Size()})
	}

	// Sort the MyFile array based on the NumericPart
	sort.Slice(myFiles, func(i, j int) bool {
		return myFiles[i].NumericPart < myFiles[j].NumericPart
	})

	return myFiles
}

func extractNumericPartOfFileName(fileName string) (int64, error) {
	numericPart := strings.Split(fileName, "_test_")[0]

	return strconv.ParseInt(numericPart, 10, 64)

}


type DownloadMediaRequest struct {
	Url string `json:"url"`
	FileName string `json:"fileName"`
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



type ResizeImagesRequest struct {
	FilePath string `json:"filePath"`
}


func (c *MediaController) ResizeImagesController(w http.ResponseWriter, r *http.Request) {

	var body ResizeImagesRequest

	err := json.NewDecoder(r.Body).Decode(&body)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	newFiles, err := c.Service.ResizeImages(body.FilePath)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
    /// TODO - add resize images to db

	bytes, err := json.Marshal(newFiles)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}


	w.Write([]byte(bytes))

}