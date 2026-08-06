package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"gateway/internal/models"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const clusterRootDir = "/tmp/r10_cluster"
const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func randomString(n int) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, n)
	for i := range b {
		b[i] = charset[r.Intn(len(charset))]
	}
	return string(b)
}

type WorkerConfig struct {
	Name            string
	MachineType     models.MachineType
	MachineCount    int
	MachineCapacity int64
}

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

	fmt.Println("Setting up simulated multiplexed local cluster...")

	if err := os.MkdirAll(clusterRootDir, 0755); err != nil {
		log.Fatalf("Failed to create cluster root dir: %v", err)
	}

	// Configuration objects for 4 workers
	configs := []WorkerConfig{
		{Name: "wkr10_1", MachineType: models.MachineTypeBlock, MachineCount: 12, MachineCapacity: 80},
		{Name: "wkr10_2", MachineType: models.MachineTypeBlock, MachineCount: 12, MachineCapacity: 80},
		{Name: "wkr10_3", MachineType: models.MachineTypeBlock, MachineCount: 12, MachineCapacity: 80},
		{Name: "wkr10_4", MachineType: models.MachineTypeInline, MachineCount: 2, MachineCapacity: 80},
	}

	totalMachinesCreated := 0

	for _, cfg := range configs {
		// Calculate total capacity for this worker daemon
		workerCapacity := cfg.MachineCapacity * int64(cfg.MachineCount)

		worker := models.Worker{
			ID:         uuid.New(),
			Name:       cfg.Name,
			CapacityMB: workerCapacity,
			UsedMB:     0,
			Status:     models.WorkerStatusActive,
		}

		if err := db.Create(&worker).Error; err != nil {
			log.Fatalf("Failed to create worker %s: %v", cfg.Name, err)
		}

		fmt.Printf("Created Worker %s managing %d %s machines.\n", cfg.Name, cfg.MachineCount, cfg.MachineType)

		for i := 0; i < cfg.MachineCount; i++ {
			// Generate 8-char random alphanumeric namespace
			namespace := randomString(8)
			machineName := fmt.Sprintf("machine_%s", namespace)

			machine := models.Machine{
				ID:       uuid.New(),
				Name:     machineName,
				Type:     cfg.MachineType,
				WorkerID: worker.ID,
			}

			if err := db.Create(&machine).Error; err != nil {
				log.Fatalf("Failed to create machine %s: %v", machineName, err)
			}

			disc := models.Disc{
				ID:           uuid.New(),
				SerialNumber: fmt.Sprintf("SN-R10-LOCAL-%s", namespace),
				MachineID:    machine.ID,
				CapacityMB:   cfg.MachineCapacity,
				UsedMB:       0,
				Status:       models.DiscStatusActive,
			}

			if err := db.Create(&disc).Error; err != nil {
				log.Fatalf("Failed to create disc for machine %s: %v", machineName, err)
			}

			machineDir := filepath.Join(clusterRootDir, machineName)
			if err := os.MkdirAll(machineDir, 0755); err != nil {
				log.Fatalf("Failed to create machine dir %s: %v", machineDir, err)
			}

			totalMachinesCreated++
			fmt.Printf("  -> Created %s | Worker: %s | Dir: %s\n", machineName, worker.Name, machineDir)
		}
	}

	fmt.Printf("Cluster setup complete! Created %d workers and %d machines total.\n", len(configs), totalMachinesCreated)
}
