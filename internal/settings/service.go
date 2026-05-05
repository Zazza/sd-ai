package settings

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"go-sd/internal/config"
	"go-sd/internal/llm"
	"go-sd/internal/logger"
	"go-sd/internal/preset"
	"go-sd/internal/rembg"
	"go-sd/internal/sd"
)

type ServiceInfo struct {
	Available   bool   `json:"available"`
	Model       string `json:"model"`
	VisionModel string `json:"vision_model,omitempty"`
}

type ServiceStatus struct {
	LLM ServiceInfo `json:"llm"`
	SD  ServiceInfo `json:"sd"`
}

type Service struct {
	db      *preset.DB
	llm     llm.Service
	sd      sd.Service
	cfg     *config.Config
	rembg   *rembg.Client
	log     *logger.Logger
}

func New(db *preset.DB, llmSvc llm.Service, sdSvc sd.Service, cfg *config.Config, rembgClient *rembg.Client, log *logger.Logger) *Service {
	return &Service{db: db, llm: llmSvc, sd: sdSvc, cfg: cfg, rembg: rembgClient, log: log}
}

func (s *Service) CheckServices() ServiceStatus {
	var status ServiceStatus
	var mu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		var info ServiceInfo
		if err := s.llm.HealthCheck(); err != nil {
			info.Available = false
			s.log.Warn("LLM unavailable: %s", err)
		} else {
			info.Available = true
			if v, err := s.db.GetSetting("llm_generate_model"); err == nil && v != "" {
				info.Model = v
			} else {
				info.Model = s.cfg.SDPromptModel
			}
			if v, err := s.db.GetSetting("llm_analyze_model"); err == nil && v != "" {
				info.VisionModel = v
			} else {
				info.VisionModel = s.cfg.VisionModel
			}
		}
		mu.Lock()
		status.LLM = info
		mu.Unlock()
	}()

	go func() {
		defer wg.Done()
		var info ServiceInfo
		if err := s.sd.HealthCheck(); err != nil {
			info.Available = false
			s.log.Warn("SD unavailable: %s", err)
		} else {
			info.Available = true
			opts, err := s.sd.GetOptions()
			if err == nil {
				if m, ok := opts["sd_model_checkpoint"].(string); ok {
					info.Model = m
				}
			}
		}
		mu.Lock()
		status.SD = info
		mu.Unlock()
	}()

	wg.Wait()
	s.log.Debug("Service check: LLM=%v SD=%v", status.LLM.Available, status.SD.Available)
	return status
}

func (s *Service) CheckRembg() error {
	rembgURL, _ := s.db.GetSetting("rembg_url")
	if rembgURL == "" {
		return fmt.Errorf("rembg URL not configured")
	}
	s.rembg.SetURL(rembgURL)
	err := s.rembg.HealthCheck()
	if err != nil {
		s.log.Error("Rembg check failed: %s", err)
	} else {
		s.log.Info("Rembg connected: %s", rembgURL)
	}
	return err
}

func (s *Service) GetSettings() (map[string]string, error) {
	settings, err := s.db.GetAllSettings()
	if err != nil {
		return nil, err
	}

	defaults := map[string]string{
		"llm_url":                   s.cfg.LLMUrl,
		"sd_url":                    s.cfg.SDUrl,
		"llm_model":                 s.cfg.LLMModel,
		"sd_prompt_model":           s.cfg.SDPromptModel,
		"vision_model":              s.cfg.VisionModel,
		"llm_backend":               s.cfg.LLMBackend,
		"llm_keep_alive":            "5m",
		"llm_num_ctx":               "4096",
		"llm_num_gpu":               "0",
		"llm_max_tokens":            "256",
		"llm_generate_model":        s.cfg.SDPromptModel,
		"llm_analyze_model":         s.cfg.VisionModel,
		"llm_generate_temperature":  "0.4",
		"llm_generate_num_ctx":      "4096",
		"llm_generate_num_predict":  "256",
		"llm_generate_top_p":        "0.9",
		"llm_generate_num_thread":   "0",
		"llm_analyze_temperature":   "0.4",
		"llm_analyze_num_ctx":       "4096",
		"llm_analyze_num_predict":   "256",
		"llm_analyze_top_p":         "0.9",
		"llm_analyze_num_thread":    "0",
		"kids_mode":                 "false",
		"kids_cat_violence":         "true",
		"kids_cat_horror":           "true",
		"kids_cat_weapons":          "true",
		"kids_cat_substances":       "true",
		"kids_cat_mature":           "true",
		"rembg_url":                 "",
		"preview_mode":              "false",
		"preview_width":             "512",
		"preview_height":            "512",
	}
	for k, v := range defaults {
		if _, ok := settings[k]; !ok {
			settings[k] = v
		}
	}
	return settings, nil
}

