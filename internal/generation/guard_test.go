package generation

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-sd/internal/llm"
	"go-sd/internal/logger"
	"go-sd/internal/sd"
)

var (
	_ llm.Service = (*guardLLM)(nil)
	_ sd.Service  = (*guardSD)(nil)
)

type guardLLM struct {
	backend      string
	listLoadedFn func(call int) ([]llm.LoadedModel, error)
	unloadAllFn  func(ctx context.Context) error

	listLoadedCalls atomic.Int32
	unloadAllCalls  atomic.Int32
}

func (m *guardLLM) Backend() string { return m.backend }

func (m *guardLLM) ListLoaded() ([]llm.LoadedModel, error) {
	call := int(m.listLoadedCalls.Add(1))
	if m.listLoadedFn == nil {
		return nil, nil
	}
	return m.listLoadedFn(call)
}

func (m *guardLLM) UnloadAll(ctx context.Context) error {
	m.unloadAllCalls.Add(1)
	if m.unloadAllFn == nil {
		return nil
	}
	return m.unloadAllFn(ctx)
}

func (m *guardLLM) Chat(model, systemPrompt, userMessage string, temperature float64, maxTokens int) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (m *guardLLM) ChatVision(model, systemPrompt, userText, imageBase64 string, temperature float64, maxTokens int) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (m *guardLLM) ChatWithMessages(model string, messages []llm.Message, temperature float64, maxTokens int) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (m *guardLLM) GenerateSDPrompt(systemPrompt, description, presetType, model string, maxTokens int) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (m *guardLLM) AnalyzeImage(model, systemPrompt, imageBase64 string, maxTokens int) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (m *guardLLM) AnalyzeImageDescribe(model, prompt, imageBase64 string, maxTokens int) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (m *guardLLM) GetModels() ([]llm.LLMModel, error)       { return nil, nil }
func (m *guardLLM) HealthCheck() error                       { return nil }
func (m *guardLLM) SetURL(baseURL string)                    {}
func (m *guardLLM) SetBackend(backend string)                {}
func (m *guardLLM) SetBackendConfig(cfg llm.BackendConfig)   {}

type guardSD struct {
	stats           *sd.MemoryStats
	err             error
	memoryInfoCalls atomic.Int32
}

func (m *guardSD) MemoryInfo() (*sd.MemoryStats, error) {
	m.memoryInfoCalls.Add(1)
	if m.err != nil {
		return nil, m.err
	}
	return m.stats, nil
}

func (m *guardSD) Txt2Img(req sd.Txt2ImgRequest) (*sd.Txt2ImgResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *guardSD) Img2Img(req sd.Img2ImgRequest) (*sd.Txt2ImgResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *guardSD) GetModels() ([]sd.SDModel, error)            { return nil, nil }
func (m *guardSD) GetSamplers() ([]sd.Sampler, error)          { return nil, nil }
func (m *guardSD) GetSchedulers() ([]sd.Scheduler, error)      { return nil, nil }
func (m *guardSD) GetUpscalers() ([]sd.Upscaler, error)        { return nil, nil }
func (m *guardSD) GetVAEs() ([]sd.VAE, error)                  { return nil, nil }
func (m *guardSD) GetLoRAs() ([]sd.LoRA, error)                { return nil, nil }
func (m *guardSD) GetOptions() (map[string]interface{}, error) { return nil, nil }
func (m *guardSD) GetProgress() (*sd.ProgressResponse, error)  { return nil, nil }
func (m *guardSD) Interrupt() error                            { return nil }
func (m *guardSD) HealthCheck() error                          { return nil }
func (m *guardSD) SetURL(baseURL string)                       {}
func (m *guardSD) SetModel(modelName string) error             { return nil }
func (m *guardSD) SetVAE(vaeName string) error                 { return nil }

func (m *guardSD) UpscaleImage(base64Img string, upscaler string, scale float64) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func gib(g float64) float64 { return g * (1 << 30) }

func gibI(g int64) int64 { return g << 30 }

func memPool(free, total float64) sd.MemoryPool {
	return sd.MemoryPool{Free: free, Used: total - free, Total: total}
}

func cudaStats(cudaFree, ramFree float64) *sd.MemoryStats {
	return &sd.MemoryStats{
		RAM:  memPool(ramFree, gib(32)),
		CUDA: &sd.CUDAMemory{System: memPool(cudaFree, gib(24))},
	}
}

func cpuStats(ramFree float64) *sd.MemoryStats {
	return &sd.MemoryStats{RAM: memPool(ramFree, gib(32))}
}

func staticLoaded(models []llm.LoadedModel, err error) func(int) ([]llm.LoadedModel, error) {
	return func(int) ([]llm.LoadedModel, error) { return models, err }
}

func loadedOnce(models []llm.LoadedModel) func(int) ([]llm.LoadedModel, error) {
	return func(call int) ([]llm.LoadedModel, error) {
		if call == 1 {
			return models, nil
		}
		return nil, nil
	}
}

