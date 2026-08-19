package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type WorkerStatus string

const (
	WorkerStatusActive   WorkerStatus = "active"
	WorkerStatusInactive WorkerStatus = "inactive"
)

// Worker represents a storage node process running on a Machine.
type Worker struct {
	ID         uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"worker_uuid"`
	Name       string         `gorm:"type:varchar(255);not null;uniqueIndex" json:"worker_name"`
	// Address is the base URL of the wkr10 daemon that serves this worker's machines.
	Address    string         `gorm:"type:varchar(255);not null;default:''" json:"worker_address"`
	CapacityMB int64          `gorm:"not null" json:"worker_capacity_mb"`
	UsedMB     int64          `gorm:"not null;default:0" json:"worker_used_mb"`
	Status     WorkerStatus   `gorm:"type:varchar(50);not null;default:'active'" json:"worker_status"`
	CreatedAt  time.Time      `json:"worker_created_at"`
	UpdatedAt  time.Time      `json:"worker_updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Machines   []Machine   `gorm:"foreignKey:WorkerID" json:"machines,omitempty"`
	BlobChunks []BlobChunk `gorm:"foreignKey:WorkerID" json:"blob_chunks,omitempty"`
}

// TableName overrides the default table name to specify the DDD schema namespace.
func (Worker) TableName() string {
	return "infra.workers"
}

