package llm

import (
	"bytes"
	"context"
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

var _ Service = (*Client)(nil)

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
	Content any    `json:"content"`
}

type ContentPart struct {
	Type     string       `json:"type"`
	Text     string       `json:"text,omitempty"`
	ImageURL *ImageURLPart `json:"image_url,omitempty"`
}

type ImageURLPart struct {
	URL string `json:"url"`
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
		Stream:      false,
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

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
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

func (c *Client) ChatVision(model, systemPrompt, userText, imageBase64 string, temperature float64, maxTokens int) (string, error) {
	reqBody := ChatRequest{
		Model: model,
		Messages: []Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: []ContentPart{
				{Type: "text", Text: userText},
				{Type: "image_url", ImageURL: &ImageURLPart{URL: "data:image/png;base64," + imageBase64}},
			}},
		},
		Temperature: temperature,
		MaxTokens:   maxTokens,
		Stream:      false,
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
	log.Printf("[LLM Vision] POST %s model=%s", url, model)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("empty response from vision model")
	}

	return strings.TrimSpace(stripThinkTags(chatResp.Choices[0].Message.Content)), nil
}

func (c *Client) AnalyzeImage(model, systemPrompt, imageBase64 string, maxTokens int) (string, error) {
	userText := "Describe this image as comma-separated SD tags. Output ONLY tags, nothing else."
	result, err := c.ChatVision(model, systemPrompt, userText, imageBase64, 0.4, maxTokens)
	if err != nil {
		return "", err
	}
	result = strings.TrimSpace(extractTags(result))
	result = truncateRepetitive(result, 1000)
	return result, nil
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

func (c *Client) GenerateSDPrompt(systemPrompt, description, presetType, model string, maxTokens int) (string, error) {
	userMessage := description
	if presetType != "" {
		userMessage = fmt.Sprintf("[Type: %s] %s", presetType, description)
	}

	result, err := c.Chat(model, systemPrompt, userMessage, 0.4, maxTokens)
	if err != nil {
		return "", err
	}
	return result, nil
}

func truncateRepetitive(s string, maxLen int) string {
	if s == "" {
		return s
	}

	parts := strings.Split(s, ", ")
	result := make([]string, 0, len(parts))
	prevPrefix := ""
	repeatCount := 0

	for _, part := range parts {
		prefix := part
		if idx := strings.Index(part, ":"); idx > 0 {
			prefix = part[:idx]
		}
		prefix = strings.ToLower(strings.TrimSpace(prefix))

		if prefix == prevPrefix && prefix != "" {
			repeatCount++
			if repeatCount >= 3 {
				break
			}
		} else {
			prevPrefix = prefix
			repeatCount = 0
		}
		result = append(result, part)
	}

	s = strings.Join(result, ", ")

	if len(s) > maxLen {
		if idx := strings.LastIndex(s[:maxLen], ","); idx > 0 {
			s = s[:idx]
		} else {
			s = s[:maxLen]
		}
	}

	return strings.TrimRight(s, " ,.")
}

func (c *Client) ChatWithMessages(model string, messages []Message, temperature float64, maxTokens int) (string, error) {
	reqBody := ChatRequest{
		Model:       model,
		Messages:    messages,
		Temperature: temperature,
		MaxTokens:   maxTokens,
		Stream:      false,
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
	log.Printf("[LLM] POST %s model=%s msgs=%d", url, model, len(messages))

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("empty response from LLM")
	}

	return strings.TrimSpace(stripThinkTags(chatResp.Choices[0].Message.Content)), nil
}

func CleanTags(s string) string {
	s = strings.TrimSpace(extractTags(s))
	s = truncateRepetitive(s, 1000)
	return s
}

func (c *Client) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/models", nil)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	resp, err := c.httpClient.Do(req)
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
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, c.baseURL+"/v1/models", nil)
	if err != nil {
		return nil, fmt.Errorf("get models: %w", err)
	}
	resp, err := c.httpClient.Do(req)
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
