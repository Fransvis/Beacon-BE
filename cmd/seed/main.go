package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"

	"scam-directory/internal/database"
	"scam-directory/internal/repository"
	"scam-directory/internal/seed"
)

func main() {
	truncate := flag.Bool("truncate", false, "truncate existing scams before seeding")
	flag.Parse()

	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	cfg := database.NewConfigFromEnv()
	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if *truncate {
		log.Println("Truncating existing scams...")
		if _, err := db.ExecContext(context.Background(), "TRUNCATE TABLE keywords, related_scams, evidence, demographics, transfer_methods, contact_methods, locations, scam_reports, comments, scams CASCADE"); err != nil {
			log.Fatalf("Failed to truncate: %v", err)
		}
	}

	scams := seed.SeedScams()
	repo := repository.NewScamRepository(db)
	ctx := context.Background()

	created := 0
	for _, s := range scams {
		if err := repo.CreateScam(ctx, &s); err != nil {
			log.Printf("Failed to create scam %s: %v", *s.Title, err)
			continue
		}

		for _, kw := range s.Keywords {
			if _, err := db.ExecContext(ctx, "INSERT INTO keywords (scam_id, keyword) VALUES ($1, $2) ON CONFLICT DO NOTHING", s.ID, kw); err != nil {
				log.Printf("Failed to insert keyword for %s: %v", *s.Title, err)
			}
		}

		for _, d := range s.Demographics {
			if _, err := db.ExecContext(ctx, "INSERT INTO demographics (scam_id, age_range, location, occupation, count) VALUES ($1, $2, $3, $4, $5)", s.ID, d.AgeRange, d.Location, d.Occupation, d.Count); err != nil {
				log.Printf("Failed to insert demographic for %s: %v", *s.Title, err)
			}
		}

		created++
	}

	fmt.Printf("Seeded %d/%d scams\n", created, len(scams))
	if created < len(scams) {
		os.Exit(1)
	}
}
