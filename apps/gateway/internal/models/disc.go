package models

import (
	"github.com/google/uuid"
)

type DiscStatus string

const (
	DiscStatusActive   DiscStatus = "active"
	DiscStatusInactive DiscStatus = "inactive"
)

// Disc represents a physical storage device attached to a Machine.
type Disc struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"disc_uuid"`
	SerialNumber string     `gorm:"type:varchar(255);uniqueIndex;not null" json:"disc_serial_number"`
	MachineID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"disc_machine_id"`
	CapacityMB   int64      `gorm:"not null" json:"disc_capacity_mb"`
	UsedMB       int64      `gorm:"not null;default:0" json:"disc_used_mb"`
	UsedBytes    int64      `gorm:"not null;default:0" json:"disc_used_bytes"`
	Status       DiscStatus `gorm:"type:varchar(50);not null;default:'active'" json:"disc_status"`

	// Relationships
	Machine     *Machine     `gorm:"foreignKey:MachineID" json:"machine,omitempty"`
	BlobChunks  []BlobChunk  `gorm:"foreignKey:DiscID" json:"blob_chunks,omitempty"`
}

// TableName overrides the default table name to specify the DDD schema namespace.
func (Disc) TableName() string {
	return "infra.discs"
}

