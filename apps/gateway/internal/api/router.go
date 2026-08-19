package api

import (
	"time"

	"gateway/internal/controllers"
	"gateway/internal/services"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupRouter(db *gorm.DB) (*gin.Engine, error) {
	storage, err := services.NewStorageService(db)
	if err != nil {
		return nil, err
	}

	r := gin.Default()

	// CORS config (allow frontend to connect)
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:4200"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length", "Content-Disposition"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Basic health check
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})

	fileCtrl := controllers.NewFileController(db, storage)
	uploadCtrl := controllers.NewUploadController(db, storage)
	mgtCtrl := controllers.NewManagementController(db)

	v1 := r.Group("/api/v1")
	{
		files := v1.Group("/files")
		{
			files.GET("", fileCtrl.ListFiles)
			files.GET("/:blob_id", fileCtrl.GetFile)
			files.GET("/:blob_id/download", fileCtrl.DownloadFile)
			files.DELETE("/:blob_id", fileCtrl.DeleteFile)
		}

		uploads := v1.Group("/uploads")
		{
			uploads.POST("/init", uploadCtrl.InitUpload)
			uploads.PUT("/:upload_id/parts/:part_number", uploadCtrl.UploadPart)
			uploads.POST("/:upload_id/complete", uploadCtrl.CompleteUpload)
		}

		management := v1.Group("/management")
		{
			management.GET("/cluster", mgtCtrl.GetClusterStats)
			management.GET("/workers", mgtCtrl.GetWorkers)
			management.POST("/bootstrap", mgtCtrl.BootstrapCluster)
			management.POST("/reset", mgtCtrl.ResetCluster)
			management.GET("/jobs/:job_id", mgtCtrl.GetJobStatus)
		}
	}

	return r, nil
}
