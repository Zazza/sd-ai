package generation

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-sd/internal/preset"
)

func strPtr(v string) *string { return &v }

func overrideTestPreset() preset.Preset {
	return preset.Preset{
		Name:           "test-preset",
		Sampler:        "Euler a",
		ScheduleType:   "Karras",
		Steps:          20,
		CfgScale:       7.0,
		ClipSkip:       intPtr(1),
		ModelName:      "sd-xl",
		Loras:          `[{"name":"test-lora","weight":0.8}]`,
	}
}

func TestApplyPresetOverrides(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		mutateBase func(p *preset.Preset)
		ov         *PresetOverrides
		want       func(p *preset.Preset)
	}{
		{
			name: "nil overrides no-op",
			ov:   nil,
			want: func(p *preset.Preset) {},
		},
		{
			name: "empty overrides no-op",
			ov:   &PresetOverrides{},
			want: func(p *preset.Preset) {},
		},
		{
			name: "sampler applied",
			ov:   &PresetOverrides{Sampler: strPtr("DPM++ 2M")},
			want: func(p *preset.Preset) { p.Sampler = "DPM++ 2M" },
		},
		{
			name: "schedule type applied",
			ov:   &PresetOverrides{ScheduleType: strPtr("Exponential")},
			want: func(p *preset.Preset) { p.ScheduleType = "Exponential" },
		},
		{
			name: "steps applied",
			ov:   &PresetOverrides{Steps: intPtr(33)},
			want: func(p *preset.Preset) { p.Steps = 33 },
		},
		{
			name: "cfg scale applied",
			ov:   &PresetOverrides{CfgScale: floatPtr(4.5)},
			want: func(p *preset.Preset) { p.CfgScale = 4.5 },
		},
		{
			name: "clip skip applied over existing value",
			ov:   &PresetOverrides{ClipSkip: intPtr(2)},
			want: func(p *preset.Preset) { p.ClipSkip = intPtr(2) },
		},
		{
			name:       "clip skip applied over nil",
			mutateBase: func(p *preset.Preset) { p.ClipSkip = nil },
			ov:         &PresetOverrides{ClipSkip: intPtr(2)},
			want:       func(p *preset.Preset) { p.ClipSkip = intPtr(2) },
		},
		{
			name: "model name applied",
			ov:   &PresetOverrides{ModelName: strPtr("flux1-dev-fp8")},
			want: func(p *preset.Preset) { p.ModelName = "flux1-dev-fp8" },
		},
		{
			name: "loras empty array clears preset loras",
			ov:   &PresetOverrides{Loras: strPtr("[]")},
			want: func(p *preset.Preset) { p.Loras = "[]" },
		},
		{
			name: "loras replaced",
			ov:   &PresetOverrides{Loras: strPtr(`[{"name":"other-lora","weight":0.5}]`)},
			want: func(p *preset.Preset) { p.Loras = `[{"name":"other-lora","weight":0.5}]` },
		},
		{
			name: "empty sampler no-op",
			ov:   &PresetOverrides{Sampler: strPtr("")},
			want: func(p *preset.Preset) {},
		},
		{
			name: "empty schedule type no-op",
			ov:   &PresetOverrides{ScheduleType: strPtr("")},
			want: func(p *preset.Preset) {},
		},
		{
			name: "zero steps no-op",
			ov:   &PresetOverrides{Steps: intPtr(0)},
			want: func(p *preset.Preset) {},
		},
		{
			name: "negative steps no-op",
			ov:   &PresetOverrides{Steps: intPtr(-5)},
			want: func(p *preset.Preset) {},
		},
		{
			name: "zero cfg scale no-op",
			ov:   &PresetOverrides{CfgScale: floatPtr(0)},
			want: func(p *preset.Preset) {},
		},
		{
			name: "negative cfg scale no-op",
			ov:   &PresetOverrides{CfgScale: floatPtr(-1.5)},
			want: func(p *preset.Preset) {},
		},
		{
			name: "zero clip skip no-op",
			ov:   &PresetOverrides{ClipSkip: intPtr(0)},
			want: func(p *preset.Preset) {},
		},
		{
			name: "negative clip skip no-op",
			ov:   &PresetOverrides{ClipSkip: intPtr(-2)},
			want: func(p *preset.Preset) {},
		},
		{
			name: "empty model name no-op",
			ov:   &PresetOverrides{ModelName: strPtr("")},
			want: func(p *preset.Preset) {},
		},
		{
			name: "empty loras no-op",
			ov:   &PresetOverrides{Loras: strPtr("")},
			want: func(p *preset.Preset) {},
		},
		{
			name: "steps capped at 150",
			ov:   &PresetOverrides{Steps: intPtr(300)},
			want: func(p *preset.Preset) { p.Steps = 150 },
		},
		{
			name: "steps exactly 150 kept",
			ov:   &PresetOverrides{Steps: intPtr(150)},
			want: func(p *preset.Preset) { p.Steps = 150 },
		},
		{
			name: "all fields applied together",
			ov: &PresetOverrides{
				Sampler:      strPtr("DPM++ 2M"),
				ScheduleType: strPtr("SGM Uniform"),
				Steps:        intPtr(40),
				CfgScale:     floatPtr(3.5),
				ClipSkip:     intPtr(2),
				ModelName:    strPtr("flux1-dev-fp8"),
				Loras:        strPtr("[]"),
			},
			want: func(p *preset.Preset) {
				p.Sampler = "DPM++ 2M"
				p.ScheduleType = "SGM Uniform"
				p.Steps = 40
				p.CfgScale = 3.5
				p.ClipSkip = intPtr(2)
				p.ModelName = "flux1-dev-fp8"
				p.Loras = "[]"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := overrideTestPreset()
			want := overrideTestPreset()
			if tt.mutateBase != nil {
				tt.mutateBase(&got)
				tt.mutateBase(&want)
			}
			tt.want(&want)
			ApplyPresetOverrides(&got, tt.ov)
			assert.Equal(t, want, got)
		})
	}
}

