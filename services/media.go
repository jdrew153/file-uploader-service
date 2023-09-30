package services

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/golang-lru/v2"
	"github.com/nfnt/resize"
	"github.com/redis/go-redis/v9"
	"github.com/savsgio/gotils/uuid"
)

type MediaService struct {
	Cache *lru.Cache[string, []byte]
	Redis *redis.Client
	Db    *sql.DB
}

func NewMediaService(cache *lru.Cache[string, []byte], redis *redis.Client, db *sql.DB) *MediaService {
	return &MediaService{
		Cache: cache,
		Redis: redis,
		Db:    db,
	}
}

func (s *MediaService) Get(key string) []byte {
	if value, ok := s.Cache.Get(key); ok {
		return value
	}
	return nil
}

func (s *MediaService) Set(inputPath string) error {

	file, err := os.Open(inputPath)

	if err != nil {
		log.Println(err)
		return err
	}

	buffer := make([]byte, 1024)

	data := bytes.Buffer{}

	defer file.Close()

	for {

		n, err := file.Read(buffer)

		if err != nil {
			if err == io.EOF {
				break
			}
			log.Println(err)
			return err
		}

		data.Write(buffer[:n])
	}

	s.Cache.Add(inputPath, data.Bytes())

	return nil
}

func (s *MediaService) CalculateCacheWeight() {

	size := 0

	var heaviestIndx int
	var heaviestSize int

	for indx, entries := range s.Cache.Values() {
		size += len(entries)

		if len(entries) > heaviestSize {
			heaviestSize = len(entries)
			heaviestIndx = indx
		}
	}

	cachedSizeMB := size / 1000000

	log.Println("Cache size", cachedSizeMB, "MB")

	if cachedSizeMB > 150 {
		log.Println("Cache is too big, removing heaviest key")
		heaviestKey := s.Cache.Keys()[heaviestIndx]

		ok := s.Cache.Remove(heaviestKey)

		if ok {

			log.Printf("Removed key %s, result %v", heaviestKey, ok)
		} else {
			key, _, ok := s.Cache.RemoveOldest()

			if ok {
				log.Printf("Removed key %s", key)
			} else {
				log.Println("Could not remove oldest key")
			}
		}
	} else {
		log.Println("Cache is not too big")
	}
}

type ValidUserIDAndAppIDModel struct {
	UserId        string `json:"userId"`
	ApplicationId string `json:"applicationId"`
}

func (s *MediaService) APIKeyCheck(apiKey string) (ValidUserIDAndAppIDModel, error) {

	var validUserIDAndAppID ValidUserIDAndAppIDModel

	value, err := s.Redis.Get(context.Background(), apiKey).Result()

	log.Println("Value from api key", value)

	if err != nil {
		log.Println(err)
		return validUserIDAndAppID, err
	}

	err = json.Unmarshal([]byte(value), &validUserIDAndAppID)

	if err != nil {
		return validUserIDAndAppID, err

	}

	log.Printf("Value from api key %s\n", value)

	return validUserIDAndAppID, nil
}

type ResizedImageUrlAndSizeModel struct {
	Url  string `json:"url"`
	Size int64  `json:"size"`
}

