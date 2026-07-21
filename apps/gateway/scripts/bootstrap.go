package main

import (
	"fmt"
	"log"
	"os"

	"gateway/internal/models"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// This script is meant to be run manually to bootstrap and seed the database.
// Usage: go run scripts/bootstrap.go
func migrate() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables")
	}

	// Fetch connection details from environment variables
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbSSLMode := os.Getenv("DB_SSLMODE")

	if dbHost == "" || dbUser == "" || dbName == "" {
		log.Fatal("Missing required database environment variables")
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC", 
		dbHost, dbUser, dbPassword, dbName, dbPort, dbSSLMode)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	fmt.Println("Database connection established. Starting migrations...")

	// Enable uuid-ossp extension in case gen_random_uuid() requires it (though in PG 13+ it's built-in)
	db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";")

	// Run AutoMigrate for all defined models
	err = db.AutoMigrate(
		&models.Machine{},
		&models.Disc{},
		&models.Worker{},
		&models.User{},
		&models.Memblock32{},
		&models.Blob{},
		&models.BlobChunk{},
		&models.Backup{},
	)

	if err != nil {
		log.Fatalf("Failed to migrate database schemas: %v", err)
	}

	fmt.Println("Successfully migrated schemas! Database is bootstrapped.")
}

func main() {
	migrate()
}
