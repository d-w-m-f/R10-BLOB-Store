package main

import (
	"fmt"
	"log"
	"os"

	"gateway/internal/services"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// setup_cluster provisions the simulated local cluster synchronously.
// It shares ManagementService.BuildTopology with the async /management/bootstrap
// job so the CLI and the Control Plane can never drift apart.
func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables")
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
		os.Getenv("DB_HOST"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"), os.Getenv("DB_PORT"), os.Getenv("DB_SSLMODE"))

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	fmt.Println("Setting up simulated multiplexed local cluster...")

	if err := services.NewManagementService(db).BuildTopology(); err != nil {
		log.Fatalf("Failed to build cluster topology: %v", err)
	}

	machines := 0
	for _, cfg := range services.ClusterTopology {
		fmt.Printf("Created Worker %-8s @ %-22s -> %2d %s machines\n",
			cfg.Name, cfg.Address, cfg.MachineCount, cfg.MachineType)
		machines += cfg.MachineCount
	}

	fmt.Printf("Cluster setup complete! Created %d workers and %d machines total.\n",
		len(services.ClusterTopology), machines)
}
