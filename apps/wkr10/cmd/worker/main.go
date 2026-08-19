package main

import (
	"log"
	"net/http"
	"os"

	"wkr10/internal/handlers"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env file if present
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables")
	}

	// Parse Environment Variables
	port := os.Getenv("PORT")
	clusterRootDir := os.Getenv("CLUSTER_ROOT_DIR") // e.g. /tmp/r10_cluster
	workerName := os.Getenv("WORKER_NAME")

	if port == "" || clusterRootDir == "" || workerName == "" {
		log.Fatal("Missing required environment variables. Ensure PORT, CLUSTER_ROOT_DIR, and WORKER_NAME are set.")
	}

	// Ensure the root directory exists
	if err := os.MkdirAll(clusterRootDir, 0755); err != nil {
		log.Fatalf("Failed to initialize CLUSTER_ROOT_DIR at %s: %v", clusterRootDir, err)
	}

	log.Printf("Starting Multiplexed wkr10 (%s) on port %s", workerName, port)
	log.Printf("Cluster Root Dir: %s", clusterRootDir)

	r := gin.Default()

	// Basic Health Check
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message":     "pong from multiplexed wkr10",
			"worker_name": workerName,
			"status":      "active",
		})
	})

	chunkHandler := handlers.NewChunkHandler(clusterRootDir)

	v1 := r.Group("/api/v1")
	{
		v1.POST("/machines/:machine_namespace/chunks", chunkHandler.WriteBlockChunk)
		v1.POST("/machines/:machine_namespace/append", chunkHandler.AppendInlineChunk)
		v1.GET("/machines/:machine_namespace/chunks", chunkHandler.ReadChunk)
	}

	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start wkr10 server: %v", err)
	}
}
