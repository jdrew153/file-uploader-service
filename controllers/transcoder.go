package controllers

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/jdrew153/services"
)

type TranscoderController struct {
	Service *services.TranscoderService
}

func NewTranscoderController(service *services.TranscoderService) *TranscoderController {
	return &TranscoderController{
		Service: service,
	}
}

type TranscodeRequest struct {
	InputPath string `json:"inputPath"`
}


func (c *TranscoderController) Transcode(w http.ResponseWriter, r *http.Request) {
	var body TranscodeRequest

	err := json.NewDecoder(r.Body).Decode(&body)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	result := c.Service.Transcode(body.InputPath)

	if result != 1 {
		http.Error(w, "Transcode failed", http.StatusInternalServerError)
	}

	w.Write([]byte("Transcode complete"))
}

func (c *TranscoderController) ThumbnailFileReceiver(w http.ResponseWriter, r *http.Request) {

	file, header, err := r.FormFile("file")

	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	log.Println("got file form req...")
	if file == nil {
		log.Println("No file received")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	defer file.Close()

	out , err := os.Create(header.Filename)

	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Println("created file...")

	defer out.Close()

	_, err = io.Copy(out, file)

	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Println("copied file...")


	thumbnail, err := c.Service.CreateThumbnail(header.Filename)

	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Println("created thumbnail...")

	if len(thumbnail) > 0 {
		// err := os.WriteFile(header.Filename+"_thumbnail.jpg", thumbnail, 0644)

		// if err != nil {
		// 	log.Println(err)
		// 	w.WriteHeader(http.StatusInternalServerError)
		// }

		err = os.Remove(header.Filename) 

		
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		log.Println("removed file...")

		err = os.Remove(header.Filename+"_thumbnail.jpeg")


		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		log.Println("removed thumbnail...")

		 // Set the appropriate content type header
		 w.Header().Set("Content-Type", "image/jpeg") 

		 
		base64Rep := base64.StdEncoding.EncodeToString(thumbnail)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(base64Rep))
	} else {
		log.Println("No thumbnail created")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

}