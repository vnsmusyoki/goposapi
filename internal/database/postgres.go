package database

import (
	"context"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

func Connect(database string) (*pgxpool.Pool, error) {
	
	var ctx context.Context = context.Background()
	var config *pgxpool.Config
	var err error = nil
	
	config, err = pgxpool.ParseConfig(database)
	if err != nil {
		return nil, err
	}

	config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		_, err := conn.Exec(ctx, "SET TIME ZONE 'UTC'")
		return err
	}

    var pool *pgxpool.Pool 

	pool,err = pgxpool.ConnectConfig(ctx, config)

	if err != nil {
		return nil, err
	}
	

	err = pool.Ping(ctx)
	if err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}
