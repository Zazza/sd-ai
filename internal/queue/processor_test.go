package queue

import (
	"encoding/json"
	"strings"
	"testing"

	"go-sd/internal/generation"
)

func TestProcessGeneration_QualityFlagsInJobResult(t *testing.T) {
	cases := []struct {
		name              string
		result            generation.GenerateImageResult
		wantUpscaleSkip   bool
		wantRedenoiseSkip bool
	}{
		{
			name: "no skips",
		},
		{
			name:            "upscale skipped",
			result:          generation.GenerateImageResult{QualityUpscaleSkipped: true},
			wantUpscaleSkip: true,
		},
		{
			name:              "redenoise skipped",
			result:            generation.GenerateImageResult{QualityRedenoiseSkipped: true},
			wantRedenoiseSkip: true,
		},
		{
			name:              "both skipped",
			result:            generation.GenerateImageResult{QualityUpscaleSkipped: true, QualityRedenoiseSkipped: true},
			wantUpscaleSkip:   true,
			wantRedenoiseSkip: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := &processor{emit: &mockEmitter{}}
			res, err := p.processGeneration(&Job{}, func() (*generation.GenerateImageResult, error) {
				return &tc.result, nil
			})
			if err != nil {
				t.Fatalf("processGeneration returned error: %v", err)
			}
			if res.QualityUpscaleSkipped != tc.wantUpscaleSkip {
				t.Errorf("QualityUpscaleSkipped = %v, want %v", res.QualityUpscaleSkipped, tc.wantUpscaleSkip)
			}
			if res.QualityRedenoiseSkipped != tc.wantRedenoiseSkip {
				t.Errorf("QualityRedenoiseSkipped = %v, want %v", res.QualityRedenoiseSkipped, tc.wantRedenoiseSkip)
			}
		})
	}
}

func TestEventPayload_QualityFlagsSerialization(t *testing.T) {
	cases := []struct {
		name   string
		result *JobResult
		want   []string
	}{
		{
			name:   "flags present",
			result: &JobResult{QualityUpscaleSkipped: true, QualityRedenoiseSkipped: true},
			want:   []string{`"quality_upscale_skipped":true`, `"quality_redenoise_skipped":true`},
		},
		{
			name:   "flags omitted",
			result: &JobResult{ImageBase64: "img"},
			want:   []string{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(EventPayload{JobID: 1, Result: tc.result})
			if err != nil {
				t.Fatalf("marshal EventPayload: %v", err)
			}
			encoded := string(data)
			for _, w := range tc.want {
				if !strings.Contains(encoded, w) {
					t.Errorf("payload JSON %s does not contain %s", encoded, w)
				}
			}
			if tc.name == "flags omitted" {
				if strings.Contains(encoded, "quality_upscale_skipped") || strings.Contains(encoded, "quality_redenoise_skipped") {
					t.Errorf("payload JSON %s contains omitted quality flags", encoded)
				}
			}
		})
	}
}
