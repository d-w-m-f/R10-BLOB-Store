package services

import (
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"

	"gateway/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const clusterRootDir = "/tmp/r10_cluster"
const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// randomString uses the auto-seeded, goroutine-safe generator from math/rand/v2.
// The previous implementation re-seeded from time.Now().UnixNano() on every call,
// which returns the same namespace for calls made inside the same clock tick and
// collides on the unique index of infra.discs.serial_number.
func randomString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = charset[rand.IntN(len(charset))]
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
	Address         string
	MachineType     models.MachineType
	MachineCount    int
	MachineCapacity int64
}

// ClusterTopology is the local simulated cluster: 3 block daemons with 12 machines
// each (enough for a full 8+4 stripe on distinct machines) plus 1 inline daemon.
var ClusterTopology = []WorkerConfig{
	{Name: "wkr10_1", Address: "http://localhost:8081", MachineType: models.MachineTypeBlock, MachineCount: 12, MachineCapacity: 80},
	{Name: "wkr10_2", Address: "http://localhost:8082", MachineType: models.MachineTypeBlock, MachineCount: 12, MachineCapacity: 80},
	{Name: "wkr10_3", Address: "http://localhost:8083", MachineType: models.MachineTypeBlock, MachineCount: 12, MachineCapacity: 80},
	{Name: "wkr10_4", Address: "http://localhost:8084", MachineType: models.MachineTypeInline, MachineCount: 2, MachineCapacity: 80},
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

	if err := s.BuildTopology(); err != nil {
		s.failJob(jobID, err.Error())
		return
	}

	s.db.Model(&models.Job{}).Where("id = ?", jobID).Update("status", models.JobStatusSuccess)
}

// BuildTopology provisions the whole simulated cluster: worker rows with the address
// of the daemon serving them, logical machines with their namespaces, discs, and the
// on-disk machine directories. It is the single source of truth shared by the async
// bootstrap job and the setup_cluster CLI.
func (s *ManagementService) BuildTopology() error {
	// Bootstrap is idempotent: wipe any previous topology first, otherwise the
	// unique index on infra.workers.name aborts the job on a second run.
	if err := s.wipeTopology(); err != nil {
		return fmt.Errorf("failed to wipe previous topology: %w", err)
	}

	if err := os.MkdirAll(clusterRootDir, 0755); err != nil {
		return fmt.Errorf("failed to create cluster root dir: %w", err)
	}

	for _, cfg := range ClusterTopology {
		workerCapacity := cfg.MachineCapacity * int64(cfg.MachineCount)

		worker := models.Worker{
			ID:         uuid.New(),
			Name:       cfg.Name,
			Address:    cfg.Address,
			CapacityMB: workerCapacity,
			UsedMB:     0,
			Status:     models.WorkerStatusActive,
		}

		if err := s.db.Create(&worker).Error; err != nil {
			return fmt.Errorf("failed to create worker %s: %w", cfg.Name, err)
		}

		for i := 0; i < cfg.MachineCount; i++ {
			namespace := randomString(8)
			machineName := fmt.Sprintf("machine_%s", namespace)

			machine := models.Machine{
				ID:        uuid.New(),
				Name:      machineName,
				Namespace: namespace,
				Type:      cfg.MachineType,
				WorkerID:  worker.ID,
			}

			if err := s.db.Create(&machine).Error; err != nil {
				return fmt.Errorf("failed to create machine %s: %w", machineName, err)
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
				return fmt.Errorf("failed to create disc for machine %s: %w", machineName, err)
			}

			machineDir := filepath.Join(clusterRootDir, machineName)
			if err := os.MkdirAll(machineDir, 0755); err != nil {
				return fmt.Errorf("failed to create machine dir %s: %w", machineDir, err)
			}
		}
	}

	return nil
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

	if err := s.wipeTopology(); err != nil {
		s.failJob(jobID, fmt.Sprintf("Failed to truncate topology: %v", err))
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

// wipeTopology removes every stored object and the hardware topology backing it.
// Chunks reference discs and workers, so they have to go first.
func (s *ManagementService) wipeTopology() error {
	global := s.db.Session(&gorm.Session{AllowGlobalUpdate: true})
	for _, model := range []interface{}{
		&models.BlobChunk{}, &models.Blob{}, &models.Disc{}, &models.Machine{}, &models.Worker{},
	} {
		if err := global.Unscoped().Delete(model).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *ManagementService) failJob(jobID uuid.UUID, errMsg string) {
	s.db.Model(&models.Job{}).Where("id = ?", jobID).Updates(map[string]interface{}{
		"status": models.JobStatusFailed,
		"error":  errMsg,
	})
}
