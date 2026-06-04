package generation

import (
	"encoding/json"
	"fmt"

	"go-sd/internal/preset"
	"go-sd/internal/sd"
)

func (s *Service) resolveStepResolution(step preset.CompoundPresetStep, p *preset.Preset, fallbackResolutionID *int64) (width int, height int) {
	if step.ResolutionID != nil && *step.ResolutionID > 0 {
		if r, err := s.db.GetResolution(*step.ResolutionID); err == nil {
			width = r.Width
			height = r.Height
		}
	}
	if width == 0 || height == 0 {
		width, height = s.resolveResolution(p, fallbackResolutionID)
	}
	return width, height
}

func (s *Service) runCompoundFirstStep(step preset.CompoundPresetStep, p *preset.Preset, prompt, negPrompt, samplerName string, width, height, clipSkip int, isSingleStep bool, hiresProfileID *int64, stepNum int) (image string, info json.RawMessage, err error) {
	batchSize := 1
	batchCount := 1
	var hiresFix *bool
	var hiresUpscale *float64
	var hiresDenoising *float64
	var hiresUpscaler string
	if isSingleStep {
		hiresEnabled, hu, hd, hup := s.resolveHires(hiresProfileID)
		if hiresEnabled {
			hiresFix = &hiresEnabled
			hiresUpscale = hu
			hiresDenoising = hd
			hiresUpscaler = hup
		}
	}
	req := sd.Txt2ImgRequest{
		Prompt:                 prompt,
		NegativePrompt:         negPrompt,
		SamplerName:            samplerName,
		Scheduler:              p.ScheduleType,
		Steps:                  p.Steps,
		CfgScale:               p.CfgScale,
		Width:                  width,
		Height:                 height,
		Seed:                   p.Seed,
		ClipSkip:               &clipSkip,
		BatchSize:              &batchSize,
		BatchCount:             &batchCount,
		HiresFix:               hiresFix,
		HiresUpscale:           hiresUpscale,
		HiresDenoisingStrength: hiresDenoising,
		HiresUpscaler:          hiresUpscaler,
		DoNotSaveImages:        true,
		DoNotSaveGrid:          true,
	}
	result, err := s.sd.Txt2Img(req)
	if err != nil {
		if ierr := s.checkSDInterrupted(); ierr != nil {
			return "", nil, ierr
		}
		fb := s.doHiresFallback(req, err, hiresUpscale, hiresDenoising, hiresUpscaler, fmt.Sprintf("compound step %d", stepNum))
		result = fb.Result
		err = fb.Err
		if err != nil {
			return "", nil, fmt.Errorf("step %d (txt2img): %w", stepNum, err)
		}
	}
	if len(result.Images) == 0 {
		if ierr := s.checkSDInterrupted(); ierr != nil {
			return "", nil, ierr
		}
		return "", nil, fmt.Errorf("step %d: no image returned", stepNum)
	}
	return result.Images[0], result.Info, nil
}

func (s *Service) runCompoundStep(step preset.CompoundPresetStep, p *preset.Preset, lastImage, prompt, negPrompt, samplerName string, width, height, clipSkip int, stepNum int) (image string, info json.RawMessage, err error) {
	denoising := step.DenoisingStrength
	if denoising <= 0 {
		denoising = 0.5
	}
	batchSize := 1
	batchCount := 1
	result, err := s.sd.Img2Img(sd.Img2ImgRequest{
		InitImages:      []string{lastImage},
		Prompt:          prompt,
		NegativePrompt:  negPrompt,
		SamplerName:     samplerName,
		Scheduler:       p.ScheduleType,
		Steps:           p.Steps,
		CfgScale:        p.CfgScale,
		Width:           width,
		Height:          height,
		Seed:            p.Seed,
		DenoisingStrength: &denoising,
		ClipSkip:        &clipSkip,
		BatchSize:       &batchSize,
		BatchCount:      &batchCount,
		DoNotSaveImages: true,
		DoNotSaveGrid:   true,
	})
	if err != nil {
		if ierr := s.checkSDInterrupted(); ierr != nil {
			return "", nil, ierr
		}
		return "", nil, fmt.Errorf("step %d (img2img): %w", stepNum, err)
	}
	if len(result.Images) == 0 {
		if ierr := s.checkSDInterrupted(); ierr != nil {
			return "", nil, ierr
		}
		return "", nil, fmt.Errorf("step %d: no image returned", stepNum)
	}
	return result.Images[0], result.Info, nil
}

