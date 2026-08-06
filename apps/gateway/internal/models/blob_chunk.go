package models

import (
	"github.com/google/uuid"
)

// BlobChunk represents a physical chunk of a Blob stored in a specific Disc/Worker.
type BlobChunk struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"blob_chunk_id"`
	BlobID   uuid.UUID `gorm:"type:uuid;not null;index" json:"blob_id"`
	DiscID   uuid.UUID `gorm:"type:uuid;not null;index" json:"disc_id"`
	WorkerID uuid.UUID `gorm:"type:uuid;not null;index" json:"worker_id"`
	Checksum string    `gorm:"type:varchar(255);not null" json:"blob_checksum"`
	Size           int64     `gorm:"not null" json:"blob_size"`
	LogicalOffset  int64     `gorm:"not null" json:"logical_offset"` // Posicao logica no arquivo original
	PhysicalPath   string    `gorm:"type:varchar(255);not null" json:"physical_path"`
	PhysicalOffset int64     `gorm:"not null" json:"physical_offset"`

	// Relationships
	Blob   *Blob   `gorm:"foreignKey:BlobID" json:"blob,omitempty"`
	Disc   *Disc   `gorm:"foreignKey:DiscID" json:"disc,omitempty"`
	Worker *Worker `gorm:"foreignKey:WorkerID" json:"worker,omitempty"`
}

// TableName overrides the default table name to specify the DDD schema namespace.
func (BlobChunk) TableName() string {
	return "storage.blob_chunks"
}

