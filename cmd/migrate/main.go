package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found")
	}

	var (
		migrationsPath = flag.String("path", "file://migrations", "path to migrations files")
		dbURL          = flag.String("db", "", "database connection string")
		command        = flag.String("command", "", "migration command (up/down/version/force)")
		forceVersion   = flag.Int("version", -1, "version to force (used with force command)")
	)

	flag.Parse()

	if *dbURL == "" {
		*dbURL = fmt.Sprintf(
			"postgres://%s:%s@%s:%s/%s?sslmode=%s",
			os.Getenv("DB_USER"),
			os.Getenv("DB_PASSWORD"),
			os.Getenv("DB_HOST"),
			os.Getenv("DB_PORT"),
			os.Getenv("DB_NAME"),
			os.Getenv("DB_SSLMODE"),
		)
	}

	m, err := migrate.New(*migrationsPath, *dbURL)
	if err != nil {
		log.Fatalf("Migration initialization failed: %v", err)
	}
	defer m.Close()

	switch *command {
	case "up":
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("Migration up failed: %v", err)
		}
		log.Println("Migration up completed successfully")

	case "down":
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("Migration down failed: %v", err)
		}
		log.Println("Migration down completed successfully")

	case "version":
		v, dirty, err := m.Version()
		if err != nil {
			log.Fatalf("Failed to get version: %v", err)
		}
		log.Printf("Current version: %d, dirty: %v", v, dirty)

	case "force":
		if *forceVersion < 0 {
			log.Fatalf("force command requires -version flag")
		}
		if err := m.Force(*forceVersion); err != nil {
			log.Fatalf("Force failed: %v", err)
		}
		log.Printf("Forced version to %d", *forceVersion)

	default:
		log.Fatalf("Invalid command. Use 'up', 'down', 'version', or 'force'")
	}
}
