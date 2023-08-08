package main

import (
	"github.com/jdrew153/controllers"
	"github.com/jdrew153/lib"
	"github.com/jdrew153/server"
	"github.com/jdrew153/services"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		fx.Provide(
			controllers.NewTranscoderController,
			controllers.NewMediaController,
			services.NewTranscoderService,
			services.NewMediaService,
			lib.CreatePusherClient,
			lib.CreateRedisClient,
			lib.CreateCache,
			lib.CreateDBConnection,
		),
		fx.Invoke(server.NewMuxServer),
	).Run()
}