func (s *Service) applyHiresOnLastStep(p *preset.Preset, lastImage, prompt, negPrompt, samplerName string, width, height, clipSkip, batchSize, batchCount int, hiresProfileID *int64) (string, json.RawMessage, bool) {
	hiresEnabled, hiresUpscale, hiresDenoising, hiresUpscaler := s.resolveHires(hiresProfileID)
	if !hiresEnabled {
		return lastImage, nil, false
	}
	scale := 2.0
	if hiresUpscale != nil {
		scale = *hiresUpscale
	}
	ds := 0.5
	if hiresDenoising != nil {
		ds = *hiresDenoising
	}
	s.log.Info("compound last step: hires upscale %.1fx, denoise=%.2f, upscaler=%s", scale, ds, hiresUpscaler)
	hrResult, hrErr := s.manualHiresUpscale(lastImage, sd.Txt2ImgRequest{
		Prompt:         prompt,
		NegativePrompt: negPrompt,
		SamplerName:    samplerName,
		Scheduler:      p.ScheduleType,
		Steps:          p.Steps,
		CfgScale:       p.CfgScale,
		Width:          width,
		Height:         height,
		Seed:           p.Seed,
		ClipSkip:       &clipSkip,
		BatchSize:      &batchSize,
		BatchCount:     &batchCount,
	}, scale, ds, hiresUpscaler)
	if hrErr != nil {
		s.log.Warn("compound last step: hires upscale failed, using base image: %s", hrErr)
		return lastImage, nil, false
	}
	if len(hrResult.Images) > 0 {
		return hrResult.Images[0], hrResult.Info, true
	}
	return lastImage, nil, false
}

func (s *Service) runFromImageCompoundFirstStep(p *preset.Preset, initImage, prompt, negPrompt, samplerName string, width, height, clipSkip int, mode string, denoisingStrength float64, stepNum int) (image string, info json.RawMessage, err error) {
	batchSize := 1
	batchCount := 1
	if mode == "img2img" {
		denoising := denoisingStrength
		if denoising <= 0 {
			denoising = 0.5
		}
		result, err := s.sd.Img2Img(sd.Img2ImgRequest{
			InitImages:      []string{initImage},
			Prompt:          prompt,
			NegativePrompt:  negPrompt,
			SamplerName:     samplerName,
			Scheduler:       p.ScheduleType,
			Steps:           p.Steps,
			CfgScale:        p.CfgScale,
			Width:           width,
			Height:          height,
			Seed:            p.Seed,
			DenoisingStrength: &denoising,
			ClipSkip:        &clipSkip,
			BatchSize:       &batchSize,
			BatchCount:      &batchCount,
			DoNotSaveImages: true,
			DoNotSaveGrid:   true,
		})
		if err != nil {
			if ierr := s.checkSDInterrupted(); ierr != nil {
				return "", nil, ierr
			}
			return "", nil, fmt.Errorf("step %d (img2img): %w", stepNum, err)
		}
		if len(result.Images) == 0 {
			if ierr := s.checkSDInterrupted(); ierr != nil {
				return "", nil, ierr
			}
			return "", nil, fmt.Errorf("step %d: no image returned", stepNum)
		}
		return result.Images[0], result.Info, nil
	}
	result, err := s.sd.Txt2Img(sd.Txt2ImgRequest{
		Prompt:          prompt,
		NegativePrompt:  negPrompt,
		SamplerName:     samplerName,
		Scheduler:       p.ScheduleType,
		Steps:           p.Steps,
		CfgScale:        p.CfgScale,
		Width:           width,
		Height:          height,
		Seed:            p.Seed,
		ClipSkip:        &clipSkip,
		BatchSize:       &batchSize,
		BatchCount:      &batchCount,
		DoNotSaveImages: true,
		DoNotSaveGrid:   true,
	})
	if err != nil {
		if ierr := s.checkSDInterrupted(); ierr != nil {
			return "", nil, ierr
		}
		return "", nil, fmt.Errorf("step %d (txt2img): %w", stepNum, err)
	}
	if len(result.Images) == 0 {
		if ierr := s.checkSDInterrupted(); ierr != nil {
			return "", nil, ierr
		}
		return "", nil, fmt.Errorf("step %d: no image returned", stepNum)
	}
	return result.Images[0], result.Info, nil
}

func (s *Service) runFromImageCompoundStep(step preset.CompoundPresetStep, p *preset.Preset, lastImage, prompt, negPrompt, samplerName string, width, height, clipSkip int, stepNum int) (image string, info json.RawMessage, err error) {
	denoising := step.DenoisingStrength
	if denoising <= 0 {
		denoising = 0.5
	}
	batchSize := 1
	batchCount := 1
	result, err := s.sd.Img2Img(sd.Img2ImgRequest{
		InitImages:      []string{lastImage},
		Prompt:          prompt,
		NegativePrompt:  negPrompt,
		SamplerName:     samplerName,
		Scheduler:       p.ScheduleType,
		Steps:           p.Steps,
		CfgScale:        p.CfgScale,
		Width:           width,
		Height:          height,
		Seed:            p.Seed,
		DenoisingStrength: &denoising,
		ClipSkip:        &clipSkip,
		BatchSize:       &batchSize,
		BatchCount:      &batchCount,
		DoNotSaveImages: true,
		DoNotSaveGrid:   true,
	})
	if err != nil {
		if ierr := s.checkSDInterrupted(); ierr != nil {
			return "", nil, ierr
		}
		return "", nil, fmt.Errorf("step %d (img2img): %w", stepNum, err)
	}
	if len(result.Images) == 0 {
		if ierr := s.checkSDInterrupted(); ierr != nil {
			return "", nil, ierr
		}
		return "", nil, fmt.Errorf("step %d: no image returned", stepNum)
	}
	return result.Images[0], result.Info, nil
}
