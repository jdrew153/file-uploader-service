package services

import (
	"context"
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
	"github.com/jdrew153/models"
	"github.com/nfnt/resize"
	"github.com/redis/go-redis/v9"
	"github.com/savsgio/gotils/uuid"
	"gorm.io/gorm"
)

type MediaService struct {
	Cache *lru.Cache[string, []byte]
	Redis *redis.Client
	Db *gorm.DB
}

func NewMediaService(cache *lru.Cache[string, []byte], redis *redis.Client, db *gorm.DB) *MediaService {
	return &MediaService{
		Cache: cache,
		Redis: redis,
		Db: db,
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

	var buffer []byte

	defer file.Close()

	for {
		bytes := make([]byte, 1024)

		_, err := file.Read(bytes)

		if err != nil {
			if err == io.EOF {
				break
			}
			log.Println(err)
			return err
		}

		buffer = append(buffer, bytes...)
	}

	s.Cache.Add(inputPath, buffer)



	return nil
}

func (s *MediaService) CalculateCacheWeight() {

	size := 0
	
	for _, entries  := range s.Cache.Values() {
		size += len(entries)
	}

	log.Println("Cache size", size / 1000000, "MB")
}

type ValidUserIDAndAppIDModel struct {
	UserId string `json:"user_id"`
	ApplicationId string `json:"application_id"`
}

func (s *MediaService) APIKeyCheck(apiKey string) (*ValidUserIDAndAppIDModel, error) {

	var validUserIDAndAppID ValidUserIDAndAppIDModel

	value, err := s.Redis.Get(context.Background(), apiKey).Result()


	if err != nil {
		log.Println(err)
		return nil, err
	}

	err = json.Unmarshal([]byte(value), &validUserIDAndAppID)

	if err != nil {
		return nil, err

	}	

	log.Printf("Value from api key %s\n", value)

	return &validUserIDAndAppID, nil
}

type ResizedImageUrlAndSizeModel struct {
	Url string `json:"url"`
	Size int64 `json:"size"`
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
					Url: newUrl,
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
					Url: newUrl,
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
					Url: newUrl,
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
		Url: originalUrl,
		Size: fileInfo.Size(),
	}

	newFiles = append(newFiles, model)

	return newFiles, nil

}


type NewUploadModel struct {
	Url string `json:"url"`
	FileType string `json:"fileType"`
	Size string `json:"size"`
	ApplicationId string `json:"applicationId"`
	UserId string `json:"userId"`
}

func (s *MediaService) WriteNewUploadsToDB(uploads []NewUploadModel) error {

	
	for _, upload := range uploads {

		err := s.Db.Create(&models.Upload{
			Url: upload.Url,
			FileType: upload.FileType,
			Size: upload.Size,
			ApplicationId: upload.ApplicationId,
			CreatedAt: time.Now().UnixMilli(),
			Id: uuid.V4(),
			UserId: upload.UserId,
		}).Error

		if err != nil {
			return err
		}

	}

	log.Println("Wrote new uploads to db")

	return nil
}