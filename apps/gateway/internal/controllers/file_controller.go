package controllers

import (
	"fmt"
	"net/http"

	"gateway/internal/models"
	"gateway/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FileController struct {
	db      *gorm.DB
	storage *services.StorageService
}

func NewFileController(db *gorm.DB, storage *services.StorageService) *FileController {
	return &FileController{db: db, storage: storage}
}

// findBlob loads a live (non-deleted) blob by id.
func (fc *FileController) findBlob(c *gin.Context) (*models.Blob, bool) {
	blobID, err := uuid.Parse(c.Param("blob_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "blob_id must be a valid uuid"})
		return nil, false
	}

	var blob models.Blob
	if err := fc.db.Where("id = ? AND deleted = ?", blobID, false).First(&blob).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
			return nil, false
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return nil, false
	}
	return &blob, true
}

// GetFile returns a single blob's catalog entry together with its chunk placement.
func (fc *FileController) GetFile(c *gin.Context) {
	blob, ok := fc.findBlob(c)
	if !ok {
		return
	}

	if err := fc.db.Preload("BlobChunks").First(blob, "id = ?", blob.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, blob)
}

// DownloadFile rebuilds the blob from its chunks (reconstructing missing shards via
// Reed-Solomon when needed) and streams the original bytes back to the client.
func (fc *FileController) DownloadFile(c *gin.Context) {
	blob, ok := fc.findBlob(c)
	if !ok {
		return
	}

	data, err := fc.storage.RetrieveBlob(blob)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	contentType := blob.MimeType
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", blob.Filename))
	c.Header("X-Blob-Checksum", blob.Checksum)
	c.Data(http.StatusOK, contentType, data)
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
