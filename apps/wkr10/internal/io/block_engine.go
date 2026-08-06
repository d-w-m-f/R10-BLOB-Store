package io

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type BlockEngine struct {
	ClusterRootDir string
}

func NewBlockEngine(rootDir string) *BlockEngine {
	return &BlockEngine{ClusterRootDir: rootDir}
}

// getMachineDir returns the absolute path to the machine's directory
func (e *BlockEngine) getMachineDir(namespace string) string {
	return filepath.Join(e.ClusterRootDir, fmt.Sprintf("machine_%s", namespace))
}

func (e *BlockEngine) Write(machineNamespace string, chunkID string, data []byte) (string, int64, error) {
	machineDir := e.getMachineDir(machineNamespace)
	
	// For block engine, path is just chunks/chunkID.dat
	relPath := fmt.Sprintf("chunks/%s.dat", chunkID)
	absPath := filepath.Join(machineDir, relPath)

	// Ensure chunks directory exists
	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return "", 0, fmt.Errorf("failed to create chunks dir: %w", err)
	}

	if err := os.WriteFile(absPath, data, 0644); err != nil {
		return "", 0, fmt.Errorf("failed to write chunk file: %w", err)
	}

	// Offset is always 0 for standalone block files
	return relPath, 0, nil
}

func (e *BlockEngine) Read(machineNamespace string, physicalPath string, physicalOffset int64, size int64) ([]byte, error) {
	machineDir := e.getMachineDir(machineNamespace)
	absPath := filepath.Join(machineDir, physicalPath)

	file, err := os.Open(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open chunk file: %w", err)
	}
	defer file.Close()

	data := make([]byte, size)
	n, err := file.ReadAt(data, physicalOffset) // physicalOffset should be 0
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read chunk data: %w", err)
	}

	return data[:n], nil
}

func (e *BlockEngine) StreamWrite(machineNamespace string, chunkID string, reader io.Reader, size int64) (string, int64, error) {
	machineDir := e.getMachineDir(machineNamespace)
	relPath := fmt.Sprintf("chunks/%s.dat", chunkID)
	absPath := filepath.Join(machineDir, relPath)

	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return "", 0, fmt.Errorf("failed to create chunks dir: %w", err)
	}

	file, err := os.OpenFile(absPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return "", 0, fmt.Errorf("failed to open chunk file for streaming: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, reader); err != nil {
		return "", 0, fmt.Errorf("failed to stream write chunk: %w", err)
	}

	return relPath, 0, nil
}
