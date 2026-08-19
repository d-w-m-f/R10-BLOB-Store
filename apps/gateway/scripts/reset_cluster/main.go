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

const clusterRootDir = "/tmp/r10_cluster"

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables")
	}

	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbSSLMode := os.Getenv("DB_SSLMODE")

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
		dbHost, dbUser, dbPassword, dbName, dbPort, dbSSLMode)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	fmt.Println("Resetting local simulated cluster...")

	// 1. Delete DB rows for hardware topology
	// Order matters due to foreign keys: Disc -> Machine -> Worker
	if err := db.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&models.Disc{}).Error; err != nil {
		log.Fatalf("Failed to truncate discs: %v", err)
	}
	if err := db.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&models.Machine{}).Error; err != nil {
		log.Fatalf("Failed to truncate machines: %v", err)
	}
	if err := db.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&models.Worker{}).Error; err != nil {
		log.Fatalf("Failed to truncate workers: %v", err)
	}

	fmt.Println("Cleared hardware metadata from Database.")

	// 2. Remove physical directories
	if err := os.RemoveAll(clusterRootDir); err != nil {
		log.Fatalf("Failed to delete physical cluster directories: %v", err)
	}

	fmt.Printf("Deleted cluster directory: %s\n", clusterRootDir)
	fmt.Println("Cluster reset successfully.")
}
