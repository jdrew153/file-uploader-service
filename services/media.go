package services

import (
	"io"
	"log"
	"os"

	"github.com/hashicorp/golang-lru/v2"
)

type MediaService struct {
	Cache *lru.Cache[string, []byte]
}

func NewMediaService(cache *lru.Cache[string, []byte]) *MediaService {
	return &MediaService{
		Cache: cache,
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