func (s *MediaService) ResizeImages(filePath string) ([]ResizedImageUrlAndSizeModel, error) {
	sizes := []string{
		"720p",
		"480p",
		"360p",
	}

	basePath := strings.Split(filePath, ".")[0]
	ext := strings.Split(filePath, ".")[1]

	file, err := os.Open(fmt.Sprintf("./media/%s", filePath))

	if err != nil {
		log.Println(err)
		return nil, err
	}

	defer file.Close()

	var newFiles []ResizedImageUrlAndSizeModel

	switch ext {
	case "jpg":

		img, err := jpeg.Decode(file)

		if err != nil {
			return nil, err
		}

		for _, size := range sizes {
			parsedSize := strings.Split(size, "p")[0]
			intSize, err := strconv.Atoi(parsedSize)

			if err != nil {
				return nil, err
			}

			m := resize.Resize(0, uint(intSize), img, resize.Lanczos3)

			newFileName := fmt.Sprintf("./media/%s-%s.%s", basePath, size, ext)

			out, err := os.Create(newFileName)

			if err != nil {
				return nil, err
			}

			jpeg.Encode(out, m, nil)

			baseNewFilePath := strings.Split(newFileName, "./media/")[1]

			newUrl := fmt.Sprintf("https://kaykatjd.com/media/%s", baseNewFilePath)

			fileInfo, err := out.Stat()

			if err != nil {
				return nil, err
			}

			model := ResizedImageUrlAndSizeModel{
				Url:  newUrl,
				Size: fileInfo.Size(),
			}

			newFiles = append(newFiles, model)

			out.Close()

		}

	case "png":
		img, err := png.Decode(file)

		if err != nil {
			return nil, err
		}

		for _, size := range sizes {
			parsedSize := strings.Split(size, "p")[0]
			intSize, err := strconv.Atoi(parsedSize)

			if err != nil {
				return nil, err
			}

			m := resize.Resize(0, uint(intSize), img, resize.Lanczos3)

			newFileName := fmt.Sprintf("./media/%s-%s.%s", basePath, size, ext)
			out, err := os.Create(newFileName)

			if err != nil {
				return nil, err
			}

			png.Encode(out, m)

			baseNewFilePath := strings.Split(newFileName, "./media/")[1]

			newUrl := fmt.Sprintf("https://kaykatjd.com/media/%s", baseNewFilePath)

			fileInfo, err := out.Stat()

			if err != nil {
				return nil, err
			}

			model := ResizedImageUrlAndSizeModel{
				Url:  newUrl,
				Size: fileInfo.Size(),
			}

			newFiles = append(newFiles, model)

			out.Close()

		}

	case "jpeg":
		img, err := jpeg.Decode(file)

		if err != nil {
			return nil, err
		}

		for _, size := range sizes {
			parsedSize := strings.Split(size, "p")[0]
			intSize, err := strconv.Atoi(parsedSize)

			if err != nil {
				return nil, err
			}

			m := resize.Resize(0, uint(intSize), img, resize.Lanczos3)

			newFileName := fmt.Sprintf("./media/%s-%s.%s", basePath, size, ext)
			out, err := os.Create(newFileName)

			if err != nil {
				return nil, err
			}

			jpeg.Encode(out, m, nil)

			baseNewFilePath := strings.Split(newFileName, "./media/")[1]

			newUrl := fmt.Sprintf("https://kaykatjd.com/media/%s", baseNewFilePath)

			fileInfo, err := out.Stat()

			if err != nil {
				return nil, err
			}

			model := ResizedImageUrlAndSizeModel{
				Url:  newUrl,
				Size: fileInfo.Size(),
			}

			newFiles = append(newFiles, model)

			out.Close()

		}

	}

	log.Println("Resized images for", filePath)

	originalUrl := fmt.Sprintf("https://kaykatjd.com/media/%s", filePath)

	fileInfo, err := file.Stat()

	if err != nil {
		return nil, err
	}

	model := ResizedImageUrlAndSizeModel{
		Url:  originalUrl,
		Size: fileInfo.Size(),
	}

	newFiles = append(newFiles, model)

	return newFiles, nil

}

func (s *MediaService) GenerateThumbnail(fileName string) (string, error) {

	file, err := os.Open(fmt.Sprintf("./media/%s", fileName))

	if err != nil {
		log.Println(err)
		return "", err
	}

	defer file.Close()

	img, err := jpeg.Decode(file)

	if err != nil {
		return "", err
	}

	m := resize.Thumbnail(200, 200, img, resize.Lanczos3)

	newFileName := fmt.Sprintf("./media/%s-thumbnail.jpg", fileName)

	out, err := os.Create(newFileName)

	if err != nil {
		return "", err
	}

	jpeg.Encode(out, m, nil)

	out.Close()

	baseNewFilePath := strings.Split(newFileName, "./media/")[1]

	newUrl := fmt.Sprintf("https://kaykatjd.com/media/%s", baseNewFilePath)

	log.Println("Generated thumbnail for", fileName)

	return newUrl, nil

}

type NewUploadModel struct {
	Url           string `json:"url"`
	FileType      string `json:"fileType"`
	Size          string `json:"size"`
	ApplicationId string `json:"applicationId"`
	UserId        string `json:"userId"`
}

func (s *MediaService) WriteNewUploadsToDB(uploads []NewUploadModel) error {

	for _, upload := range uploads {

		log.Println("Application ID", upload.ApplicationId)

		query := "INSERT INTO uploads (id, url, fileType, createdAt, size, applicationId) VALUES (?, ?, ?, ?, ?, ?)"

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		defer cancel()

		stmt, err := s.Db.PrepareContext(ctx, query)

		if err != nil {
			return err
		}

		defer stmt.Close()

		id := uuid.V4()

		result, err := stmt.ExecContext(ctx, id, upload.Url, upload.FileType, time.Now().UnixMilli(), upload.Size, upload.ApplicationId)

		if err != nil {
			return err
		}

		log.Printf("Wrote new upload to db with result %v", result)

	}

	log.Println("Wrote new uploads to db")

	return nil
}

// func (s *MediaService) ConvJPEGToWEBP(
// 	filename string,
// ) (string, error) {

// 	file, err := os.Open(fmt.Sprintf("./media/%s", filename))

// 	if err != nil {
// 		return "", err
// 	}
// 	defer file.Close()

// 	img, err := jpeg.Decode(file)

// 	if err != nil {
// 		return "", err
// 	}

// 	var buff bytes.Buffer

// 	err = webp.Encode(&buff, img, &webp.Options{Lossless: true})

// 	if err != nil {
// 		return "", err
// 	}

// 	baseFileName := strings.Split(filename, ".")[0]

// 	newFileName := fmt.Sprintf("./media/%s.webp", baseFileName)

// 	out, err := os.Create(newFileName)

// 	if err != nil {
// 		return "", err
// 	}

// 	defer out.Close()

// 	_, err = io.Copy(out, &buff)

// 	if err != nil {
// 		return "", err
// 	}

// 	newUrl := fmt.Sprintf("https://kaykatjd.com/media/%s", baseFileName + ".webp")

// 	return newUrl, nil

// }
