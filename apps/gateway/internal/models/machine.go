package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MachineType string

const (
	MachineTypeBlock  MachineType = "block"
	MachineTypeInline MachineType = "inline"
)

// Machine represents a physical or virtual node running storage workers.
type Machine struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"machine_uuid"`
	Name      string         `gorm:"type:varchar(255);not null" json:"machine_name"`
	Type      MachineType    `gorm:"type:varchar(50);not null;default:'block'" json:"machine_type"`
	WorkerID  uuid.UUID      `gorm:"type:uuid;not null;index" json:"machine_worker_id"`
	CreatedAt time.Time      `json:"machine_created_at"`
	UpdatedAt time.Time      `json:"machine_updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Worker *Worker `gorm:"foreignKey:WorkerID" json:"worker,omitempty"`
	Discs  []Disc  `gorm:"foreignKey:MachineID" json:"discs,omitempty"`
}

// TableName overrides the default table name to specify the DDD schema namespace.
func (Machine) TableName() string {
	return "infra.machines"
}

