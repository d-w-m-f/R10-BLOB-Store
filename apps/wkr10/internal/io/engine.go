package io

import "io"

// StorageEngine defines how a Worker interacts with physical Machine directories.
type StorageEngine interface {
	// Write saves data and returns the physical relative path and the physical byte offset.
	Write(machineNamespace string, chunkID string, data []byte) (physicalPath string, physicalOffset int64, err error)
	
	// Read retrieves data from a specific file path and offset.
	Read(machineNamespace string, physicalPath string, physicalOffset int64, size int64) ([]byte, error)
	
	// StreamWrite writes data directly from an io.Reader to avoid large memory buffering.
	// It reports how many bytes were persisted so callers can detect truncated transfers.
	StreamWrite(machineNamespace string, chunkID string, reader io.Reader, size int64) (physicalPath string, physicalOffset int64, written int64, err error)
}

// Compile-time proof that both engines stay in sync with the interface.
var (
	_ StorageEngine = (*BlockEngine)(nil)
	_ StorageEngine = (*InlineEngine)(nil)
)
