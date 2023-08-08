package lib

import (
	"context"
	"database/sql"
	"os"

	"go.uber.org/fx"
	_ "github.com/go-sql-driver/mysql"
)


func CreateDBConnection(lc fx.Lifecycle) *sql.DB {

	db, err := sql.Open("mysql", os.Getenv("DSN"))


	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			
			if err != nil {
				return err
			}

			return nil
		},
		OnStop: func(ctx context.Context) error {
		
			err := db.Close()
			if err != nil {
				return err
			}
			return nil
		},
	})

	return db
}