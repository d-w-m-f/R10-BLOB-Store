package handlers

import (
	"fmt"
	"net/http"
	"strconv"

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

func (h *ChunkHandler) WriteBlockChunk(c *gin.Context) {
	namespace := c.Param("machine_namespace")
	chunkID := c.GetHeader("X-Chunk-ID")
	
	if chunkID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Chunk-ID header is required"})
		return
	}

	sizeStr := c.GetHeader("X-Chunk-Size")
	size, _ := strconv.ParseInt(sizeStr, 10, 64)

	path, offset, err := h.BlockEngine.StreamWrite(namespace, chunkID, c.Request.Body, size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to write block chunk: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":          "success",
		"physical_path":   path,
		"physical_offset": offset,
	})
}

func (h *ChunkHandler) AppendInlineChunk(c *gin.Context) {
	namespace := c.Param("machine_namespace")
	chunkID := c.GetHeader("X-Chunk-ID")
	
	if chunkID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Chunk-ID header is required"})
		return
	}

	sizeStr := c.GetHeader("X-Chunk-Size")
	size, _ := strconv.ParseInt(sizeStr, 10, 64)

	path, offset, err := h.InlineEngine.StreamWrite(namespace, chunkID, c.Request.Body, size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to append inline chunk: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":          "success",
		"physical_path":   path,
		"physical_offset": offset,
	})
}
