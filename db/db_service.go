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

func Initialize() *pgxpool.Pool {
	runMigrations()
	return connect()
}

func connect() *pgxpool.Pool {
	connStr := os.Getenv("DATABASE_URL")
	pool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		log.Fatal(err)
	}

	return pool
}

func runMigrations() {
	dbURL := os.Getenv("DATABASE_URL")
	// Convert postgres:// scheme to postgresql:// for lib/pq driver
	migrationURL := strings.Replace(dbURL, "postgres://", "postgresql://", 1)
	log.Printf("Migration DATABASE_URL: %s", migrationURL)
	m, err := migrate.New(
		"file://db/migrations",
		migrationURL)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatal(err)
	}
}
