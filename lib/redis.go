package lib

import (
	"context"
	"log"
	"os"

	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
)

func CreateRedisClient(lc fx.Lifecycle) *redis.Client {
	//err := godotenv.Load("./cmd/.env")
	log.Println("redis url...", os.Getenv("REDIS_URL"))
	opts, _ := redis.ParseURL(os.Getenv("REDIS_URL"))
	client := redis.NewClient(opts)

	log.Println(client)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// if err != nil {
			// 	log.Println(err)
			// 	return err
			// }
			return nil
		},
		OnStop: func(ctx context.Context) error {
			err := client.Conn().Close()
			if err != nil {
				return err
			}
			return nil
		},
	})
	return client
}