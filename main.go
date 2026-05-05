package main

import (
	"embed"
	"fmt"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"

	"go-sd/internal/config"
	"go-sd/internal/llm"
	"go-sd/internal/preset"
	"go-sd/internal/sd"
)

var version = "dev"

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	cfg := config.Load()

	presets, err := preset.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer presets.Close()

	llmClient := llm.New(cfg.LLMUrl, cfg.LLMBackend)
	sdClient := sd.New(cfg.SDUrl)

	if v, _ := presets.GetSetting("llm_url"); v != "" {
		cfg.LLMUrl = v
		llmClient.SetURL(v)
	}
	if v, _ := presets.GetSetting("sd_url"); v != "" {
		cfg.SDUrl = v
		sdClient.SetURL(v)
	}
	if v, _ := presets.GetSetting("llm_model"); v != "" {
		cfg.LLMModel = v
	}
	if v, _ := presets.GetSetting("sd_prompt_model"); v != "" {
		cfg.SDPromptModel = v
	}
	if v, _ := presets.GetSetting("vision_model"); v != "" {
		cfg.VisionModel = v
	}
	if v, _ := presets.GetSetting("llm_backend"); v != "" {
		cfg.LLMBackend = v
		llmClient.SetBackend(v)
	}

	var backendCfg llm.BackendConfig
	if v, _ := presets.GetSetting("llm_keep_alive"); v != "" {
		backendCfg.KeepAlive = v
	} else {
		backendCfg.KeepAlive = "5m"
	}
	if v, _ := presets.GetSetting("llm_num_ctx"); v != "" {
		fmt.Sscanf(v, "%d", &backendCfg.NumCtx)
	} else {
		backendCfg.NumCtx = 4096
	}
	if v, _ := presets.GetSetting("llm_num_gpu"); v != "" {
		fmt.Sscanf(v, "%d", &backendCfg.NumGPU)
	}
	llmClient.SetBackendConfig(backendCfg)

	app := NewApp(presets, llmClient, sdClient, cfg)

	if err := wails.Run(&options.App{
		Title:     "SD Studio",
		Width:     1280,
		Height:    800,
		Frameless: false,
		MinWidth:  900,
		MinHeight: 600,
		MaxWidth:  7680,
		MaxHeight: 4320,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:  app.startup,
		Bind: []interface{}{
			app,
		},
		Linux: &linux.Options{
			WebviewGpuPolicy: linux.WebviewGpuPolicyAlways,
		},
	}); err != nil {
		log.Fatalf("Error: %v", err)
	}
}
