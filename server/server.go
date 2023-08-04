package server

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/jdrew153/controllers"
	"github.com/rs/cors"
	"go.uber.org/fx"
)

func NewMuxServer(lc fx.Lifecycle, 
	mediaController *controllers.MediaController,
	transcoderController *controllers.TranscoderController) *http.ServeMux {

	mux := http.NewServeMux()

	mux.HandleFunc("/", mediaController.ServeContent)

	mux.HandleFunc("/transcode", transcoderController.Transcode)

	mux.HandleFunc("/download", mediaController.DownloadContent)

	mux.HandleFunc("/resize", mediaController.ResizeImagesController)

	mux.HandleFunc("/download-transcode", transcoderController.DownloadFromUrlToTranscode)

	mux.HandleFunc("/thumbnail", transcoderController.ThumbnailFileReceiver)

	mux.HandleFunc("/m3u8", transcoderController.WriteNewM3U8FileFromMP4)

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "OPTIONS", "PUT", "DELETE"},
		AllowedHeaders: []string{"*"},
	})

	handler := c.Handler(mux)

	var serverHolder *http.Server

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			
			go func() {
				if os.Getenv("PORT") != "" {
					log.Println("Starting server on port " + os.Getenv("PORT"))
					server := &http.Server{
						Addr:    ":" + os.Getenv("PORT"),
						Handler: handler,
					}

					serverHolder = server
				if err := server.ListenAndServe(); err != nil {
					log.Println(err)
				}
				} else {
					log.Println("Starting server on port 3002")
					server := &http.Server{
						Addr:    ":3002",
						Handler: handler,
					}
					serverHolder = server
					if err := server.ListenAndServe(); err != nil {
						log.Println(err)
					}
				}
				
			}()
		
			return nil
		},
		OnStop: func(context.Context) error {
			if err := serverHolder.Shutdown(context.Background()); err != nil {
				return err
			}
			return nil
		},
	})

	return mux
}