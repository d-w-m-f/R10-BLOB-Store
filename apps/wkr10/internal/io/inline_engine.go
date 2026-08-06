package io

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

type InlineEngine struct {
	ClusterRootDir string
	machineMutexes sync.Map
}

func NewInlineEngine(rootDir string) *InlineEngine {
	return &InlineEngine{ClusterRootDir: rootDir}
}

func (e *InlineEngine) getMutex(namespace string) *sync.Mutex {
	mu, _ := e.machineMutexes.LoadOrStore(namespace, &sync.Mutex{})
	return mu.(*sync.Mutex)
}

func (e *InlineEngine) getMachineDir(namespace string) string {
	return filepath.Join(e.ClusterRootDir, fmt.Sprintf("machine_%s", namespace))
}

func (e *InlineEngine) Write(machineNamespace string, chunkID string, data []byte) (string, int64, error) {
	mu := e.getMutex(machineNamespace)
	mu.Lock()
	defer mu.Unlock()

	machineDir := e.getMachineDir(machineNamespace)
	relPath := "volume_01.dat" // Simplified: a single volume per machine for V1
	absPath := filepath.Join(machineDir, relPath)

	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return "", 0, fmt.Errorf("failed to create volume dir: %w", err)
	}

	file, err := os.OpenFile(absPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return "", 0, fmt.Errorf("failed to open volume file: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return "", 0, fmt.Errorf("failed to stat volume file: %w", err)
	}
	
	offset := stat.Size()

	if _, err := file.Write(data); err != nil {
		return "", 0, fmt.Errorf("failed to write to volume file: %w", err)
	}

	return relPath, offset, nil
}

func (e *InlineEngine) Read(machineNamespace string, physicalPath string, physicalOffset int64, size int64) ([]byte, error) {
	machineDir := e.getMachineDir(machineNamespace)
	absPath := filepath.Join(machineDir, physicalPath)

	file, err := os.Open(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open volume file for read: %w", err)
	}
	defer file.Close()

	data := make([]byte, size)
	n, err := file.ReadAt(data, physicalOffset)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read from volume: %w", err)
	}

	return data[:n], nil
}

func (e *InlineEngine) StreamWrite(machineNamespace string, chunkID string, reader io.Reader, size int64) (string, int64, error) {
	mu := e.getMutex(machineNamespace)
	mu.Lock()
	defer mu.Unlock()

	machineDir := e.getMachineDir(machineNamespace)
	relPath := "volume_01.dat"
	absPath := filepath.Join(machineDir, relPath)

	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return "", 0, fmt.Errorf("failed to create volume dir: %w", err)
	}

	file, err := os.OpenFile(absPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return "", 0, fmt.Errorf("failed to open volume file for streaming: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return "", 0, fmt.Errorf("failed to stat volume file: %w", err)
	}
	
	offset := stat.Size()

	if _, err := io.Copy(file, reader); err != nil {
		return "", 0, fmt.Errorf("failed to stream to volume file: %w", err)
	}

	return relPath, offset, nil
}
