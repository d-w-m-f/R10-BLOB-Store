package controllers

import (
	"net/http"

	"gateway/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type FileController struct {
	db *gorm.DB
}

func NewFileController(db *gorm.DB) *FileController {
	return &FileController{db: db}
}

// ListFiles retrieves all non-deleted files stored in the R10 Blob Store catalog.
func (fc *FileController) ListFiles(c *gin.Context) {
	var blobs []models.Blob
	if err := fc.db.Where("deleted = ?", false).Order("created_at desc").Find(&blobs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Always return an empty array instead of nil if no files exist
	if blobs == nil {
		blobs = []models.Blob{}
	}

	c.JSON(http.StatusOK, blobs)
}

// DeleteFile performs a LOGICAL deletion of a file in the GORM catalog.
func (fc *FileController) DeleteFile(c *gin.Context) {
	blobID := c.Param("blob_id")

	if blobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "blob_id is required"})
		return
	}

	// Perform logical soft delete in the database
	res := fc.db.Model(&models.Blob{}).Where("id = ? AND deleted = ?", blobID, false).Update("deleted", true)
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": res.Error.Error()})
		return
	}

	// Trigger GORM soft delete timestamp updates as well
	fc.db.Delete(&models.Blob{}, "id = ?", blobID)

	c.JSON(http.StatusOK, gin.H{
		"message":           "File logically deleted successfully.",
		"blob_id":           blobID,
		"physical_deletion": false,
	})
}
