package api

import (
	"time"

	"gateway/internal/controllers"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	// CORS config (allow frontend to connect)
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:4200"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Basic health check
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})

	v1 := r.Group("/api/v1")
	{
		files := v1.Group("/files")
		{
			files.DELETE("/:blob_id", controllers.DeleteFile)
		}

		uploads := v1.Group("/uploads")
		{
			uploads.POST("/init", controllers.InitUpload)
			uploads.PUT("/:upload_id/parts/:part_number", controllers.UploadPart)
			uploads.POST("/:upload_id/complete", controllers.CompleteUpload)
		}
	}

	return r
}
