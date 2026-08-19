package clients

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// WorkerClient talks to the wkr10 daemons over the binary streaming protocol:
// payload in the raw body, metadata in X-Chunk-* headers.
type WorkerClient struct {
	http *http.Client
}

func NewWorkerClient() *WorkerClient {
	return &WorkerClient{
		http: &http.Client{Timeout: 60 * time.Second},
	}
}

type writeResponse struct {
	Status         string `json:"status"`
	PhysicalPath   string `json:"physical_path"`
	PhysicalOffset int64  `json:"physical_offset"`
	Error          string `json:"error"`
}

// WriteBlock stores a standalone chunk file on a block machine.
func (wc *WorkerClient) WriteBlock(addr, namespace, chunkID string, data []byte) (string, int64, error) {
	return wc.write(fmt.Sprintf("%s/api/v1/machines/%s/chunks", addr, url.PathEscape(namespace)), chunkID, data)
}

// AppendInline appends a chunk to a machine's append-only volume.
func (wc *WorkerClient) AppendInline(addr, namespace, chunkID string, data []byte) (string, int64, error) {
	return wc.write(fmt.Sprintf("%s/api/v1/machines/%s/append", addr, url.PathEscape(namespace)), chunkID, data)
}

func (wc *WorkerClient) write(endpoint, chunkID string, data []byte) (string, int64, error) {
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("X-Chunk-ID", chunkID)
	req.Header.Set("X-Chunk-Size", strconv.Itoa(len(data)))
	req.ContentLength = int64(len(data))

	res, err := wc.http.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("worker unreachable at %s: %w", endpoint, err)
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	var parsed writeResponse
	_ = json.Unmarshal(body, &parsed)

	if res.StatusCode != http.StatusOK {
		if parsed.Error != "" {
			return "", 0, fmt.Errorf("worker rejected chunk %s: %s", chunkID, parsed.Error)
		}
		return "", 0, fmt.Errorf("worker returned %d for chunk %s: %s", res.StatusCode, chunkID, string(body))
	}
	if parsed.PhysicalPath == "" {
		return "", 0, fmt.Errorf("worker returned no physical path for chunk %s", chunkID)
	}

	return parsed.PhysicalPath, parsed.PhysicalOffset, nil
}

// ReadChunk pulls back an exact byte range from a machine.
func (wc *WorkerClient) ReadChunk(addr, namespace, physicalPath string, offset, size int64) ([]byte, error) {
	endpoint := fmt.Sprintf("%s/api/v1/machines/%s/chunks?path=%s&offset=%d&size=%d",
		addr, url.PathEscape(namespace), url.QueryEscape(physicalPath), offset, size)

	res, err := wc.http.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("worker unreachable at %s: %w", addr, err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return nil, fmt.Errorf("worker returned %d reading %s: %s", res.StatusCode, physicalPath, string(body))
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read chunk body: %w", err)
	}
	if int64(len(data)) != size {
		return nil, fmt.Errorf("worker returned %d bytes, expected %d", len(data), size)
	}

	return data, nil
}
