package services

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"

	"gateway/internal/clients"
	"gateway/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	// InlineThreshold: below this a whole file lives inline on an inline machine (Case 1).
	InlineThreshold = 128 * 1024
	// BlockSize is the logical block the Reed-Solomon stripe operates on (Case 3).
	BlockSize = 32 * 1024 * 1024

	DataShards   = 8
	ParityShards = 4
	TotalShards  = DataShards + ParityShards

	// ShardIndexWhole marks a chunk that holds a whole block verbatim (no erasure coding).
	ShardIndexWhole = -1

	bytesPerMB = 1024 * 1024
)

// StorageService turns a staged file into durable chunks spread over the cluster,
// and puts them back together on the way out.
type StorageService struct {
	db        *gorm.DB
	placement *PlacementService
	erasure   *ErasureService
	workers   *clients.WorkerClient
}

func NewStorageService(db *gorm.DB) (*StorageService, error) {
	erasure, err := NewErasureService(DataShards, ParityShards)
	if err != nil {
		return nil, err
	}
	return &StorageService{
		db:        db,
		placement: NewPlacementService(db),
		erasure:   erasure,
		workers:   clients.NewWorkerClient(),
	}, nil
}

// chunkLocation is a BlobChunk joined with everything needed to reach its bytes.
type chunkLocation struct {
	models.BlobChunk
	Namespace  string `gorm:"column:namespace"`
	WorkerAddr string `gorm:"column:worker_addr"`
}

// -----------------------------------------------------------------------------
// Write path
// -----------------------------------------------------------------------------

// StoreFile ingests a staged file and returns the persisted Blob.
//
// Routing follows docs/erasure-coding.md:
//   - < 128KB          -> Case 1: stored whole on an inline machine's append-only volume
//   - 128KB .. 32MB    -> Case 2: stored whole as one chunk on a block machine
//   - > 32MB           -> Case 3: full 32MB blocks are Reed-Solomon 8+4 encoded across
//     12 distinct block machines; the trailing partial block falls back to Case 2.
func (ss *StorageService) StoreFile(ownerID uuid.UUID, filename, mimeType, stagedPath string) (*models.Blob, error) {
	file, err := os.Open(stagedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open staged file: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat staged file: %w", err)
	}
	totalSize := stat.Size()
	if totalSize == 0 {
		return nil, errors.New("refusing to store an empty file")
	}

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return nil, fmt.Errorf("failed to checksum staged file: %w", err)
	}
	blobChecksum := hex.EncodeToString(hasher.Sum(nil))

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to rewind staged file: %w", err)
	}

	blobID := uuid.New()
	var chunks []models.BlobChunk

	if totalSize < InlineThreshold {
		payload, err := io.ReadAll(file)
		if err != nil {
			return nil, fmt.Errorf("failed to read staged file: %w", err)
		}
		chunk, err := ss.storeWholeBlock(blobID, models.MachineTypeInline, payload, 0, 0)
		if err != nil {
			return nil, err
		}
		chunks = append(chunks, *chunk)
	} else {
		buf := make([]byte, BlockSize)
		var logicalOffset int64
		for blockIndex := 0; logicalOffset < totalSize; blockIndex++ {
			n, err := io.ReadFull(file, buf)
			if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
				return nil, fmt.Errorf("failed to read block %d: %w", blockIndex, err)
			}
			if n == 0 {
				break
			}
			block := buf[:n]

			if n == BlockSize {
				stripe, err := ss.storeErasureBlock(blobID, block, blockIndex, logicalOffset)
				if err != nil {
					return nil, err
				}
				chunks = append(chunks, stripe...)
			} else {
				// Trailing partial block (and any file <= 32MB) is a Case 2 chunk.
				chunk, err := ss.storeWholeBlock(blobID, models.MachineTypeBlock, block, blockIndex, logicalOffset)
				if err != nil {
					return nil, err
				}
				chunks = append(chunks, *chunk)
			}
			logicalOffset += int64(n)
		}
	}

	blob := models.Blob{
		ID:          blobID,
		OwnerID:     ownerID,
		Size:        totalSize,
		Checksum:    blobChecksum,
		ChecksumAlg: "sha256",
		MimeType:    mimeType,
		Filename:    filename,
	}

	// Metadata is committed only once every byte is durably on a worker, so a
	// failed upload never leaves a listable-but-unreadable blob behind.
	err = ss.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&blob).Error; err != nil {
			return err
		}
		if err := tx.Create(&chunks).Error; err != nil {
			return err
		}
		return chargeUsage(tx, chunks)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to persist blob catalog entry: %w", err)
	}

	return &blob, nil
}

