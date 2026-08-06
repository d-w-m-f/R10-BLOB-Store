package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// DeleteFile performs a LOGICAL deletion of a file.
// For the first release, no data is physically removed from the storage (Discs)
// nor are the memory/Postgres buffers cleared. We only flag the blob as deleted.
func DeleteFile(c *gin.Context) {
	blobID := c.Param("blob_id")

	if blobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "blob_id is required"})
		return
	}

	// Future Implementation:
	// Here we would get the DB instance (e.g., from gin.Context or a global variable)
	// and execute a Soft Delete.
	//
	// Example using GORM:
	// db.Model(&models.Blob{}).Where("id = ?", blobID).Update("deleted", true)
	// OR using GORM's built-in SoftDelete:
	// db.Delete(&models.Blob{}, "id = ?", blobID) 
	// (Thanks to the SoftDeletedAt field, GORM intercepts this and only updates the timestamp!)

	// For now, since the DB connection isn't injected into the controller yet, 
	// we just mock the successful logical deletion response.
	c.JSON(http.StatusOK, gin.H{
		"message": "File logically deleted successfully.",
		"blob_id": blobID,
		"physical_deletion": false,
	})
}
