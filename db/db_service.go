package db

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
)

func Connect() (conn *pgx.Conn) {
	connStr := os.Getenv("DATABASE_URL")
	conn, err := pgx.Connect(context.Background(), connStr)
	if err != nil {
		log.Fatal(err)
	}

	return conn
}