// storeWholeBlock ships one block verbatim to a single machine of the given type.
func (ss *StorageService) storeWholeBlock(blobID uuid.UUID, machineType models.MachineType, data []byte, blockIndex int, logicalOffset int64) (*models.BlobChunk, error) {
	target, err := ss.placement.SelectMachine(machineType, requiredMB(int64(len(data))))
	if err != nil {
		return nil, err
	}
	return ss.pushChunk(blobID, target, data, blockIndex, ShardIndexWhole, logicalOffset, int64(len(data)))
}

// storeErasureBlock encodes one 32MB block into 8+4 shards and spreads them over
// 12 distinct block machines, so no single machine loss costs more than one shard.
func (ss *StorageService) storeErasureBlock(blobID uuid.UUID, block []byte, blockIndex int, logicalOffset int64) ([]models.BlobChunk, error) {
	shards, err := ss.erasure.EncodeAndBracket(block)
	if err != nil {
		return nil, fmt.Errorf("erasure coding failed on block %d: %w", blockIndex, err)
	}

	targets, err := ss.placement.SelectMachinesForEC(TotalShards, requiredMB(int64(len(shards[0]))))
	if err != nil {
		return nil, err
	}

	out := make([]models.BlobChunk, 0, TotalShards)
	for shardIndex, shard := range shards {
		chunk, err := ss.pushChunk(blobID, targets[shardIndex], shard, blockIndex, shardIndex, logicalOffset, int64(len(block)))
		if err != nil {
			return nil, err
		}
		out = append(out, *chunk)
	}
	return out, nil
}

// pushChunk streams one chunk to its machine and builds the catalog row for it.
func (ss *StorageService) pushChunk(blobID uuid.UUID, target Target, data []byte, blockIndex, shardIndex int, logicalOffset, logicalSize int64) (*models.BlobChunk, error) {
	chunkID := uuid.New()

	var (
		physicalPath   string
		physicalOffset int64
		err            error
	)
	if models.MachineType(target.MachineType) == models.MachineTypeInline {
		physicalPath, physicalOffset, err = ss.workers.AppendInline(target.WorkerAddr, target.Namespace, chunkID.String(), data)
	} else {
		physicalPath, physicalOffset, err = ss.workers.WriteBlock(target.WorkerAddr, target.Namespace, chunkID.String(), data)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to store chunk on machine %s: %w", target.Namespace, err)
	}

	sum := sha256.Sum256(data)
	discID, err := uuid.Parse(target.DiscID)
	if err != nil {
		return nil, fmt.Errorf("invalid disc id from placement: %w", err)
	}
	workerID, err := uuid.Parse(target.WorkerID)
	if err != nil {
		return nil, fmt.Errorf("invalid worker id from placement: %w", err)
	}

	return &models.BlobChunk{
		ID:             chunkID,
		BlobID:         blobID,
		DiscID:         discID,
		WorkerID:       workerID,
		Checksum:       hex.EncodeToString(sum[:]),
		Size:           int64(len(data)),
		LogicalOffset:  logicalOffset,
		LogicalSize:    logicalSize,
		BlockIndex:     blockIndex,
		ShardIndex:     shardIndex,
		PhysicalPath:   physicalPath,
		PhysicalOffset: physicalOffset,
	}, nil
}

// chargeUsage books the written bytes against the discs and workers that took them.
func chargeUsage(tx *gorm.DB, chunks []models.BlobChunk) error {
	perDisc := map[uuid.UUID]int64{}
	perWorker := map[uuid.UUID]int64{}
	for _, chunk := range chunks {
		perDisc[chunk.DiscID] += chunk.Size
		perWorker[chunk.WorkerID] += chunk.Size
	}

	for discID, size := range perDisc {
		err := tx.Model(&models.Disc{}).Where("id = ?", discID).
			Updates(map[string]interface{}{
				"used_bytes": gorm.Expr("used_bytes + ?", size),
				"used_mb":    gorm.Expr("(used_bytes + ?) / ?", size, bytesPerMB),
			}).Error
		if err != nil {
			return err
		}
	}
	for workerID, size := range perWorker {
		err := tx.Model(&models.Worker{}).Where("id = ?", workerID).
			Updates(map[string]interface{}{
				"used_bytes": gorm.Expr("used_bytes + ?", size),
				"used_mb":    gorm.Expr("(used_bytes + ?) / ?", size, bytesPerMB),
			}).Error
		if err != nil {
			return err
		}
	}
	return nil
}

// requiredMB rounds a byte count up to whole megabytes for placement filtering.
func requiredMB(size int64) int64 {
	return (size + bytesPerMB - 1) / bytesPerMB
}

