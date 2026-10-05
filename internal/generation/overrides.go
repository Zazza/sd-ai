package generation

import (
	"strings"

	"go-sd/internal/preset"
)

const (
	maxOverrideSteps    = 150
	maxOverrideCfg      = 100.0
	maxOverrideClipSkip = 12
)

type PresetOverrides struct {
	Sampler      *string  `json:"sampler,omitempty"`
	ScheduleType *string  `json:"schedule_type,omitempty"`
	Steps        *int     `json:"steps,omitempty"`
	CfgScale     *float64 `json:"cfg_scale,omitempty"`
	ClipSkip     *int     `json:"clip_skip,omitempty"`
	ModelName    *string  `json:"model_name,omitempty"`
	Loras        *string  `json:"loras,omitempty"`
}

func OverrideModelName(ov *PresetOverrides) string {
	if ov == nil || ov.ModelName == nil {
		return ""
	}
	return *ov.ModelName
}

func ApplyPresetOverrides(p *preset.Preset, ov *PresetOverrides) {
	if p == nil || ov == nil {
		return
	}
	if ov.Sampler != nil {
		if v := strings.TrimSpace(*ov.Sampler); v != "" {
			p.Sampler = v
		}
	}
	if ov.ScheduleType != nil {
		if v := strings.TrimSpace(*ov.ScheduleType); v != "" {
			p.ScheduleType = v
		}
	}
	if ov.Steps != nil && *ov.Steps > 0 {
		steps := *ov.Steps
		if steps > maxOverrideSteps {
			steps = maxOverrideSteps
		}
		p.Steps = steps
	}
	if ov.CfgScale != nil && *ov.CfgScale > 0 {
		cfg := *ov.CfgScale
		if cfg > maxOverrideCfg {
			cfg = maxOverrideCfg
		}
		p.CfgScale = cfg
	}
	if ov.ClipSkip != nil && *ov.ClipSkip > 0 {
		clipSkip := *ov.ClipSkip
		if clipSkip > maxOverrideClipSkip {
			clipSkip = maxOverrideClipSkip
		}
		p.ClipSkip = &clipSkip
	}
	if ov.ModelName != nil {
		if v := strings.TrimSpace(*ov.ModelName); v != "" {
			p.ModelName = v
		}
	}
	if ov.Loras != nil && *ov.Loras != "" {
		p.Loras = *ov.Loras
	}
}
