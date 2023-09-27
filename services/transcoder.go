package services

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pusher/pusher-http-go/v5"
	"github.com/redis/go-redis/v9"
	"github.com/xfrr/goffmpeg/models"
	"github.com/xfrr/goffmpeg/transcoder"
)

type TranscoderService struct {
	Pusher *pusher.Client
	Redis *redis.Client
}

func NewTranscoderService(p *pusher.Client, r *redis.Client) *TranscoderService {
	return &TranscoderService{
		Pusher: p,
		Redis: r,
	}

}

const (
	HIGH = "1280x720"
	MEDIUM = "854x480"
	LOW = "640x360"
)

type TranscodeRequest struct {
	InputPath string `json:"inputPath"`
	Resolutions []string `json:"resolutions"`
	ApiKey string `json:"apiKey"`
}

func (s *TranscoderService) Transcode(request TranscodeRequest) int {

	inputPath := request.InputPath
	resolutions := request.Resolutions

	for _, resolution := range resolutions {
		if resolution != HIGH && resolution != MEDIUM && resolution != LOW {
			log.Printf("Invalid resolution %s\n", resolution)
			return 0
		}
	}

	if !IsFilePathValid(inputPath) {
		log.Printf("Invalid file path %s\n", inputPath)
		return 0
	}

	log.Println("Transcoding " + inputPath)

	wg := sync.WaitGroup{}
	wg.Add(len(resolutions))

	fileName := strings.Split(inputPath, "/")[2]

	model := SetActiveTranscodingModel{
		Qualities: resolutions,
		FileId: strings.Split(fileName, ".mp4")[0],
		ApiKey: request.ApiKey,
	}

	err := SetActiveTranscodingUploadId(model, s.Redis)

	if err != nil {
		log.Println("Error setting active transcoding upload id")
		log.Println(err)
		return 0
	}
	

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
			trans.MediaFile().SetVideoBitRate("1000k")
			

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

	RemoveActiveTranscodingKeys(model.ApiKey, s.Redis)

	log.Println("Transcode complete")
	log.Println("Updating upload sizes")

	finalWG := sync.WaitGroup{}
	finalWG.Add(len(resolutions))

	for _, resolution := range resolutions {
	
		go func (resolution string, apiKey string, wg *sync.WaitGroup) {
			defer wg.Done()

			//baseFilePath := strings.Split(inputPath, ".mp4")[0]
			baseResolution := strings.Split(resolution, "x")[1]

			formattedFilePath := fmt.Sprintf("%s_%s.mp4", inputPath, baseResolution)

			CallbackFunctionToUpdateUpload(apiKey, formattedFilePath)

			s.CreateM3U8(inputPath, resolution)

			s.CreateSrcubbingPhotoDirectory(inputPath)

		}(resolution, request.ApiKey, &finalWG)
	}

	finalWG.Wait()

	log.Println("Upload sizes updated")

	//RemoveActiveTranscodingKeys(model.ApiKey, s.Redis)

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

type DownloadRequest struct {
	URL      string `json:"url"`
	FileName string `json:"fileName"`
}

func (s *TranscoderService) DownloadFile(URL, fileName string) error {
	response, err := http.Get(URL)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", response.Status)
	}

	file, err := os.Create(fmt.Sprintf("./media/%s", fileName))
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, response.Body)
	if err != nil {
		return err
	}
	
	return nil
}

func (s *TranscoderService) CreateM3U8(inputPath string, resoultion string) error {

	baseFilePath := strings.Split(inputPath, ".mp4")[0]

	log.Println("base file path", baseFilePath)

	fileId := strings.Split(baseFilePath, "/")[2]

	// Create the directory if it doesn't exist
	m3u8Dir := fmt.Sprintf("./media/%s", fileId)



	if _, err := os.Stat(m3u8Dir); os.IsNotExist(err) {
		if err := os.MkdirAll(m3u8Dir, 0755); err != nil {
			log.Println("Error creating m3u8 directory:", err)
			return err
		}
	}

	baseResolution := strings.Split(resoultion, "x")[1]

	m3u8FilePath := fmt.Sprintf("./media/%s/%s.m3u8", fileId, baseResolution)

	currFilePath := fmt.Sprintf("./media/%s.mp4", fileId)

	log.Printf("current file path: %s\n", currFilePath)
	log.Printf("m3u8 file path: %s\n", m3u8FilePath)

	cmd := exec.Command("ffmpeg", "-i", currFilePath, "-c:v", "h264", "-b:v", "1M", "-hls_time", "10", "-hls_list_size", "0", m3u8FilePath)


	cmd.Stderr = os.Stderr

	err := cmd.Run()

	if err != nil {
		log.Println("Error converting mp4 to m3u8:", err)
		// Print the stderr output from ffmpeg command for debugging purposes.
		log.Println("ffmpeg stderr:", cmd.Stderr)
		return err
	}


	log.Println("Created m3u8 file")
	return nil
}

