package llm

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-sd/internal/promptutil"
)

func TestTruncateRepetitive(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "empty string",
			input:  "",
			maxLen: 1000,
			want:   "",
		},
		{
			name:   "normal prompt unchanged",
			input:  "masterpiece, best quality, detailed skin, beautiful eyes, long hair",
			maxLen: 1000,
			want:   "masterpiece, best quality, detailed skin, beautiful eyes, long hair",
		},
		{
			name:   "truncate repetitive blouse pattern",
			input:  "masterpiece, best quality, (blouse:1), (blouse:open), (blouse:satin), (blouse:glitter), (blouse:blouse), (blouse:blouse:blouse), (blouse:blouse:blouse:blouse)",
			maxLen: 1000,
			want:   "masterpiece, best quality, (blouse:1), (blouse:open), (blouse:satin)",
		},
		{
			name:   "allow up to 3 same-prefix tags",
			input:  "(hair:blonde:1.2), (hair:long:1.1), (hair:wavy:1.0)",
			maxLen: 1000,
			want:   "(hair:blonde:1.2), (hair:long:1.1), (hair:wavy:1.0)",
		},
		{
			name:   "hard truncation at maxLen",
			input:  "masterpiece, best quality, " + string(make([]byte, 2000)),
			maxLen: 50,
			want:   "masterpiece, best quality",
		},
		{
			name:   "different prefixes not truncated",
			input:  "masterpiece, best quality, (eyes:blue:1.2), (hair:blonde:1.1), (skin:detailed:1.0)",
			maxLen: 1000,
			want:   "masterpiece, best quality, (eyes:blue:1.2), (hair:blonde:1.1), (skin:detailed:1.0)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := promptutil.TruncateRepetitive(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateRepetitive() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCleanTags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "already clean tags",
			input: "masterpiece, best quality, 1girl, long hair",
			want:  "masterpiece, best quality, 1girl, long hair",
		},
		{
			name:  "extra whitespace",
			input: "  masterpiece ,   best quality  ,   1girl  ",
			want:  "masterpiece ,   best quality  ,   1girl",
		},
		{
			name:  "commas with spaces preserved",
			input: "tag1, tag2, tag3",
			want:  "tag1, tag2, tag3",
		},
		{
			name:  "trailing comma handled by truncateRepetitive",
			input: "tag1, tag2,",
			want:  "tag1, tag2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := CleanTags(tt.input)
			if got != tt.want {
				t.Errorf("CleanTags() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStripThinkTags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no think tags",
			input: "hello world",
			want:  "hello world",
		},
		{
			name:  "with think tags",
			input: "<think >reasoning here</think > actual output",
			want:  " actual output",
		},
		{
			name:  "think tags with content",
			input: "<think >some reasoning</think >final answer",
			want:  "final answer",
		},
		{
			name:  "multiline think block",
			input: "<think >\nline1\nline2\n</think >result",
			want:  "result",
		},
		{
			name:  "empty think block",
			input: "<think ></think >content",
			want:  "content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := stripThinkTags(tt.input)
			if got != tt.want {
				t.Errorf("stripThinkTags() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestChat_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("expected /v1/chat/completions, got %s", r.URL.Path)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}

		var req ChatRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}

		if req.Model != "test-model" {
			t.Errorf("expected model test-model, got %s", req.Model)
		}
		if req.Temperature != 0.7 {
			t.Errorf("expected temperature 0.7, got %f", req.Temperature)
		}
		if req.MaxTokens != 100 {
			t.Errorf("expected max_tokens 100, got %d", req.MaxTokens)
		}
		if req.Stream != false {
			t.Errorf("expected stream false, got %v", req.Stream)
		}
		if len(req.Messages) != 2 {
			t.Fatalf("expected 2 messages, got %d", len(req.Messages))
		}
		if req.Messages[0].Role != "system" {
			t.Errorf("expected first message role system, got %s", req.Messages[0].Role)
		}
		if req.Messages[1].Role != "user" {
			t.Errorf("expected second message role user, got %s", req.Messages[1].Role)
		}

		resp := ChatResponse{}
		resp.Choices = append(resp.Choices, struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		}{})
		resp.Choices[0].Message.Content = "generated prompt output"

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := New(server.URL, BackendLMStudio)
	result, err := client.Chat("test-model", "system prompt", "user message", 0.7, 100)
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}
	if result != "generated prompt output" {
		t.Errorf("Chat() = %q, want %q", result, "generated prompt output")
	}
}

func TestChat_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
	}))
	defer server.Close()

	client := New(server.URL, BackendLMStudio)
	_, err := client.Chat("model", "sys", "user", 0.5, 50)
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if !strings.Contains(err.Error(), "API error 500") {
		t.Errorf("error = %q, want to contain 'API error 500'", err.Error())
	}
}

