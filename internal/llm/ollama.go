package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type ollamaTagsResponse struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
}

type LoadedModel struct {
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	SizeVRAM  int64  `json:"size_vram"`
	ExpiresAt string `json:"expires_at"`
}

type ollamaPsResponse struct {
	Models []LoadedModel `json:"models"`
}

type ollamaUnloadRequest struct {
	Model     string `json:"model"`
	KeepAlive int    `json:"keep_alive"`
}

func (c *Client) getOllamaModels() ([]LLMModel, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, c.baseURL+"/api/tags", nil)
	if err != nil {
		return nil, fmt.Errorf("get ollama models: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get ollama models: %w", err)
	}
	defer resp.Body.Close()

	var result ollamaTagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode ollama models: %w", err)
	}

	models := make([]LLMModel, len(result.Models))
	for i, m := range result.Models {
		models[i] = LLMModel{ID: m.Name, Object: "model"}
	}
	return models, nil
}

func (c *Client) listOllamaLoaded() ([]LoadedModel, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/ps", nil)
	if err != nil {
		return nil, fmt.Errorf("list loaded models: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list loaded models: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, io.LimitReader(resp.Body, maxLLMResponseBodySize))
		return nil, fmt.Errorf("list loaded models: status %d", resp.StatusCode)
	}

	var result ollamaPsResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxLLMResponseBodySize)).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode loaded models: %w", err)
	}
	return result.Models, nil
}

func (c *Client) unloadOllamaAll(ctx context.Context) error {
	loaded, err := c.listOllamaLoaded()
	if err != nil {
		return err
	}
	for _, m := range loaded {
		body, err := json.Marshal(ollamaUnloadRequest{Model: m.Name})
		if err != nil {
			log.Printf("[LLM] unload %q: marshal: %v", m.Name, err)
			continue
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/generate", bytes.NewReader(body))
		if err != nil {
			log.Printf("[LLM] unload %q: %v", m.Name, err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := c.httpClient.Do(req)
		if err != nil {
			log.Printf("[LLM] unload %q: %v", m.Name, err)
			continue
		}
		io.Copy(io.Discard, io.LimitReader(resp.Body, maxLLMResponseBodySize))
		resp.Body.Close()
	}
	return nil
}
