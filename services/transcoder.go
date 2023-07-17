package services

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pusher/pusher-http-go/v5"
	"github.com/xfrr/goffmpeg/models"
	"github.com/xfrr/goffmpeg/transcoder"
)

type TranscoderService struct {
	Pusher *pusher.Client
}

func NewTranscoderService(p *pusher.Client) *TranscoderService {
	return &TranscoderService{
		Pusher: p,
	}
}

func (s *TranscoderService) Transcode(inputPath string) int {

	if !IsFilePathValid(inputPath) {
		log.Printf("Invalid file path %s\n", inputPath)
		return 0
	}

	log.Println("Transcoding " + inputPath)

	resolutions := []string{"1920x1080", "854x480", "640x360", "480x360", "426x240"}
	wg := sync.WaitGroup{}
	wg.Add(len(resolutions))
	for _, resolution := range resolutions {

		go func(resolution string) {

			defer wg.Done()

			trans := new(transcoder.Transcoder)
			err := trans.Initialize(inputPath, inputPath+"_"+resolution+".mp4")

			if err != nil {
				return
			}
			log.Println("Transcoding to " + resolution)

			trans.MediaFile().SetResolution(resolution)
			trans.MediaFile().SetVideoBitRate("10000k")
			trans.MediaFile().SetAspect("16:9")
			done := trans.Run(true)

			progress := trans.Output()

			for msg := range progress {
				log.Println("resolution: " + resolution)
				log.Println(msg)
				SendPusherNotif(inputPath, resolution, msg, s.Pusher)
			}

			err = <-done

			if err != nil {
				log.Println("Error transcoding " + inputPath + " to " + resolution)
				log.Println(err)
			}
			
		}(resolution)
	}

	wg.Wait()

	log.Println("Transcode complete")
	return 1
}

func (s *TranscoderService) CreateThumbnail(inputPath string) ([]byte, error) {

	cmd := exec.Command("ffmpeg", "-i", inputPath, 
	"-ss", "00:00:01.000", 
	"-vframes", 
	"1", inputPath+"_thumbnail.jpeg");

	err := cmd.Run()

	if err != nil {
		return nil, err
	}

	file, err := os.Open(inputPath+"_thumbnail.jpeg")

	if err != nil {
		return nil, err
	}

	defer file.Close()

	fileInfo, _ := file.Stat()

	var size int64 = fileInfo.Size()
	bytes := make([]byte, size)

	buffer := bufio.NewReader(file)

	_, err = buffer.Read(bytes)

	if err != nil {
		return nil, err
	}

	return bytes, nil
	
}

// Additional functions related to Transcoder service, but not required for direct use in controllers..

func IsFilePathValid(filePath string) bool {
    _, err := os.Stat(filePath)
    if err != nil {
        if os.IsNotExist(err) {
            // File does not exist
            return false
        }
        // Other error occurred
        return false
    }
    // File exists
    return true
}

func SendPusherNotif(filePath string, quality string, msg models.Progress, client *pusher.Client) {

	time.Sleep(2 * time.Second)
	
	progressMsg := strconv.FormatFloat(msg.Progress, 'f', 2, 64)

	data := map[string]string{ "progress": progressMsg }

	err := client.Trigger(fmt.Sprintf("transcoding-%s-quality-%s", strings.Split(strings.Split(filePath, "/")[2], ".")[0], quality), "progress", data)

	if err != nil {
		log.Println(err)
	}
	
	log.Printf("Sent pusher notification for %s\n", filePath)
}

