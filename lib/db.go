package lib

import (
	"context"
	"os"

	"go.uber.org/fx"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)


func CreateDBConnection(lc fx.Lifecycle) *gorm.DB {

	db, err := gorm.Open(mysql.Open(os.Getenv("DSN")), &gorm.Config{})


	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			
			if err != nil {
				return err
			}

			return nil
		},
		OnStop: func(ctx context.Context) error {
			sqlDB, _ := db.DB()
			err := sqlDB.Close()
			if err != nil {
				return err
			}
			return nil
		},
	})

	return db
}