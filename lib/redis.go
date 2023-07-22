package lib

import (
	"context"
	"os"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
)

func CreateRedisClient(lc fx.Lifecycle) *redis.Client {
	err := godotenv.Load("../cmd/.env")
	opts, _ := redis.ParseURL(os.Getenv("REDIS_URL"))
	client := redis.NewClient(opts)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err != nil {
				return err
			}
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