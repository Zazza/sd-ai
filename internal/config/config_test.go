package config

import (
	"strings"
	"testing"
)

func TestLoad_DefaultValues(t *testing.T) {
	t.Parallel()

	cfg := Load()

	if cfg.LLMUrl != "http://localhost:1234" {
		t.Errorf("LLMUrl = %q, want %q", cfg.LLMUrl, "http://localhost:1234")
	}
	if cfg.SDUrl != "http://localhost:7860" {
		t.Errorf("SDUrl = %q, want %q", cfg.SDUrl, "http://localhost:7860")
	}
	if cfg.LLMBackend != "lmstudio" {
		t.Errorf("LLMBackend = %q, want %q", cfg.LLMBackend, "lmstudio")
	}
	if cfg.LLMModel != "openai/gpt-oss-20b" {
		t.Errorf("LLMModel = %q, want %q", cfg.LLMModel, "openai/gpt-oss-20b")
	}
	if cfg.SDPromptModel != "default" {
		t.Errorf("SDPromptModel = %q, want %q", cfg.SDPromptModel, "default")
	}
	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want %q", cfg.Port, "8080")
	}
	if cfg.DefaultSampler != "Euler a" {
		t.Errorf("DefaultSampler = %q, want %q", cfg.DefaultSampler, "Euler a")
	}
	if cfg.DefaultSteps != 20 {
		t.Errorf("DefaultSteps = %d, want %d", cfg.DefaultSteps, 20)
	}
	if cfg.DefaultCfgScale != 7.0 {
		t.Errorf("DefaultCfgScale = %f, want %f", cfg.DefaultCfgScale, 7.0)
	}
	if cfg.DefaultWidth != 512 {
		t.Errorf("DefaultWidth = %d, want %d", cfg.DefaultWidth, 512)
	}
	if cfg.DefaultHeight != 512 {
		t.Errorf("DefaultHeight = %d, want %d", cfg.DefaultHeight, 512)
	}
	if cfg.DefaultNegative != "blurry, low quality, watermark, text, signature" {
		t.Errorf("DefaultNegative = %q, want %q", cfg.DefaultNegative, "blurry, low quality, watermark, text, signature")
	}
}

func TestLoad_DefaultPromptContent(t *testing.T) {
	t.Parallel()

	cfg := Load()

	if !strings.Contains(cfg.SystemPrompt, "SD ONLY understands concrete visual tags") {
		t.Error("SystemPrompt should contain SD tag instructions")
	}
	if !strings.Contains(cfg.SystemPrompt, "JSON") {
		t.Error("SystemPrompt should reference JSON output format")
	}
}

func TestLoad_DefaultDBPath(t *testing.T) {
	t.Parallel()

	cfg := Load()

	if !strings.Contains(cfg.DBPath, "presets.db") {
		t.Errorf("DBPath = %q, should contain presets.db", cfg.DBPath)
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	tests := []struct {
		name     string
		envKey   string
		envValue string
		check    func(cfg *Config) string
		want     string
	}{
		{
			name:     "LLM_URL override",
			envKey:   "LLM_URL",
			envValue: "http://custom-llm:9999",
			check:    func(cfg *Config) string { return cfg.LLMUrl },
			want:     "http://custom-llm:9999",
		},
		{
			name:     "SD_URL override",
			envKey:   "SD_URL",
			envValue: "http://custom-sd:8888",
			check:    func(cfg *Config) string { return cfg.SDUrl },
			want:     "http://custom-sd:8888",
		},
		{
			name:     "LLM_MODEL override",
			envKey:   "LLM_MODEL",
			envValue: "custom-model-v2",
			check:    func(cfg *Config) string { return cfg.LLMModel },
			want:     "custom-model-v2",
		},
		{
			name:     "SD_PROMPT_MODEL override",
			envKey:   "SD_PROMPT_MODEL",
			envValue: "prompt-model-v1",
			check:    func(cfg *Config) string { return cfg.SDPromptModel },
			want:     "prompt-model-v1",
		},
		{
			name:     "LLM_BACKEND override",
			envKey:   "LLM_BACKEND",
			envValue: "ollama",
			check:    func(cfg *Config) string { return cfg.LLMBackend },
			want:     "ollama",
		},
		{
			name:     "PORT override",
			envKey:   "PORT",
			envValue: "9090",
			check:    func(cfg *Config) string { return cfg.Port },
			want:     "9090",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.envKey, tt.envValue)
			cfg := Load()
			got := tt.check(cfg)
			if got != tt.want {
				t.Errorf("config field = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLoad_DBPathOverride(t *testing.T) {
	t.Setenv("DB_PATH", "/tmp/test-presets.db")
	cfg := Load()
	if cfg.DBPath != "/tmp/test-presets.db" {
		t.Errorf("DBPath = %q, want %q", cfg.DBPath, "/tmp/test-presets.db")
	}
}

func TestLoad_EmptyEnvFallsBack(t *testing.T) {
	t.Setenv("LLM_URL", "")
	cfg := Load()
	if cfg.LLMUrl != "http://localhost:1234" {
		t.Errorf("LLMUrl with empty env = %q, want default %q", cfg.LLMUrl, "http://localhost:1234")
	}
}