func TestMemoryGuard_EnsureHeadroom(t *testing.T) {
	t.Parallel()

	vramHeavy := llm.LoadedModel{Name: "llama3:latest", Size: gibI(12), SizeVRAM: gibI(10)}

	tests := []struct {
		name            string
		model           string
		autoUnload      string
		heavyModels     string
		backend         string
		listLoadedFn    func(call int) ([]llm.LoadedModel, error)
		stats           *sd.MemoryStats
		memoryErr       error
		shortTimeout    bool
		want            bool
		wantUnloadCalls int32
		wantZeroAPI     bool
		wantNoMemInfo   bool
	}{
		{
			name:            "empty model name skips guard",
			model:           "",
			backend:         llm.BackendOllama,
			listLoadedFn:    staticLoaded([]llm.LoadedModel{vramHeavy}, nil),
			stats:           cudaStats(gib(3), gib(8)),
			want:            false,
			wantUnloadCalls: 0,
			wantZeroAPI:     true,
		},
		{
			name:            "auto unload disabled by setting",
			model:           "flux1-dev-fp8",
			autoUnload:      "false",
			backend:         llm.BackendOllama,
			listLoadedFn:    staticLoaded([]llm.LoadedModel{vramHeavy}, nil),
			stats:           cudaStats(gib(3), gib(8)),
			want:            false,
			wantUnloadCalls: 0,
			wantZeroAPI:     true,
		},
		{
			name:            "non heavy model skipped",
			model:           "epicrealismXL_pureFix",
			backend:         llm.BackendOllama,
			listLoadedFn:    staticLoaded([]llm.LoadedModel{vramHeavy}, nil),
			stats:           cudaStats(gib(3), gib(8)),
			want:            false,
			wantUnloadCalls: 0,
			wantZeroAPI:     true,
		},
		{
			name:            "heavy model on non ollama backend skipped",
			model:           "flux1-dev-fp8",
			backend:         llm.BackendLMStudio,
			listLoadedFn:    staticLoaded([]llm.LoadedModel{vramHeavy}, nil),
			stats:           cudaStats(gib(3), gib(8)),
			want:            false,
			wantUnloadCalls: 0,
			wantZeroAPI:     true,
		},
		{
			name:            "list loaded error fails open",
			model:           "flux1-dev-fp8",
			backend:         llm.BackendOllama,
			listLoadedFn:    staticLoaded(nil, fmt.Errorf("ollama unreachable")),
			stats:           cudaStats(gib(3), gib(8)),
			want:            false,
			wantUnloadCalls: 0,
			wantNoMemInfo:   true,
		},
		{
			name:            "no loaded models nothing to unload",
			model:           "flux1-dev-fp8",
			backend:         llm.BackendOllama,
			listLoadedFn:    staticLoaded(nil, nil),
			stats:           cudaStats(gib(3), gib(8)),
			want:            false,
			wantUnloadCalls: 0,
			wantNoMemInfo:   true,
		},
		{
			name:            "memory info error fails open",
			model:           "flux1-dev-fp8",
			backend:         llm.BackendOllama,
			listLoadedFn:    staticLoaded([]llm.LoadedModel{vramHeavy}, nil),
			memoryErr:       fmt.Errorf("sd memory endpoint unavailable"),
			want:            false,
			wantUnloadCalls: 0,
		},
		{
			name:            "enough headroom keeps llm warm",
			model:           "flux1-dev-fp8",
			backend:         llm.BackendOllama,
			listLoadedFn:    staticLoaded([]llm.LoadedModel{vramHeavy}, nil),
			stats:           cudaStats(gib(14), gib(16)),
			want:            false,
			wantUnloadCalls: 0,
		},
		{
			name:            "headroom at exact thresholds no deficit",
			model:           "flux1-dev-fp8",
			backend:         llm.BackendOllama,
			listLoadedFn:    staticLoaded([]llm.LoadedModel{vramHeavy}, nil),
			stats:           cudaStats(gib(12), gib(10)),
			want:            false,
			wantUnloadCalls: 0,
		},
		{
			name:            "deficit with relief unloads llm",
			model:           "flux1-dev-fp8",
			backend:         llm.BackendOllama,
			listLoadedFn:    loadedOnce([]llm.LoadedModel{vramHeavy}),
			stats:           cudaStats(gib(3), gib(8)),
			want:            true,
			wantUnloadCalls: 1,
		},
		{
			name:         "deficit without cuda relief skips unload",
			model:        "flux1-dev-fp8",
			backend:      llm.BackendOllama,
			listLoadedFn: staticLoaded([]llm.LoadedModel{{Name: "tiny", Size: gibI(2), SizeVRAM: gibI(2)}}, nil),
			stats:        cudaStats(gib(3), gib(16)),
			want:         false,
		},
		{
			name:         "deficit without ram relief skips unload",
			model:        "flux1-dev-fp8",
			backend:      llm.BackendOllama,
			listLoadedFn: staticLoaded([]llm.LoadedModel{{Name: "vram-only", Size: gibI(10), SizeVRAM: gibI(10)}}, nil),
			stats:        cudaStats(gib(3), gib(8)),
			want:         false,
		},
		{
			name:            "cpu only zero cuda free with relief unloads llm",
			model:           "qwen-image-turbo",
			backend:         llm.BackendOllama,
			listLoadedFn:    loadedOnce([]llm.LoadedModel{{Name: "cpu-llm", Size: gibI(14), SizeVRAM: gibI(12)}}),
			stats:           cpuStats(gib(16)),
			want:            true,
			wantUnloadCalls: 1,
		},
		{
			name:            "custom heavy models setting matches own name",
			model:           "megatron-xl",
			heavyModels:     "megatron",
			backend:         llm.BackendOllama,
			listLoadedFn:    loadedOnce([]llm.LoadedModel{vramHeavy}),
			stats:           cudaStats(gib(3), gib(8)),
			want:            true,
			wantUnloadCalls: 1,
		},
		{
			name:            "custom heavy models setting overrides default",
			model:           "flux1-dev-fp8",
			heavyModels:     "megatron",
			backend:         llm.BackendOllama,
			listLoadedFn:    staticLoaded([]llm.LoadedModel{vramHeavy}, nil),
			stats:           cudaStats(gib(3), gib(8)),
			want:            false,
			wantUnloadCalls: 0,
			wantZeroAPI:     true,
		},
		{
			name:            "unload wait timeout still returns true",
			model:           "flux1-dev-fp8",
			backend:         llm.BackendOllama,
			listLoadedFn:    staticLoaded([]llm.LoadedModel{vramHeavy}, nil),
			stats:           cudaStats(gib(3), gib(8)),
			shortTimeout:    true,
			want:            true,
			wantUnloadCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := openTestDB(t)
			if tt.autoUnload != "" {
				require.NoError(t, db.SetSetting("llm_auto_unload", tt.autoUnload))
			}
			if tt.heavyModels != "" {
				require.NoError(t, db.SetSetting("heavy_models", tt.heavyModels))
			}
			llmSvc := &guardLLM{backend: tt.backend, listLoadedFn: tt.listLoadedFn}
			sdSvc := &guardSD{stats: tt.stats, err: tt.memoryErr}
			g := NewMemoryGuard(llmSvc, sdSvc, db, logger.New(nil))
			if tt.shortTimeout {
				g.unloadTimeout = 100 * time.Millisecond
			}

			assert.Equal(t, tt.want, g.EnsureHeadroom(tt.model, "[test]"))

			assert.Equal(t, tt.wantUnloadCalls, llmSvc.unloadAllCalls.Load())
			if tt.wantZeroAPI {
				assert.Zero(t, llmSvc.listLoadedCalls.Load())
				assert.Zero(t, sdSvc.memoryInfoCalls.Load())
				assert.Zero(t, llmSvc.unloadAllCalls.Load())
			}
			if tt.wantNoMemInfo {
				assert.Zero(t, sdSvc.memoryInfoCalls.Load())
			}
		})
	}
}