// -----------------------------------------------------------------------------
// Read path
// -----------------------------------------------------------------------------

// RetrieveBlob rebuilds the original file from its chunks and verifies it against
// the checksum recorded at upload time.
func (ss *StorageService) RetrieveBlob(blob *models.Blob) ([]byte, error) {
	var locations []chunkLocation
	err := ss.db.Model(&models.BlobChunk{}).
		Select(`blob_chunks.*, machines.namespace AS namespace, workers.address AS worker_addr`).
		Joins("JOIN infra.discs discs ON discs.id = blob_chunks.disc_id").
		Joins("JOIN infra.machines machines ON machines.id = discs.machine_id").
		Joins("JOIN infra.workers workers ON workers.id = blob_chunks.worker_id").
		Where("blob_chunks.blob_id = ?", blob.ID).
		Order("blob_chunks.block_index ASC, blob_chunks.shard_index ASC").
		Scan(&locations).Error
	if err != nil {
		return nil, fmt.Errorf("failed to load chunk catalog: %w", err)
	}
	if len(locations) == 0 {
		return nil, errors.New("blob has no chunks registered")
	}

	blocks := map[int][]chunkLocation{}
	var blockIndexes []int
	for _, loc := range locations {
		if _, seen := blocks[loc.BlockIndex]; !seen {
			blockIndexes = append(blockIndexes, loc.BlockIndex)
		}
		blocks[loc.BlockIndex] = append(blocks[loc.BlockIndex], loc)
	}
	sort.Ints(blockIndexes)

	out := bytes.NewBuffer(make([]byte, 0, blob.Size))
	for _, blockIndex := range blockIndexes {
		data, err := ss.readBlock(blocks[blockIndex])
		if err != nil {
			return nil, fmt.Errorf("failed to rebuild block %d: %w", blockIndex, err)
		}
		out.Write(data)
	}

	if int64(out.Len()) != blob.Size {
		return nil, fmt.Errorf("reassembled size mismatch: got %d bytes, catalog says %d", out.Len(), blob.Size)
	}

	sum := sha256.Sum256(out.Bytes())
	if got := hex.EncodeToString(sum[:]); got != blob.Checksum {
		return nil, fmt.Errorf("integrity check failed: checksum %s does not match stored %s", got, blob.Checksum)
	}

	return out.Bytes(), nil
}

func (ss *StorageService) readBlock(group []chunkLocation) ([]byte, error) {
	if len(group) == 1 && group[0].ShardIndex == ShardIndexWhole {
		return ss.readChunkVerified(group[0])
	}

	// Erasure-coded stripe: any DataShards of the TotalShards are enough, so a
	// missing or corrupted shard is recoverable rather than fatal.
	shards := make([][]byte, TotalShards)
	logicalSize := group[0].LogicalSize
	available := 0
	var readErrs []string

	for _, loc := range group {
		if loc.ShardIndex < 0 || loc.ShardIndex >= TotalShards {
			return nil, fmt.Errorf("chunk %s has invalid shard index %d", loc.ID, loc.ShardIndex)
		}
		data, err := ss.readChunkVerified(loc)
		if err != nil {
			readErrs = append(readErrs, fmt.Sprintf("shard %d: %v", loc.ShardIndex, err))
			continue
		}
		shards[loc.ShardIndex] = data
		available++
	}

	if available < DataShards {
		return nil, fmt.Errorf("only %d of %d shards readable, need %d: %v", available, TotalShards, DataShards, readErrs)
	}

	if available < TotalShards {
		if err := ss.erasure.Enc.Reconstruct(shards); err != nil {
			return nil, fmt.Errorf("reed-solomon reconstruction failed: %w", err)
		}
	}

	buf := new(bytes.Buffer)
	if err := ss.erasure.Enc.Join(buf, shards, int(logicalSize)); err != nil {
		return nil, fmt.Errorf("failed to join shards: %w", err)
	}
	return buf.Bytes(), nil
}

// readChunkVerified fetches a chunk and rejects it if the bytes do not match the
// checksum taken at write time. For a stripe this downgrades corruption into a
// simple missing shard, which Reed-Solomon can repair.
func (ss *StorageService) readChunkVerified(loc chunkLocation) ([]byte, error) {
	data, err := ss.workers.ReadChunk(loc.WorkerAddr, loc.Namespace, loc.PhysicalPath, loc.PhysicalOffset, loc.Size)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(data)
	if got := hex.EncodeToString(sum[:]); got != loc.Checksum {
		return nil, fmt.Errorf("checksum mismatch on chunk %s (got %s, want %s)", loc.ID, got, loc.Checksum)
	}
	return data, nil
}
