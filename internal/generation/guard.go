package generation

import (
	"context"
	"sync"
	"time"

	"go-sd/internal/config"
	"go-sd/internal/llm"
	"go-sd/internal/logger"
	"go-sd/internal/preset"
	"go-sd/internal/sd"
)

const (
	heavyVRAMHeadroom = 12 << 30
	heavyRAMHeadroom  = 10 << 30
	unloadWaitTimeout = 15 * time.Second
	unloadPollPeriod  = 500 * time.Millisecond
)

type MemoryGuard struct {
	llm           llm.Service
	sd            sd.Service
	db            *preset.DB
	log           *logger.Logger
	mu            sync.Mutex
	unloadTimeout time.Duration
}

func NewMemoryGuard(llmSvc llm.Service, sdSvc sd.Service, db *preset.DB, log *logger.Logger) *MemoryGuard {
	return &MemoryGuard{llm: llmSvc, sd: sdSvc, db: db, log: log, unloadTimeout: unloadWaitTimeout}
}

func (g *MemoryGuard) setting(key, fallback string) string {
	v, err := g.db.GetSetting(key)
	if err != nil || v == "" {
		return fallback
	}
	return v
}

func (g *MemoryGuard) isHeavy(modelName string) bool {
	return config.ModelMatchesCSV(modelName, g.setting("heavy_models", config.DefaultHeavyModels))
}

func (g *MemoryGuard) EnsureHeadroom(modelName, logPrefix string) bool {
	g.mu.Lock()
	unload := g.decide(modelName, logPrefix)
	g.mu.Unlock()
	if !unload {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), g.unloadTimeout)
	defer cancel()
	if err := g.llm.UnloadAll(ctx); err != nil {
		g.log.Warn("%s: memory guard: unload: %s", logPrefix, err)
	}
	if !g.waitUnloaded(ctx) {
		g.log.Warn("%s: memory guard: unload wait timeout, proceeding anyway", logPrefix)
	}
	return true
}

func (g *MemoryGuard) decide(modelName, logPrefix string) bool {
	if modelName == "" {
		return false
	}
	if g.setting("llm_auto_unload", "true") == "false" {
		return false
	}
	if !g.isHeavy(modelName) {
		return false
	}
	if g.llm.Backend() != llm.BackendOllama {
		return false
	}
	loaded, err := g.llm.ListLoaded()
	if err != nil {
		g.log.Warn("%s: memory guard: list loaded models: %s", logPrefix, err)
		return false
	}
	if len(loaded) == 0 {
		return false
	}
	stats, err := g.sd.MemoryInfo()
	if err != nil || stats == nil {
		g.log.Warn("%s: memory guard: memory info: %s", logPrefix, err)
		return false
	}

	cudaFree := 0.0
	if stats.CUDA != nil {
		cudaFree = stats.CUDA.System.Free
	}
	ramFree := stats.RAM.Free
	var reclVRAM, reclRAM float64
	for _, m := range loaded {
		reclVRAM += float64(m.SizeVRAM)
		if diff := m.Size - m.SizeVRAM; diff > 0 {
			reclRAM += float64(diff)
		}
	}

	gib := float64(1 << 30)
	deficit := cudaFree < float64(heavyVRAMHeadroom) || ramFree < float64(heavyRAMHeadroom)
	if !deficit {
		g.log.Info("%s: memory guard: enough headroom for %q (cuda_free=%.1fG ram_free=%.1fG), LLM kept warm", logPrefix, modelName, cudaFree/gib, ramFree/gib)
		return false
	}
	relief := cudaFree+reclVRAM >= float64(heavyVRAMHeadroom) && ramFree+reclRAM >= float64(heavyRAMHeadroom)
	if !relief {
		g.log.Info("%s: memory guard: low memory for %q (cuda_free=%.1fG ram_free=%.1fG reclaimable cuda=%.1fG ram=%.1fG), unload would not free enough", logPrefix, modelName, cudaFree/gib, ramFree/gib, reclVRAM/gib, reclRAM/gib)
		return false
	}

	g.log.Info("%s: memory guard: unloading LLM before %q (cuda_free=%.1fG ram_free=%.1fG reclaimable cuda=%.1fG ram=%.1fG)", logPrefix, modelName, cudaFree/gib, ramFree/gib, reclVRAM/gib, reclRAM/gib)
	return true
}

func (g *MemoryGuard) waitUnloaded(ctx context.Context) bool {
	ticker := time.NewTicker(unloadPollPeriod)
	defer ticker.Stop()
	for {
		loaded, err := g.llm.ListLoaded()
		if err == nil && len(loaded) == 0 {
			return true
		}
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
		}
	}
}
