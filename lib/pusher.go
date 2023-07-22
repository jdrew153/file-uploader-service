package lib

import (
	"context"
	"os"

	"github.com/pusher/pusher-http-go/v5"
	"go.uber.org/fx"
)


func CreatePusherClient(lc fx.Lifecycle) *pusher.Client {
	//err := godotenv.Load("./cmd/.env")

	client := pusher.Client{
		AppID:   os.Getenv("PUSHER_APP_ID"),
		Key:     os.Getenv("PUSHER_KEY"),
		Secret:  os.Getenv("PUSHER_SECRET"),
		Cluster: os.Getenv("PUSHER_CLUSTER"),
		Secure:  true,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {

			// if err != nil {
			// 	return err
			// }

			return nil
		},
		OnStop: func(ctx context.Context) error {
			return nil
		},
	})

	return &client
}