func TestMemoryGuard_IsHeavy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		model       string
		heavyModels string
		want        bool
	}{
		{name: "default flux substring", model: "flux1-dev-fp8", want: true},
		{name: "default qwen-image substring", model: "QWEN-Image-Turbo", want: true},
		{name: "default no match", model: "epicrealismXL_pureFix", want: false},
		{name: "empty model name", model: "", want: false},
		{name: "custom setting overrides default list", model: "flux1-dev-fp8", heavyModels: "megatron", want: false},
		{name: "custom setting match with spaces and case", model: "Megatron-XL", heavyModels: " megatron , sdxl ", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := openTestDB(t)
			if tt.heavyModels != "" {
				require.NoError(t, db.SetSetting("heavy_models", tt.heavyModels))
			}
			g := NewMemoryGuard(&guardLLM{}, &guardSD{}, db, logger.New(nil))
			assert.Equal(t, tt.want, g.isHeavy(tt.model))
		})
	}
}

func TestMemoryGuard_WaitUnloaded(t *testing.T) {
	t.Parallel()

	t.Run("already unloaded returns true", func(t *testing.T) {
		t.Parallel()
		g := NewMemoryGuard(&guardLLM{listLoadedFn: staticLoaded(nil, nil)}, &guardSD{}, openTestDB(t), logger.New(nil))
		assert.True(t, g.waitUnloaded(context.Background()))
	})

	t.Run("list error then empty returns true", func(t *testing.T) {
		t.Parallel()
		fn := func(call int) ([]llm.LoadedModel, error) {
			if call == 1 {
				return nil, fmt.Errorf("transient poll error")
			}
			return nil, nil
		}
		g := NewMemoryGuard(&guardLLM{listLoadedFn: fn}, &guardSD{}, openTestDB(t), logger.New(nil))
		assert.True(t, g.waitUnloaded(context.Background()))
	})

	t.Run("context timeout returns false", func(t *testing.T) {
		t.Parallel()
		g := NewMemoryGuard(&guardLLM{listLoadedFn: staticLoaded([]llm.LoadedModel{{Name: "stuck"}}, nil)}, &guardSD{}, openTestDB(t), logger.New(nil))
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		assert.False(t, g.waitUnloaded(ctx))
	})
}
