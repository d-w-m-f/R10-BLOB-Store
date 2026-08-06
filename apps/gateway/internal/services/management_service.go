package services

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"gateway/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const clusterRootDir = "/tmp/r10_cluster"
const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func randomString(n int) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, n)
	for i := range b {
		b[i] = charset[r.Intn(len(charset))]
	}
	return string(b)
}

type ManagementService struct {
	db *gorm.DB
}

func NewManagementService(db *gorm.DB) *ManagementService {
	return &ManagementService{db: db}
}

type WorkerConfig struct {
	Name            string
	MachineType     models.MachineType
	MachineCount    int
	MachineCapacity int64
}

// BootstrapCluster triggers the bootstrap process asynchronously and returns the Job ID.
func (s *ManagementService) BootstrapCluster() (uuid.UUID, error) {
	job := models.Job{
		ID:     uuid.New(),
		Type:   models.JobTypeBootstrap,
		Status: models.JobStatusPending,
	}

	if err := s.db.Create(&job).Error; err != nil {
		return uuid.Nil, err
	}

	go s.runBootstrap(job.ID)

	return job.ID, nil
}

func (s *ManagementService) runBootstrap(jobID uuid.UUID) {
	// Mark as running
	s.db.Model(&models.Job{}).Where("id = ?", jobID).Update("status", models.JobStatusRunning)

	if err := os.MkdirAll(clusterRootDir, 0755); err != nil {
		s.failJob(jobID, fmt.Sprintf("Failed to create cluster root dir: %v", err))
		return
	}

	configs := []WorkerConfig{
		{Name: "wkr10_1", MachineType: models.MachineTypeBlock, MachineCount: 12, MachineCapacity: 80},
		{Name: "wkr10_2", MachineType: models.MachineTypeBlock, MachineCount: 12, MachineCapacity: 80},
		{Name: "wkr10_3", MachineType: models.MachineTypeBlock, MachineCount: 12, MachineCapacity: 80},
		{Name: "wkr10_4", MachineType: models.MachineTypeInline, MachineCount: 2, MachineCapacity: 80},
	}

	for _, cfg := range configs {
		workerCapacity := cfg.MachineCapacity * int64(cfg.MachineCount)

		worker := models.Worker{
			ID:         uuid.New(),
			Name:       cfg.Name,
			CapacityMB: workerCapacity,
			UsedMB:     0,
			Status:     models.WorkerStatusActive,
		}

		if err := s.db.Create(&worker).Error; err != nil {
			s.failJob(jobID, fmt.Sprintf("Failed to create worker %s: %v", cfg.Name, err))
			return
		}

		for i := 0; i < cfg.MachineCount; i++ {
			namespace := randomString(8)
			machineName := fmt.Sprintf("machine_%s", namespace)

			machine := models.Machine{
				ID:       uuid.New(),
				Name:     machineName,
				Type:     cfg.MachineType,
				WorkerID: worker.ID,
			}

			if err := s.db.Create(&machine).Error; err != nil {
				s.failJob(jobID, fmt.Sprintf("Failed to create machine %s: %v", machineName, err))
				return
			}

			disc := models.Disc{
				ID:           uuid.New(),
				SerialNumber: fmt.Sprintf("SN-R10-LOCAL-%s", namespace),
				MachineID:    machine.ID,
				CapacityMB:   cfg.MachineCapacity,
				UsedMB:       0,
				Status:       models.DiscStatusActive,
			}

			if err := s.db.Create(&disc).Error; err != nil {
				s.failJob(jobID, fmt.Sprintf("Failed to create disc for machine %s: %v", machineName, err))
				return
			}

			machineDir := filepath.Join(clusterRootDir, machineName)
			if err := os.MkdirAll(machineDir, 0755); err != nil {
				s.failJob(jobID, fmt.Sprintf("Failed to create machine dir %s: %v", machineDir, err))
				return
			}
		}
	}

	s.db.Model(&models.Job{}).Where("id = ?", jobID).Update("status", models.JobStatusSuccess)
}

// ResetCluster triggers the reset process asynchronously and returns the Job ID.
func (s *ManagementService) ResetCluster() (uuid.UUID, error) {
	job := models.Job{
		ID:     uuid.New(),
		Type:   models.JobTypeReset,
		Status: models.JobStatusPending,
	}

	if err := s.db.Create(&job).Error; err != nil {
		return uuid.Nil, err
	}

	go s.runReset(job.ID)

	return job.ID, nil
}

func (s *ManagementService) runReset(jobID uuid.UUID) {
	s.db.Model(&models.Job{}).Where("id = ?", jobID).Update("status", models.JobStatusRunning)

	if err := s.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&models.Disc{}).Error; err != nil {
		s.failJob(jobID, fmt.Sprintf("Failed to truncate discs: %v", err))
		return
	}
	if err := s.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&models.Machine{}).Error; err != nil {
		s.failJob(jobID, fmt.Sprintf("Failed to truncate machines: %v", err))
		return
	}
	if err := s.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&models.Worker{}).Error; err != nil {
		s.failJob(jobID, fmt.Sprintf("Failed to truncate workers: %v", err))
		return
	}

	if err := os.RemoveAll(clusterRootDir); err != nil {
		// Log error but don't fail the job if it just doesn't exist
		if !os.IsNotExist(err) {
			s.failJob(jobID, fmt.Sprintf("Failed to delete physical cluster directories: %v", err))
			return
		}
	}

	s.db.Model(&models.Job{}).Where("id = ?", jobID).Update("status", models.JobStatusSuccess)
}

func (s *ManagementService) failJob(jobID uuid.UUID, errMsg string) {
	s.db.Model(&models.Job{}).Where("id = ?", jobID).Updates(map[string]interface{}{
		"status": models.JobStatusFailed,
		"error":  errMsg,
	})
}
