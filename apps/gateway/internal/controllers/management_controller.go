package controllers

import (
	"net/http"

	"gateway/internal/models"
	"gateway/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ManagementController struct {
	mgtService *services.ManagementService
	db         *gorm.DB
}

func NewManagementController(db *gorm.DB) *ManagementController {
	return &ManagementController{
		mgtService: services.NewManagementService(db),
		db:         db,
	}
}

func (mc *ManagementController) BootstrapCluster(c *gin.Context) {
	jobID, err := mc.mgtService.BootstrapCluster()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"job_id": jobID, "message": "Bootstrap started"})
}

func (mc *ManagementController) ResetCluster(c *gin.Context) {
	jobID, err := mc.mgtService.ResetCluster()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"job_id": jobID, "message": "Reset started"})
}

func (mc *ManagementController) GetJobStatus(c *gin.Context) {
	jobIDParam := c.Param("job_id")
	jobID, err := uuid.Parse(jobIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job id"})
		return
	}

	var job models.Job
	if err := mc.db.First(&job, "id = ?", jobID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	c.JSON(http.StatusOK, job)
}

func (mc *ManagementController) GetClusterStats(c *gin.Context) {
	var totalWorkers int64
	var totalMachines int64

	mc.db.Model(&models.Worker{}).Count(&totalWorkers)
	mc.db.Model(&models.Machine{}).Count(&totalMachines)

	type StatResult struct {
		Cap  int64
		Used int64
	}
	var res StatResult
	mc.db.Model(&models.Worker{}).Select("sum(capacity_mb) as cap, sum(used_mb) as used").Scan(&res)

	c.JSON(http.StatusOK, gin.H{
		"workers":       totalWorkers,
		"machines":      totalMachines,
		"capacity_mb":   res.Cap,
		"used_mb":       res.Used,
	})
}

func (mc *ManagementController) GetWorkers(c *gin.Context) {
	var workers []models.Worker
	// Preload machines
	if err := mc.db.Preload("Machines").Find(&workers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, workers)
}
