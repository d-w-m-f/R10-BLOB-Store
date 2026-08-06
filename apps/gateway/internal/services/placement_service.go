package services

import (
	"errors"
	"math/rand"
	"time"

	"gateway/internal/models"

	"gorm.io/gorm"
)

// PlacementService encapsulates the heuristics for selecting workers
type PlacementService struct {
	db *gorm.DB
}

func NewPlacementService(db *gorm.DB) *PlacementService {
	return &PlacementService{db: db}
}

// SelectWorkersForEC returns exactly 12 distinct workers for Erasure Coding (8+4).
// Heuristic:
// 1. Get Top 16 Block Workers by Free Space (CapacityMB - UsedMB).
// 2. Take the absolute Top 6.
// 3. From the remaining 10, pick 6 randomly.
func (ps *PlacementService) SelectWorkersForEC(requiredMB int64) ([]models.Worker, error) {
	var candidates []models.Worker

	// 1. Filter: Active Workers on Block Machines with enough space
	// Order by free space descending. Limit 16.
	err := ps.db.
		Joins("JOIN machines ON machines.id = workers.machine_id").
		Where("workers.status = ?", models.WorkerStatusActive).
		Where("machines.type = ?", models.MachineTypeBlock).
		Where("(workers.capacity_mb - workers.used_mb) >= ?", requiredMB).
		Order("(workers.capacity_mb - workers.used_mb) DESC").
		Limit(16).
		Find(&candidates).Error

	if err != nil {
		return nil, err
	}

	if len(candidates) < 12 {
		return nil, errors.New("not enough eligible workers for 8+4 placement")
	}

	var selected []models.Worker

	// If we got exactly 12, just return them
	if len(candidates) == 12 {
		return candidates, nil
	}

	// 3. Deterministic selection: Top 6
	selected = append(selected, candidates[:6]...)

	// 4. Random selection: 6 out of the remaining
	remaining := candidates[6:]

	// Create a new random generator
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	
	// Shuffle the remaining slice
	r.Shuffle(len(remaining), func(i, j int) {
		remaining[i], remaining[j] = remaining[j], remaining[i]
	})

	// Pick first 6 from shuffled
	selected = append(selected, remaining[:6]...)

	return selected, nil
}
