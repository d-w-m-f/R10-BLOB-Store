package controllers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type InitUploadRequest struct {
	Filename    string `json:"filename" binding:"required"`
	TotalSize   int64  `json:"total_size" binding:"required"`
	ContentType string `json:"content_type"`
}

type InitUploadResponse struct {
	UploadID string `json:"upload_id"`
}

// InitUpload starts the chunked upload process
func InitUpload(c *gin.Context) {
	var req InitUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	uploadID := uuid.New().String()

	// In a real scenario, we would register this upload intent in the Database
	// And verify the user's quota, etc.

	// Create staging directory for this upload
	stagingDir := filepath.Join(os.TempDir(), "r10_uploads", uploadID)
	if err := os.MkdirAll(stagingDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create staging directory"})
		return
	}

	c.JSON(http.StatusOK, InitUploadResponse{UploadID: uploadID})
}

// UploadPart receives a chunk of the file
func UploadPart(c *gin.Context) {
	uploadID := c.Param("upload_id")
	partNumber := c.Param("part_number")

	// Create the specific file for this part in the staging directory
	partPath := filepath.Join(os.TempDir(), "r10_uploads", uploadID, fmt.Sprintf("part_%s", partNumber))

	out, err := os.Create(partPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create part file"})
		return
	}
	defer out.Close()

	// Stream the incoming request body directly to disk
	// This uses very little RAM!
	written, err := io.Copy(out, c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to stream part data"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Part uploaded successfully",
		"part_number":  partNumber,
		"bytes_written": written,
	})
}

// CompleteUpload finishes the chunked upload and triggers processing
func CompleteUpload(c *gin.Context) {
	uploadID := c.Param("upload_id")

	// Verify if the staging directory exists
	stagingDir := filepath.Join(os.TempDir(), "r10_uploads", uploadID)
	if _, err := os.Stat(stagingDir); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Upload ID not found or expired"})
		return
	}

	// Future: Trigger the goroutine to assemble the file, check total size,
	// decide if it goes to Memblock32 (Postgres) or direct Erasure Coding,
	// and eventually clean up the stagingDir.

	c.JSON(http.StatusOK, gin.H{
		"message": "Upload completed successfully and sent to processing queue.",
		"upload_id": uploadID,
	})
}
