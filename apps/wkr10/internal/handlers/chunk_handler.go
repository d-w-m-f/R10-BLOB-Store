package handlers

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"wkr10/internal/io"

	"github.com/gin-gonic/gin"
)

type ChunkHandler struct {
	BlockEngine  *io.BlockEngine
	InlineEngine *io.InlineEngine
}

func NewChunkHandler(clusterRootDir string) *ChunkHandler {
	return &ChunkHandler{
		BlockEngine:  io.NewBlockEngine(clusterRootDir),
		InlineEngine: io.NewInlineEngine(clusterRootDir),
	}
}

// parseChunkSize validates X-Chunk-Size. It used to be parsed with the error
// discarded, so a missing header silently became size 0 and the write was accepted
// without any way to notice a truncated body.
func parseChunkSize(raw string) (int64, error) {
	if raw == "" {
		return 0, fmt.Errorf("X-Chunk-Size header is required")
	}
	size, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || size < 0 {
		return 0, fmt.Errorf("X-Chunk-Size must be a non-negative integer, got %q", raw)
	}
	return size, nil
}

func (h *ChunkHandler) WriteBlockChunk(c *gin.Context) {
	namespace := c.Param("machine_namespace")
	chunkID := c.GetHeader("X-Chunk-ID")
	
	if chunkID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Chunk-ID header is required"})
		return
	}

	size, err := parseChunkSize(c.GetHeader("X-Chunk-Size"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	path, offset, written, err := h.BlockEngine.StreamWrite(namespace, chunkID, c.Request.Body, size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to write block chunk: %v", err)})
		return
	}
	if written != size {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("truncated transfer: declared %d bytes, received %d", size, written),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":          "success",
		"physical_path":   path,
		"physical_offset": offset,
		"bytes_written":   written,
	})
}

func (h *ChunkHandler) AppendInlineChunk(c *gin.Context) {
	namespace := c.Param("machine_namespace")
	chunkID := c.GetHeader("X-Chunk-ID")
	
	if chunkID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Chunk-ID header is required"})
		return
	}

	size, err := parseChunkSize(c.GetHeader("X-Chunk-Size"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	path, offset, written, err := h.InlineEngine.StreamWrite(namespace, chunkID, c.Request.Body, size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to append inline chunk: %v", err)})
		return
	}
	if written != size {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("truncated transfer: declared %d bytes, received %d", size, written),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":          "success",
		"physical_path":   path,
		"physical_offset": offset,
		"bytes_written":   written,
	})
}

// ReadChunk streams back a byte range previously written by either engine.
// Both engines address data the same way (relative path + offset + size), so a
// single endpoint serves block chunks and inline volume slices alike.
func (h *ChunkHandler) ReadChunk(c *gin.Context) {
	namespace := c.Param("machine_namespace")

	physicalPath := c.Query("path")
	if physicalPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path query parameter is required"})
		return
	}

	// Never let a caller escape the machine directory via ../ segments.
	cleaned := filepath.Clean(physicalPath)
	if filepath.IsAbs(cleaned) || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
		return
	}

	offset, err := strconv.ParseInt(c.DefaultQuery("offset", "0"), 10, 64)
	if err != nil || offset < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid offset"})
		return
	}

	size, err := strconv.ParseInt(c.Query("size"), 10, 64)
	if err != nil || size <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "size query parameter must be a positive integer"})
		return
	}

	data, err := h.BlockEngine.Read(namespace, cleaned, offset, size)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("failed to read chunk: %v", err)})
		return
	}

	if int64(len(data)) != size {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("short read: wanted %d bytes, got %d", size, len(data)),
		})
		return
	}

	c.Data(http.StatusOK, "application/octet-stream", data)
}