func (s *Service) UpdateSettings(data map[string]string) error {
	urlFields := map[string]bool{"llm_url": true, "sd_url": true, "rembg_url": true}
	for k, v := range data {
		if urlFields[k] && v != "" {
			if _, err := url.Parse(v); err != nil {
				return fmt.Errorf("invalid %s: %w", k, err)
			}
		}
	}

	numericFields := map[string]bool{
		"llm_num_ctx": true, "llm_num_gpu": true, "llm_max_tokens": true,
		"preview_width": true, "preview_height": true,
	}
	for k, v := range data {
		if numericFields[k] {
			if v == "" {
				data[k] = "0"
			} else if n, err := strconv.Atoi(v); err != nil || n < 0 {
				data[k] = "0"
			}
		}
	}

	for k, v := range data {
		if !config.AllowedSettings[k] {
			continue
		}
		if err := s.db.SetSetting(k, v); err != nil {
			return err
		}
	}

	if v, ok := data["llm_url"]; ok {
		s.llm.SetURL(v)
		s.cfg.LLMUrl = v
	}
	if v, ok := data["sd_url"]; ok {
		s.sd.SetURL(v)
		s.cfg.SDUrl = v
	}
	if v, ok := data["llm_model"]; ok {
		s.cfg.LLMModel = v
	}
	if v, ok := data["sd_prompt_model"]; ok {
		s.cfg.SDPromptModel = v
	}
	if v, ok := data["vision_model"]; ok {
		s.cfg.VisionModel = v
	}
	if v, ok := data["llm_backend"]; ok {
		s.llm.SetBackend(v)
		s.cfg.LLMBackend = v
	}
	if v, ok := data["llm_generate_model"]; ok {
		s.cfg.SDPromptModel = v
	}
	if v, ok := data["llm_analyze_model"]; ok {
		s.cfg.VisionModel = v
	}
	if v, ok := data["rembg_url"]; ok {
		s.rembg.SetURL(v)
	}

	var changed []string
	for k := range data {
		if config.AllowedSettings[k] {
			changed = append(changed, k)
		}
	}
	s.log.UserAction("Settings updated: %s", strings.Join(changed, ", "))

	return nil
}

func (s *Service) ApplyLLMConfig(mode string) {
	prefix := "llm_generate_"
	if mode == "analyze" {
		prefix = "llm_analyze_"
	}

	var cfg llm.BackendConfig
	if v, err := s.db.GetSetting("llm_keep_alive"); err == nil {
		cfg.KeepAlive = v
	}
	if v, err := s.db.GetSetting(prefix + "num_ctx"); err == nil && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.NumCtx = n
		}
	}
	if v, err := s.db.GetSetting(prefix + "num_predict"); err == nil && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.NumPredict = n
		}
	}
	if v, err := s.db.GetSetting(prefix + "top_p"); err == nil && v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.TopP = n
		}
	}
	if v, err := s.db.GetSetting(prefix + "num_thread"); err == nil && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.NumThread = n
		}
	}
	if v, err := s.db.GetSetting("llm_num_gpu"); err == nil && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.NumGPU = n
		}
	}
	s.llm.SetBackendConfig(cfg)
}
