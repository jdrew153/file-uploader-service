package lib

import (
	"context"

	"github.com/hashicorp/golang-lru/v2"
	"go.uber.org/fx"
)

func CreateCache(lc fx.Lifecycle) *lru.Cache[string, []byte]  {
		l, err := lru.New[string, []byte](100)

		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {

				if err != nil {
					return err
				}

				return nil

			},
			OnStop: func(ctx context.Context) error {
				l.Purge()
				return nil
			},
		})

		return l

}