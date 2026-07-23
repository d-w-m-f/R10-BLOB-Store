package main

import (
	"log"

	"gateway/internal/api"

	"github.com/joho/godotenv"
)

func main() {
	// Load env variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables")
	}

	// Initialize the router
	router := api.SetupRouter()

	log.Println("Starting R10 Gateway on port 8080...")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
