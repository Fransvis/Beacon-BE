package main

import (
	"log"
	"os"

	"scam-directory/internal/api"
	"scam-directory/internal/database"
	"scam-directory/internal/repository"

	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found")
	}

	// Initialize database connection
	dbConfig := database.NewConfigFromEnv()
	db, err := database.Connect(dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize repositories
	scamRepo := repository.NewScamRepository(db)
	userRepo := repository.NewUserRepository(db)

	// Initialize handlers
	handler := api.NewHandler(scamRepo)
	authHandler := api.NewAuthHandler(userRepo)

	// Setup router
	router := api.SetupRouter(handler, authHandler)

	// Start server
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
