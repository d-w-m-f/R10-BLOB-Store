package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User represents an owner of Blobs.
type User struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"user_uuid"`
	Email     string         `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	Name      string         `gorm:"type:varchar(255);not null" json:"name"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Blobs []Blob `gorm:"foreignKey:OwnerID" json:"blobs,omitempty"`
}
