package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"go-sd/internal/config"
	"go-sd/internal/llm"
	"go-sd/internal/preset"
	"go-sd/internal/sd"
)

type Handler struct {
	presets *preset.DB
	llm     *llm.Client
	sd      *sd.Client
	config  *config.Config
}

func NewHandler(presets *preset.DB, llmClient *llm.Client, sdClient *sd.Client, cfg *config.Config) *Handler {
	return &Handler{presets: presets, llm: llmClient, sd: sdClient, config: cfg}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/presets", h.listPresets)
	mux.HandleFunc("GET /api/presets/type/{type}", h.listPresetsByType)
	mux.HandleFunc("GET /api/presets/{id}", h.getPreset)
	mux.HandleFunc("POST /api/presets", h.createPreset)
	mux.HandleFunc("PUT /api/presets/{id}", h.updatePreset)
	mux.HandleFunc("DELETE /api/presets/{id}", h.deletePreset)
	mux.HandleFunc("POST /api/generate-sd-prompt", h.generateSDPrompt)
	mux.HandleFunc("POST /api/generate", h.generateImage)
	mux.HandleFunc("GET /api/sd/models", h.getSDModels)
	mux.HandleFunc("GET /api/sd/samplers", h.getSDSamplers)
	mux.HandleFunc("GET /api/sd/schedulers", h.getSDSchedulers)
	mux.HandleFunc("GET /api/llm/models", h.getLLMModels)
	mux.HandleFunc("GET /api/settings", h.getSettings)
	mux.HandleFunc("PUT /api/settings", h.updateSettings)
	mux.HandleFunc("GET /api/descriptions", h.listDescriptions)
	mux.HandleFunc("POST /api/descriptions", h.createDescription)
	mux.HandleFunc("DELETE /api/descriptions/{id}", h.deleteDescription)
}

func (h *Handler) listPresets(w http.ResponseWriter, r *http.Request) {
	presets, err := h.presets.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if presets == nil {
		presets = []preset.Preset{}
	}
	writeJSON(w, http.StatusOK, presets)
}

func (h *Handler) listPresetsByType(w http.ResponseWriter, r *http.Request) {
	presetType := r.PathValue("type")
	presets, err := h.presets.ListByType(presetType)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if presets == nil {
		presets = []preset.Preset{}
	}
	writeJSON(w, http.StatusOK, presets)
}

func (h *Handler) getPreset(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	p, err := h.presets.Get(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "preset not found")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *Handler) createPreset(w http.ResponseWriter, r *http.Request) {
	var p preset.Preset
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := h.presets.Create(&p); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (h *Handler) updatePreset(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var p preset.Preset
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	p.ID = id
	if err := h.presets.Update(&p); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *Handler) deletePreset(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.presets.Delete(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}

func (h *Handler) generateSDPrompt(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Description string `json:"description"`
		PresetType  string `json:"preset_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	req.Description = strings.TrimSpace(req.Description)
	if req.Description == "" {
		writeError(w, http.StatusBadRequest, "description is required")
		return
	}

	prompt, err := h.llm.GenerateSDPrompt(h.config.SystemPrompt, req.Description, req.PresetType, h.config.SDPromptModel)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "LLM error: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"prompt": prompt})
}

func (h *Handler) generateImage(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PresetID           int64  `json:"preset_id"`
		ExtraPrompt        string `json:"extra_prompt"`
		ExtraNegativePrompt string `json:"extra_negative_prompt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	p, err := h.presets.Get(req.PresetID)
	if err != nil {
		writeError(w, http.StatusNotFound, "preset not found")
		return
	}

	prompt := p.Prompt
	if req.ExtraPrompt != "" {
		prompt += ", " + req.ExtraPrompt
	}

	negativePrompt := p.NegativePrompt
	if req.ExtraNegativePrompt != "" {
		negativePrompt += ", " + req.ExtraNegativePrompt
	}

	if p.ModelName != "" {
		_ = h.sd.SetModel(p.ModelName)
	}

	samplerName := p.Sampler
	if p.ScheduleType != "" {
		samplerName = p.Sampler + " " + p.ScheduleType
	}

	result, err := h.sd.Txt2Img(sd.Txt2ImgRequest{
		Prompt:         prompt,
		NegativePrompt: negativePrompt,
		SamplerName:    samplerName,
		Scheduler:      p.ScheduleType,
		Steps:          p.Steps,
		CfgScale:       p.CfgScale,
		Width:          p.Width,
		Height:         p.Height,
		Seed:           p.Seed,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "SD error: "+err.Error())
		return
	}

	if len(result.Images) == 0 {
		writeError(w, http.StatusInternalServerError, "no images returned")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"image":      result.Images[0],
		"parameters": result.Parameters,
		"info":       result.Info,
	})
}

func (h *Handler) getSDModels(w http.ResponseWriter, r *http.Request) {
	models, err := h.sd.GetModels()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, models)
}

func (h *Handler) getSDSamplers(w http.ResponseWriter, r *http.Request) {
	samplers, err := h.sd.GetSamplers()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, samplers)
}

func (h *Handler) getSDSchedulers(w http.ResponseWriter, r *http.Request) {
	schedulers, err := h.sd.GetSchedulers()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, schedulers)
}

func (h *Handler) getLLMModels(w http.ResponseWriter, r *http.Request) {
	models, err := h.llm.GetModels()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, models)
}

func (h *Handler) getSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := h.presets.GetAllSettings()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	defaults := map[string]string{
		"llm_url":          h.config.LLMUrl,
		"sd_url":           h.config.SDUrl,
		"llm_model":        h.config.LLMModel,
		"sd_prompt_model":  h.config.SDPromptModel,
	}
	for k, v := range defaults {
		if _, ok := settings[k]; !ok {
			settings[k] = v
		}
	}
	writeJSON(w, http.StatusOK, settings)
}

func (h *Handler) updateSettings(w http.ResponseWriter, r *http.Request) {
	var data map[string]string
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	allowed := map[string]bool{
		"llm_url": true, "sd_url": true, "llm_model": true, "sd_prompt_model": true,
	}

	for k, v := range data {
		if !allowed[k] {
			continue
		}
		if err := h.presets.SetSetting(k, v); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	if v, ok := data["llm_url"]; ok {
		h.llm.SetURL(v)
		h.config.LLMUrl = v
	}
	if v, ok := data["sd_url"]; ok {
		h.sd.SetURL(v)
		h.config.SDUrl = v
	}
	if v, ok := data["llm_model"]; ok {
		h.config.LLMModel = v
	}
	if v, ok := data["sd_prompt_model"]; ok {
		h.config.SDPromptModel = v
	}

	writeJSON(w, http.StatusOK, map[string]bool{"saved": true})
}

func (h *Handler) listDescriptions(w http.ResponseWriter, r *http.Request) {
	items, err := h.presets.ListDescriptions()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if items == nil {
		items = []preset.SavedDescription{}
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) createDescription(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.Text = strings.TrimSpace(req.Text)
	if req.Text == "" {
		writeError(w, http.StatusBadRequest, "text is required")
		return
	}
	saved, err := h.presets.CreateDescription(req.Text)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, saved)
}

func (h *Handler) deleteDescription(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.presets.DeleteDescription(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func writeHTML(w http.ResponseWriter, html string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, html)
}
