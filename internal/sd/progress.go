package sd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type ProgressResponse struct {
	Progress    float64 `json:"progress"`
	ETARelative float64 `json:"eta_relative"`
	State       struct {
		Job      string `json:"job"`
		JobCount int    `json:"job_count"`
		JobNo    int    `json:"job_no"`
		Sampling struct {
			Steps       int     `json:"steps"`
			SamplerName string `json:"sampler_name"`
		} `json:"sampling"`
	} `json:"state"`
	CurrentImage string `json:"current_image"`
}

func (c *Client) GetProgress() (*ProgressResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/sdapi/v1/progress", nil)
	if err != nil {
		return nil, fmt.Errorf("get progress: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get progress: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get progress: status %d", resp.StatusCode)
	}
	var result ProgressResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode progress: %w", err)
	}
	return &result, nil
}

func (c *Client) Interrupt() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/sdapi/v1/interrupt", nil)
	if err != nil {
		return fmt.Errorf("interrupt: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("interrupt: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("interrupt: status %d", resp.StatusCode)
	}
	return nil
}
