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
	// LogicalSize is how many bytes of the ORIGINAL file the block this chunk belongs to covers.
	// For non-erasure-coded chunks it equals Size.
	LogicalSize    int64     `gorm:"not null" json:"logical_size"`
	// BlockIndex identifies which 32MB logical block of the blob this chunk belongs to.
	BlockIndex     int       `gorm:"not null;index" json:"block_index"`
	// ShardIndex is the Reed-Solomon shard position (0..7 data, 8..11 parity).
	// -1 means the chunk is the whole block, stored without erasure coding.
	//
	// Deliberately declared WITHOUT a gorm `default` tag: GORM omits a field from the
	// INSERT when its value is the Go zero value AND the column declares a default, so
	// `default:-1` silently turned every shard 0 into shard -1 on write.
	ShardIndex     int       `gorm:"not null" json:"shard_index"`
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

