package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Blob represents a complete file stored in the R10 system.
type Blob struct {
	ID            uuid.UUID       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"blob_uuid"`
	OwnerID       uuid.UUID       `gorm:"type:uuid;not null;index" json:"blob_owner_id"`
	Memblock32ID  *uuid.UUID      `gorm:"type:uuid;index" json:"memblock32_id,omitempty"`
	Size          int64           `gorm:"not null" json:"blob_size"`
	Checksum      string          `gorm:"type:varchar(255);not null" json:"blob_checksum"`
	ChecksumAlg   string          `gorm:"type:varchar(50);not null" json:"blob_checksum_alg"`
	MimeType      string          `gorm:"type:varchar(255)" json:"blob_mime_type"`
	Filename      string          `gorm:"type:varchar(1024);not null" json:"blob_filename"`
	OldMetadata   json.RawMessage `gorm:"type:jsonb" json:"blob_old_metadata,omitempty"`
	Payload       []byte          `gorm:"type:bytea" json:"-"` // Temporary or inline storage
	CreatedAt     time.Time       `json:"blob_created_at"`
	UpdatedAt     time.Time       `json:"blob_updated_at"`
	Deleted       bool            `gorm:"not null;default:false" json:"blob_deleted"`
	SoftDeletedAt gorm.DeletedAt  `gorm:"index" json:"blob_deleted_at,omitempty"` // renamed to avoid conflict with 'Deleted' flag

	// Relationships
	Owner      *User       `gorm:"foreignKey:OwnerID" json:"owner,omitempty"`
	Memblock32 *Memblock32 `gorm:"foreignKey:Memblock32ID" json:"memblock32,omitempty"`
	BlobChunks []BlobChunk `gorm:"foreignKey:BlobID" json:"blob_chunks,omitempty"`
}
