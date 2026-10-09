package queue

import (
	"fmt"

	"go-sd/internal/generation"
)

func validateJobParams(jobType JobType, params any) error {
	switch jobType {
	case JobTxt2Img:
		if p, ok := params.(generation.GenerateImageParams); ok && p.PresetID <= 0 {
			return fmt.Errorf("preset is required")
		}
	case JobCompound:
		if p, ok := params.(generation.GenerateCompoundImageParams); ok && p.CompoundPresetID <= 0 {
			return fmt.Errorf("compound preset is required")
		}
	case JobFromImage:
		if p, ok := params.(generation.GenerateFromImageParams); ok {
			if p.OutputMode != "" && p.OutputMode != "standard" && p.OutputMode != "quality" {
				return fmt.Errorf("output_mode must be standard or quality")
			}
			if p.OutputMode == "quality" && (p.RemoveObject || p.Mode != "img2img" || p.GenMode != "preset" || p.ResolutionID == nil || *p.ResolutionID <= 0) {
				return fmt.Errorf("quality output mode requires img2img preset mode with resolution")
			}
			if !p.RemoveObject {
				if p.GenMode == "preset" && p.PresetID <= 0 {
					return fmt.Errorf("preset is required")
				}
				if p.GenMode == "compound" && p.CompoundPresetID <= 0 {
					return fmt.Errorf("compound preset is required")
				}
			}
		}
	}
	return nil
}
