package sd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type MemoryPool struct {
	Free  float64 `json:"free"`
	Used  float64 `json:"used"`
	Total float64 `json:"total"`
}

type CUDAMemory struct {
	System MemoryPool `json:"system"`
}

type MemoryStats struct {
	RAM  MemoryPool  `json:"ram"`
	CUDA *CUDAMemory `json:"cuda,omitempty"`
}

func (c *Client) MemoryInfo() (*MemoryStats, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/sdapi/v1/memory", nil)
	if err != nil {
		return nil, fmt.Errorf("memory info: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("memory info: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, io.LimitReader(resp.Body, maxResponseBodySize))
		return nil, fmt.Errorf("memory info: status %d", resp.StatusCode)
	}
	var stats MemoryStats
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBodySize)).Decode(&stats); err != nil {
		return nil, fmt.Errorf("decode memory info: %w", err)
	}
	return &stats, nil
}
