package controllers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"gateway/internal/models"
	"gateway/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const manifestFilename = "manifest.json"

// defaultOwnerEmail owns every blob until authentication is wired up.
const defaultOwnerEmail = "system@r10.local"

type InitUploadRequest struct {
	Filename    string `json:"filename" binding:"required"`
	TotalSize   int64  `json:"total_size" binding:"required"`
	ContentType string `json:"content_type"`
}

type InitUploadResponse struct {
	UploadID string `json:"upload_id"`
}

// uploadManifest persists the intent declared at init so that CompleteUpload still
// knows the filename, size and content type. Without it the completion handler had
// no way to name the blob it was assembling.
type uploadManifest struct {
	UploadID    string `json:"upload_id"`
	Filename    string `json:"filename"`
	TotalSize   int64  `json:"total_size"`
	ContentType string `json:"content_type"`
}

type UploadController struct {
	db      *gorm.DB
	storage *services.StorageService
}

func NewUploadController(db *gorm.DB, storage *services.StorageService) *UploadController {
	return &UploadController{db: db, storage: storage}
}

func stagingRoot() string {
	return filepath.Join(os.TempDir(), "r10_uploads")
}

// stagingDir resolves an upload's staging directory, rejecting ids that try to
// escape the staging root through path separators or traversal segments.
func stagingDir(uploadID string) (string, error) {
	if _, err := uuid.Parse(uploadID); err != nil {
		return "", fmt.Errorf("invalid upload id")
	}
	return filepath.Join(stagingRoot(), uploadID), nil
}

// InitUpload starts the chunked upload process.
func (uc *UploadController) InitUpload(c *gin.Context) {
	var req InitUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	uploadID := uuid.New().String()
	dir, err := stagingDir(uploadID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create staging directory"})
		return
	}

	manifest := uploadManifest{
		UploadID:    uploadID,
		Filename:    filepath.Base(req.Filename),
		TotalSize:   req.TotalSize,
		ContentType: req.ContentType,
	}
	payload, err := json.Marshal(manifest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encode upload manifest"})
		return
	}
	if err := os.WriteFile(filepath.Join(dir, manifestFilename), payload, 0644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to persist upload manifest"})
		return
	}

	c.JSON(http.StatusOK, InitUploadResponse{UploadID: uploadID})
}

// UploadPart receives a chunk of the file.
func (uc *UploadController) UploadPart(c *gin.Context) {
	dir, err := stagingDir(c.Param("upload_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	partNumber, err := strconv.Atoi(c.Param("part_number"))
	if err != nil || partNumber < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "part_number must be a positive integer"})
		return
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Upload ID not found or expired"})
		return
	}

	// Zero-padded so the parts sort correctly on disk: with the old "part_%s"
	// naming, part_10 sorted before part_2 and the file was reassembled scrambled.
	partPath := filepath.Join(dir, fmt.Sprintf("part_%09d", partNumber))

	out, err := os.Create(partPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create part file"})
		return
	}
	defer out.Close()

	// Stream the incoming request body directly to disk. This uses very little RAM!
	written, err := io.Copy(out, c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to stream part data"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Part uploaded successfully",
		"part_number":   partNumber,
		"bytes_written": written,
	})
}

// CompleteUpload assembles the staged parts, pushes them into the cluster and
// registers the resulting blob in the catalog.
func (uc *UploadController) CompleteUpload(c *gin.Context) {
	uploadID := c.Param("upload_id")
	dir, err := stagingDir(uploadID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Upload ID not found or expired"})
		return
	}

	manifest, err := readManifest(dir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	assembledPath := filepath.Join(dir, "assembled.bin")
	assembledSize, err := assembleParts(dir, assembledPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// The client told us up front how many bytes to expect; a mismatch means a part
	// was lost or duplicated, and silently storing it would corrupt the blob.
	if manifest.TotalSize > 0 && assembledSize != manifest.TotalSize {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("incomplete upload: assembled %d bytes, expected %d", assembledSize, manifest.TotalSize),
		})
		return
	}

	ownerID, err := uc.defaultOwnerID()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to resolve owner: %v", err)})
		return
	}

	blob, err := uc.storage.StoreFile(ownerID, manifest.Filename, manifest.ContentType, assembledPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Staging is scratch space; release it as soon as the bytes are durable.
	if err := os.RemoveAll(dir); err != nil {
		fmt.Printf("warning: failed to clean staging dir %s: %v\n", dir, err)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Upload completed successfully and stored in the cluster.",
		"upload_id": uploadID,
		"blob":      blob,
	})
}

func readManifest(dir string) (uploadManifest, error) {
	var manifest uploadManifest
	raw, err := os.ReadFile(filepath.Join(dir, manifestFilename))
	if err != nil {
		return manifest, fmt.Errorf("upload manifest missing or unreadable: %w", err)
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return manifest, fmt.Errorf("upload manifest is corrupt: %w", err)
	}
	if manifest.Filename == "" {
		manifest.Filename = "untitled"
	}
	return manifest, nil
}

// assembleParts concatenates part_* files in numeric order into destPath.
func assembleParts(dir, destPath string) (int64, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, fmt.Errorf("failed to list staging dir: %w", err)
	}

	var parts []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), "part_") {
			parts = append(parts, entry.Name())
		}
	}
	if len(parts) == 0 {
		return 0, fmt.Errorf("no parts were uploaded for this upload id")
	}
	sort.Strings(parts)

	dest, err := os.Create(destPath)
	if err != nil {
		return 0, fmt.Errorf("failed to create assembled file: %w", err)
	}
	defer dest.Close()

	var total int64
	for _, name := range parts {
		src, err := os.Open(filepath.Join(dir, name))
		if err != nil {
			return 0, fmt.Errorf("failed to open part %s: %w", name, err)
		}
		n, err := io.Copy(dest, src)
		src.Close()
		if err != nil {
			return 0, fmt.Errorf("failed to append part %s: %w", name, err)
		}
		total += n
	}

	if err := dest.Sync(); err != nil {
		return 0, fmt.Errorf("failed to flush assembled file: %w", err)
	}
	return total, nil
}

// defaultOwnerID returns the system user, creating it on first use. Blob.OwnerID is
// NOT NULL, so an upload with no authenticated user would otherwise fail to insert.
func (uc *UploadController) defaultOwnerID() (uuid.UUID, error) {
	var user models.User
	err := uc.db.Where("email = ?", defaultOwnerEmail).First(&user).Error
	if err == nil {
		return user.ID, nil
	}
	if err != gorm.ErrRecordNotFound {
		return uuid.Nil, err
	}

	user = models.User{ID: uuid.New(), Email: defaultOwnerEmail, Name: "R10 System"}
	if err := uc.db.Create(&user).Error; err != nil {
		// Lost a race with a concurrent upload; re-read the winner.
		if lookupErr := uc.db.Where("email = ?", defaultOwnerEmail).First(&user).Error; lookupErr == nil {
			return user.ID, nil
		}
		return uuid.Nil, err
	}
	return user.ID, nil
}
