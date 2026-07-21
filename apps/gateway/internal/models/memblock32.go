package models

import (
	"time"

	"github.com/google/uuid"
)

type MemblockStatus string

const (
	MemblockStatusActive  MemblockStatus = "active"
	MemblockStatusSealed  MemblockStatus = "sealed"
	MemblockStatusEncoded MemblockStatus = "encoded"
)

// Memblock32 represents a 32MB logical buffer for Case 2 archives (128KB - 32MB).
// Files are appended to this block until it hits MaxSize.
// Then it is marked "sealed", encoded via Reed-Solomon, and turned into BlobChunks.
type Memblock32 struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"memblock_uuid"`
	CurrentSize int64          `gorm:"not null;default:0" json:"current_size"`
	MaxSize     int64          `gorm:"not null;default:33554432" json:"max_size"` // 32MB default
	Status      MemblockStatus `gorm:"type:varchar(50);not null;default:'active';index" json:"status"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`

	// Blobs that are temporarily stored inside this block before encoding
	Blobs []Blob `gorm:"foreignKey:Memblock32ID" json:"blobs,omitempty"`
}