func TestApplyPresetOverrides_NilPreset(t *testing.T) {
	t.Parallel()
	require.NotPanics(t, func() {
		ApplyPresetOverrides(nil, &PresetOverrides{Sampler: strPtr("DPM++ 2M")})
	})
}

func TestApplyPresetOverrides_ClipSkipFreshPointer(t *testing.T) {
	t.Parallel()
	ovClipSkip := 2
	p := &preset.Preset{ClipSkip: intPtr(1)}
	ApplyPresetOverrides(p, &PresetOverrides{ClipSkip: &ovClipSkip})
	require.NotNil(t, p.ClipSkip)
	assert.Equal(t, 2, *p.ClipSkip)
	assert.NotSame(t, &ovClipSkip, p.ClipSkip)
}

func TestGenerateImageParams_OverridesJSONRoundTrip(t *testing.T) {
	params := GenerateImageParams{
		PresetID: 7,
		Overrides: &PresetOverrides{
			Sampler:      strPtr("Euler"),
			ScheduleType: strPtr("Simple"),
			Steps:        intPtr(8),
			CfgScale:     floatPtr(1.0),
			ClipSkip:     intPtr(1),
			ModelName:    strPtr("z-image-turbo-fp8-aio"),
			Loras:        strPtr("[]"),
		},
	}
	data, err := json.Marshal(params)
	require.NoError(t, err)

	var back GenerateImageParams
	require.NoError(t, json.Unmarshal(data, &back))
	require.NotNil(t, back.Overrides)
	assert.Equal(t, "Euler", *back.Overrides.Sampler)
	assert.Equal(t, "Simple", *back.Overrides.ScheduleType)
	assert.Equal(t, 8, *back.Overrides.Steps)
	assert.Equal(t, 1.0, *back.Overrides.CfgScale)
	assert.Equal(t, 1, *back.Overrides.ClipSkip)
	assert.Equal(t, "z-image-turbo-fp8-aio", *back.Overrides.ModelName)
	assert.Equal(t, "[]", *back.Overrides.Loras)
}

func TestApplyPresetOverrides_TrimsAndClamps(t *testing.T) {
	base := &preset.Preset{Sampler: "Euler a", CfgScale: 7.0}
	ApplyPresetOverrides(base, &PresetOverrides{
		Sampler:  strPtr("  Euler  "),
		CfgScale: floatPtr(1e6),
		ClipSkip: intPtr(99),
	})
	assert.Equal(t, "Euler", base.Sampler)
	assert.Equal(t, maxOverrideCfg, base.CfgScale)
	require.NotNil(t, base.ClipSkip)
	assert.Equal(t, maxOverrideClipSkip, *base.ClipSkip)
}
