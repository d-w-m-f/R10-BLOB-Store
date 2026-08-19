package services

import (
	"errors"
	"fmt"
	"math/rand/v2"

	"gateway/internal/models"

	"gorm.io/gorm"
)

// Target is a resolved placement destination: one logical machine, the disc that
// backs it and the daemon that serves it.
type Target struct {
	MachineID   string `gorm:"column:machine_id"`
	Namespace   string `gorm:"column:namespace"`
	MachineType string `gorm:"column:machine_type"`
	DiscID      string `gorm:"column:disc_id"`
	WorkerID    string `gorm:"column:worker_id"`
	WorkerAddr  string `gorm:"column:worker_addr"`
	FreeMB      int64  `gorm:"column:free_mb"`
}

// PlacementService encapsulates the heuristics for selecting storage targets.
type PlacementService struct {
	db *gorm.DB
}

func NewPlacementService(db *gorm.DB) *PlacementService {
	return &PlacementService{db: db}
}

// candidates lists every active machine of the given type with at least requiredMB
// of free space on its disc, ordered by free space descending.
//
// Placement is per MACHINE, not per worker: a wkr10 daemon multiplexes many logical
// machines, so the cluster has only 4 workers but 38 machines. The previous version
// joined `machines.id = workers.machine_id` (a column that does not exist) and asked
// for 12 distinct workers, so every placement failed.
func (ps *PlacementService) candidates(machineType models.MachineType, requiredMB int64) ([]Target, error) {
	var out []Target
	err := ps.db.Model(&models.Machine{}).
		Select(`machines.id AS machine_id,
		        machines.namespace AS namespace,
		        machines.type AS machine_type,
		        discs.id AS disc_id,
		        workers.id AS worker_id,
		        workers.address AS worker_addr,
		        (discs.capacity_mb - discs.used_mb) AS free_mb`).
		Joins("JOIN infra.discs discs ON discs.machine_id = machines.id").
		Joins("JOIN infra.workers workers ON workers.id = machines.worker_id").
		Where("machines.deleted_at IS NULL").
		Where("workers.deleted_at IS NULL").
		Where("workers.status = ?", models.WorkerStatusActive).
		Where("discs.status = ?", models.DiscStatusActive).
		Where("machines.type = ?", machineType).
		Where("(discs.capacity_mb - discs.used_mb) >= ?", requiredMB).
		Order("(discs.capacity_mb - discs.used_mb) DESC").
		Scan(&out).Error
	if err != nil {
		return nil, err
	}
	return out, nil
}

// SelectMachinesForEC returns exactly `count` DISTINCT block machines for one
// erasure-coded stripe. Heuristic: take the emptiest half deterministically, then
// pick the rest at random from the tail so uploads do not all pile onto the same
// machines.
func (ps *PlacementService) SelectMachinesForEC(count int, requiredMB int64) ([]Target, error) {
	candidates, err := ps.candidates(models.MachineTypeBlock, requiredMB)
	if err != nil {
		return nil, err
	}
	if len(candidates) < count {
		return nil, fmt.Errorf("not enough eligible block machines for placement: need %d, have %d (is the cluster bootstrapped?)", count, len(candidates))
	}
	if len(candidates) == count {
		return candidates, nil
	}

	head := count / 2
	selected := append([]Target{}, candidates[:head]...)

	tail := append([]Target{}, candidates[head:]...)
	rand.Shuffle(len(tail), func(i, j int) { tail[i], tail[j] = tail[j], tail[i] })

	return append(selected, tail[:count-head]...), nil
}

// SelectMachine returns the single emptiest machine of the given type.
func (ps *PlacementService) SelectMachine(machineType models.MachineType, requiredMB int64) (Target, error) {
	candidates, err := ps.candidates(machineType, requiredMB)
	if err != nil {
		return Target{}, err
	}
	if len(candidates) == 0 {
		return Target{}, errors.New("no eligible " + string(machineType) + " machine with enough free space (is the cluster bootstrapped?)")
	}
	return candidates[0], nil
}