func (s *TranscoderService) CreateSrcubbingPhotoDirectory(inputPath string) error {

	log.Println("scrubbing input path", inputPath)
	// note - input path should be .mp4 file 
    baseFileId := strings.Split(inputPath, ".mp4")[0]

	uploadId := strings.Split(baseFileId, "/")[2]

	log.Println("base file id", baseFileId)

	mediaDir := fmt.Sprintf("./%s", baseFileId)

	log.Println("media dir", mediaDir)
	
	outPath := fmt.Sprintf("%s/%s", mediaDir, uploadId)


	cmd := exec.Command("ffmpeg", "-i", inputPath, "-vf", "fps=1/10,scale=120:-1", outPath + "_%d.jpg")

	cmd.Stderr = &bytes.Buffer{}

	err := cmd.Run()

	if err != nil {
		log.Println("Error creating scrubbing photo directory:", err)
		log.Println("ffmpeg stderr:", cmd.Stderr)
		return err
	}

	log.Println("Created scrubbing photo directory")

	return nil
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


type UpdateUploadRequest struct {
	Url string `json:"url"`
	Size int `json:"size"`
}

func CallbackFunctionToUpdateUpload(apiKey string, filePath string) {

	requestUrl := "http://localhost:3000/api/applications/uploads/callback"


	out, err := os.OpenFile(filePath, os.O_RDONLY, 0666)
	
	if err != nil {
		log.Println(err)
		return
	}

	defer out.Close()

	fileInfo, _ := out.Stat()

	fileSize := fileInfo.Size()

	log.Println("file size..", fileSize)
	
	uploadUrl := fmt.Sprintf("http://localhost:3002/%s", strings.Split(filePath, "./")[1])

	requestBody, err := json.Marshal(UpdateUploadRequest{
		Url: uploadUrl,
		Size: int(fileSize),
	})

	if err != nil {
		log.Println(err)
		return
	}

	req, err := http.NewRequest(http.MethodPost, requestUrl, bytes.NewBuffer(requestBody))

	
	if err != nil {
		log.Println(err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)

	client := http.DefaultClient

	resp, err := client.Do(req)


	if err != nil {
		log.Println(err)
		return
	}

	fmt.Println("total response..", resp)

	fmt.Println("upload update request status..", resp.StatusCode)

	if resp.StatusCode == http.StatusOK {
		log.Println("Successfully updated upload")
		return
	} else {
		log.Println("Failed to update upload")
		return
	}
	
}

type SetActiveTranscodingModel struct {
	FileId string `json:"uploadId"`
	Qualities []string `json:"qualities"`
	ApiKey string `json:"apiKey"`
}


// this is the channel structure in pusher ---> "transcoding-%s-quality-%s", fileID, quality

func SetActiveTranscodingUploadId(activeTranscodingModel SetActiveTranscodingModel, r *redis.Client) error {
	
	log.Println("setting active transcoding in redis", r)

	ctx := context.Background()


	var values []string


	for _, quality := range activeTranscodingModel.Qualities {
		values = append(values, fmt.Sprintf("transcoding-%s-quality-%s", activeTranscodingModel.FileId, quality))
	}

	fmt.Println("values to set in redis", activeTranscodingModel)

	data , err := json.Marshal(values)

	if err != nil {
		log.Println(err)
		return err
	}

	cmd := r.Set(ctx, fmt.Sprintf("transcoding:%s", activeTranscodingModel.ApiKey), data, time.Duration(0))

	result, err := cmd.Result()

	if err != nil {
		log.Println(err)
		return err
	}

	log.Println("result of setting active transcoding in redis", result)

	return nil
}

func RemoveActiveTranscodingKeys(apiKey string, r *redis.Client) error {

	cmd := r.Del(context.Background(), fmt.Sprintf("transcoding:%s", apiKey))

	_, err := cmd.Result()

	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}