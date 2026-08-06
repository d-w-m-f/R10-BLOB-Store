package models

import (
	"time"

	"github.com/google/uuid"
)

// Backup represents a replication operation from one Disc to another.
type Backup struct {
	ID                   uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"backup_id"`
	SerialDiscCopiedFrom string    `gorm:"type:varchar(255);not null;index" json:"serial_disc_copied_from"`
	SerialDiscCopiedTo   string    `gorm:"type:varchar(255);not null;index" json:"serial_disc_copied_to"`
	CreatedAt            time.Time `json:"backup_created_at"`
	UpdatedAt            time.Time `json:"backup_updated_at"`
}

// TableName overrides the default table name to specify the DDD schema namespace.
func (Backup) TableName() string {
	return "control_plane.backups"
}

