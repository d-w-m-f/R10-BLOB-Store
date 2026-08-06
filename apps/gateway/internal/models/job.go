package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type JobStatus string

const (
	JobStatusPending JobStatus = "pending"
	JobStatusRunning JobStatus = "running"
	JobStatusSuccess JobStatus = "success"
	JobStatusFailed  JobStatus = "failed"
)

type JobType string

const (
	JobTypeBootstrap JobType = "bootstrap"
	JobTypeReset     JobType = "reset"
)

// Job represents an asynchronous background task
type Job struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"job_uuid"`
	Type      JobType        `gorm:"type:varchar(50);not null" json:"job_type"`
	Status    JobStatus      `gorm:"type:varchar(50);not null;default:'pending'" json:"job_status"`
	Error     string         `gorm:"type:text" json:"job_error,omitempty"`
	CreatedAt time.Time      `json:"job_created_at"`
	UpdatedAt time.Time      `json:"job_updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName overrides the default table name to specify the DDD schema namespace.
func (Job) TableName() string {
	return "control_plane.jobs"
}

