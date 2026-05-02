package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type ollamaTagsResponse struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
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
