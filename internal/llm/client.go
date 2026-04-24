package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	backend    string
	backendCfg BackendConfig
	httpClient *http.Client
}

func New(baseURL, backend string) *Client {
	if backend == "" {
		backend = BackendLMStudio
	}
	return &Client{
		baseURL: baseURL,
		backend: backend,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatOptions struct {
	NumCtx int `json:"num_ctx,omitempty"`
	NumGPU int `json:"num_gpu,omitempty"`
}

type ChatRequest struct {
	Model       string       `json:"model"`
	Messages    []Message    `json:"messages"`
	Temperature float64      `json:"temperature"`
	MaxTokens   int          `json:"max_tokens"`
	Stream      bool         `json:"stream"`
	KeepAlive   string       `json:"keep_alive,omitempty"`
	Options     *ChatOptions `json:"options,omitempty"`
}

type ChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func (c *Client) Chat(model, systemPrompt, userMessage string, temperature float64, maxTokens int) (string, error) {
	reqBody := ChatRequest{
		Model: model,
		Messages: []Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userMessage},
		},
		Temperature: temperature,
		MaxTokens:   maxTokens,
	}

	if c.backend == BackendOllama {
		reqBody.KeepAlive = c.backendCfg.KeepAlive
		opts := ChatOptions{
			NumCtx: c.backendCfg.NumCtx,
			NumGPU: c.backendCfg.NumGPU,
		}
		reqBody.Options = &opts
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	url := c.baseURL + "/v1/chat/completions"
	log.Printf("[LLM] POST %s model=%s max_tokens=%d temperature=%.1f prompt_len=%d", url, model, maxTokens, temperature, len(userMessage))

	resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("[LLM] request error: %v", err)
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[LLM] read error: %v", err)
		return "", fmt.Errorf("read response: %w", err)
	}

	log.Printf("[LLM] response status=%d body_len=%d body=%s", resp.StatusCode, len(respBody), string(respBody))

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		log.Printf("[LLM] decode error: %v", err)
		return "", fmt.Errorf("decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("empty response from LLM (body: %s)", string(respBody))
	}

	content := chatResp.Choices[0].Message.Content
	content = strings.TrimSpace(stripThinkTags(content))
	return content, nil
}

var thinkRe = regexp.MustCompile(`(?s)<think\s*>.*?</think\s*>`)

func stripThinkTags(s string) string {
	return thinkRe.ReplaceAllString(s, "")
}

func extractTags(s string) string {
	lower := strings.ToLower(s)
	tagMarkers := []string{"masterpiece", "score_9"}
	tagStart := -1
	for _, m := range tagMarkers {
		if idx := strings.Index(lower, m); idx >= 0 {
			if tagStart < 0 || idx < tagStart {
				tagStart = idx
			}
		}
	}

	if tagStart < 0 {
		return cleanResponse(s)
	}

	if nl := strings.LastIndex(s[:tagStart], "\n"); nl >= 0 {
		tagStart = nl + 1
	}

	result := s[tagStart:]

	cutMarkers := []string{"\n\nLet me", "\n\nHere ", "\n\nNote:", "\n\n---", "\n\n**", "\n\n#",
		"\nLet me create", "\nNow let me", "\nI'll ", "\nFirst,", "\nSo the"}
	for _, m := range cutMarkers {
		if idx := strings.Index(result, m); idx > 0 {
			result = result[:idx]
		}
	}

	result = strings.TrimSpace(result)
	result = strings.ReplaceAll(result, "\n", ", ")
	for strings.Contains(result, ", ,") {
		result = strings.ReplaceAll(result, ", ,", ",")
	}
	for strings.Contains(result, ",,") {
		result = strings.ReplaceAll(result, ",,", ",")
	}
	return result
}

func cleanResponse(s string) string {
	lines := strings.Split(s, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "**") && !strings.HasPrefix(line, "#") &&
			!strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "*") &&
			!strings.HasPrefix(line, ">") && !strings.HasSuffix(line, ":") &&
			!strings.HasPrefix(line, "```") {
			return strings.Join(lines[i:], "\n")
		}
	}
	return s
}

func (c *Client) GenerateSDPrompt(systemPrompt, description, presetType, model string) (string, error) {
	userMessage := description
	if presetType != "" {
		userMessage = fmt.Sprintf("[Type: %s] %s", presetType, description)
	}

	result, err := c.Chat(model, systemPrompt, userMessage, 0.4, 500)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(extractTags(result)), nil
}

func (c *Client) HealthCheck() error {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(c.baseURL + "/v1/models")
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check status %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) SetURL(baseURL string) {
	c.baseURL = baseURL
}

func (c *Client) SetBackend(backend string) {
	c.backend = backend
}

func (c *Client) SetBackendConfig(cfg BackendConfig) {
	c.backendCfg = cfg
}

type LLMModel struct {
	ID     string `json:"id"`
	Object string `json:"object"`
}

func (c *Client) GetModels() ([]LLMModel, error) {
	if c.backend == BackendOllama {
		return c.getOllamaModels()
	}
	return c.getOpenAIModels()
}

func (c *Client) getOpenAIModels() ([]LLMModel, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/v1/models")
	if err != nil {
		return nil, fmt.Errorf("get models: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Data []LLMModel `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode models: %w", err)
	}
	return result.Data, nil
}
