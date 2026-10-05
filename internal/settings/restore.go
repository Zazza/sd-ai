package settings

import (
	"fmt"

	"go-sd/internal/config"
	"go-sd/internal/llm"
	"go-sd/internal/preset"
	"go-sd/internal/sd"
	"go-sd/internal/serverclient"
)

func Restore(db *preset.DB, cfg *config.Config, llmSvc llm.Service, sdSvc sd.Service, srvClient *serverclient.Client) {
	if v, _ := db.GetSetting("llm_url"); v != "" {
		cfg.LLMUrl = v
		llmSvc.SetURL(v)
	}
	if v, _ := db.GetSetting("sd_url"); v != "" {
		cfg.SDUrl = v
		sdSvc.SetURL(v)
	}
	if v, _ := db.GetSetting("llm_model"); v != "" {
		cfg.LLMModel = v
	}
	if v, _ := db.GetSetting("sd_prompt_model"); v != "" {
		cfg.SDPromptModel = v
	}
	if v, _ := db.GetSetting("vision_model"); v != "" {
		cfg.VisionModel = v
	}
	if v, _ := db.GetSetting("llm_backend"); v != "" {
		cfg.LLMBackend = v
		llmSvc.SetBackend(v)
	}

	if mode, _ := db.GetSetting("connection_mode"); mode == "server" {
		if serverURL, _ := db.GetSetting("server_url"); serverURL != "" {
			srvClient.SetBaseURL(serverURL)
			sdURL, llmURL := srvClient.ProxyURLs()
			cfg.SDUrl = sdURL
			cfg.LLMUrl = llmURL
			sdSvc.SetURL(sdURL)
			llmSvc.SetURL(llmURL)
			llmSvc.SetBackend("ollama")
		}
	}

	var backendCfg llm.BackendConfig
	if v, _ := db.GetSetting("llm_keep_alive"); v != "" {
		backendCfg.KeepAlive = v
	} else {
		backendCfg.KeepAlive = "5m"
	}
	if v, _ := db.GetSetting("llm_num_ctx"); v != "" {
		fmt.Sscanf(v, "%d", &backendCfg.NumCtx)
	} else {
		backendCfg.NumCtx = 4096
	}
	if v, _ := db.GetSetting("llm_num_gpu"); v != "" {
		fmt.Sscanf(v, "%d", &backendCfg.NumGPU)
	}
	llmSvc.SetBackendConfig(backendCfg)
}