func TestChat_EmptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ChatResponse{}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := New(server.URL, BackendLMStudio)
	_, err := client.Chat("model", "sys", "user", 0.5, 50)
	if err == nil {
		t.Fatal("expected error for empty choices")
	}
	if !strings.Contains(err.Error(), "empty response") {
		t.Errorf("error = %q, want to contain 'empty response'", err.Error())
	}
}

func TestGetModels_OpenAI(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Errorf("expected /v1/models, got %s", r.URL.Path)
		}

		result := struct {
			Data []LLMModel `json:"data"`
		}{
			Data: []LLMModel{
				{ID: "model-a", Object: "model"},
				{ID: "model-b", Object: "model"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}))
	defer server.Close()

	client := New(server.URL, BackendLMStudio)
	models, err := client.GetModels()
	if err != nil {
		t.Fatalf("GetModels() error: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}
	if models[0].ID != "model-a" {
		t.Errorf("models[0].ID = %q, want %q", models[0].ID, "model-a")
	}
	if models[1].ID != "model-b" {
		t.Errorf("models[1].ID = %q, want %q", models[1].ID, "model-b")
	}
}

func TestGetModels_Ollama(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Errorf("expected /api/tags, got %s", r.URL.Path)
		}

		result := struct {
			Models []struct {
				Name string `json:"name"`
			} `json:"models"`
		}{
			Models: []struct {
				Name string `json:"name"`
			}{
				{Name: "llama3:latest"},
				{Name: "mistral:latest"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}))
	defer server.Close()

	client := New(server.URL, BackendOllama)
	models, err := client.GetModels()
	if err != nil {
		t.Fatalf("GetModels() error: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}
	if models[0].ID != "llama3:latest" {
		t.Errorf("models[0].ID = %q, want %q", models[0].ID, "llama3:latest")
	}
	if models[1].ID != "mistral:latest" {
		t.Errorf("models[1].ID = %q, want %q", models[1].ID, "mistral:latest")
	}
}

func TestGetModels_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := New(server.URL, BackendLMStudio)
	_, err := client.GetModels()
	if err == nil {
		t.Fatal("expected error for unavailable server")
	}
}

func TestHealthCheck_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Errorf("expected /v1/models, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := New(server.URL, BackendLMStudio)
	if err := client.HealthCheck(); err != nil {
		t.Fatalf("HealthCheck() error: %v", err)
	}
}

func TestHealthCheck_Failure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := New(server.URL, BackendLMStudio)
	err := client.HealthCheck()
	if err == nil {
		t.Fatal("expected error for unhealthy server")
	}
	if !strings.Contains(err.Error(), "health check status 503") {
		t.Errorf("error = %q, want to contain 'health check status 503'", err.Error())
	}
}

func TestHealthCheck_ConnectionRefused(t *testing.T) {
	client := New("http://127.0.0.1:0", BackendLMStudio)
	err := client.HealthCheck()
	if err == nil {
		t.Fatal("expected error for connection refused")
	}
	if !strings.Contains(err.Error(), "health check failed") {
		t.Errorf("error = %q, want to contain 'health check failed'", err.Error())
	}
}
