package db

import (
	"context"
	"log"
	"os"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Initialize reads DATABASE_URL from the environment, runs migrations, and returns a pool.
func Initialize() *pgxpool.Pool {
	pool, err := InitializeWithURL(os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	return pool
}

// InitializeWithURL runs migrations for the given URL and returns a connection pool.
func InitializeWithURL(databaseURL string) (*pgxpool.Pool, error) {
	if err := runMigrations(databaseURL); err != nil {
		return nil, err
	}
	return connect(databaseURL)
}

func connect(connStr string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		return nil, err
	}
	return pool, nil
}

func runMigrations(dbURL string) error {
	migrationURL := strings.Replace(dbURL, "postgres://", "postgresql://", 1)
	log.Printf("running migrations against %s", migrationURL)
	m, err := migrate.New("file://db/migrations", migrationURL)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}
	return nil
}
