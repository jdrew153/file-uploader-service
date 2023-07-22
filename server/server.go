package server

import (
	"context"
	"log"
	"net/http"

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

	mux.HandleFunc("/download-transcode", transcoderController.DownloadFromUrlToTranscode)

	mux.HandleFunc("/thumbnail", transcoderController.ThumbnailFileReceiver)

	mux.HandleFunc("/m3u8", transcoderController.WriteNewM3U8FileFromMP4)

	handler := cors.Default().Handler(mux)
	server := &http.Server{
		Addr:    ":3002",
		Handler: handler,
	}

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			
			go func() {
				log.Println("Starting server on port 3002")
				if err := server.ListenAndServe(); err != nil {
					log.Println(err)
				}
			}()
		
			return nil
		},
		OnStop: func(context.Context) error {
			if err := server.Shutdown(context.Background()); err != nil {
				return err
			}
			return nil
		},
	})

	return mux
}
