package services

import (
	"context"
	"fmt"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/hashicorp/golang-lru/v2"
	"github.com/nfnt/resize"
	"github.com/redis/go-redis/v9"
)

type MediaService struct {
	Cache *lru.Cache[string, []byte]
	Redis *redis.Client
}

func NewMediaService(cache *lru.Cache[string, []byte], redis *redis.Client) *MediaService {
	return &MediaService{
		Cache: cache,
		Redis: redis,
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

func (s *MediaService) APIKeyCheck(apiKey string) error {
	_, err := s.Redis.Get(context.Background(), apiKey).Result()

	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func (s *MediaService) ResizeImages(filePath string) ([]string, error) {
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

	var newFiles []string

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

				m := resize.Resize(uint(intSize), 0, img, resize.Lanczos3)

				newFileName := fmt.Sprintf("./media/%s-%s.%s", basePath, size, ext)
				out, err := os.Create(newFileName)

				if err != nil {
					return nil, err
				}

				jpeg.Encode(out, m, nil)

				baseNewFilePath := strings.Split(newFileName, "./media/")[1]

				newUrl := fmt.Sprintf("https://kaykatjd.com/media/%s", baseNewFilePath)

				newFiles = append(newFiles, newUrl)

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

				m := resize.Resize(uint(intSize), 0, img, resize.Lanczos3)

				newFileName := fmt.Sprintf("./media/%s-%s.%s", basePath, size, ext)
				out, err := os.Create(newFileName)

				if err != nil {
					return nil, err
				}

				png.Encode(out, m)

				baseNewFilePath := strings.Split(newFileName, "./media/")[1]

				newUrl := fmt.Sprintf("https://kaykatjd.com/media/%s", baseNewFilePath)

				newFiles = append(newFiles, newUrl)
				
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

				m := resize.Resize(uint(intSize), 0, img, resize.Lanczos3)

				newFileName := fmt.Sprintf("./media/%s-%s.%s", basePath, size, ext)
				out, err := os.Create(newFileName)

				if err != nil {
					return nil, err
				}

				jpeg.Encode(out, m, nil)

				baseNewFilePath := strings.Split(newFileName, "./media/")[1]

				newUrl := fmt.Sprintf("https://kaykatjd.com/media/%s", baseNewFilePath)

				newFiles = append(newFiles, newUrl)

				out.Close()
			}

	}

	log.Println("Resized images for", filePath)

	return newFiles, nil

}