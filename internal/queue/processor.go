package queue

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"go-sd/internal/generation"
)

type Processor interface {
	ProcessJob(ctx context.Context, job *Job) (*JobResult, error)
}

type EventEmitter interface {
	Emit(event string, data ...any)
}

type processor struct {
	gen     *generation.Service
	store   *Store
	dataDir string
	emit    EventEmitter
}

func NewProcessor(gen *generation.Service, store *Store, dataDir string, emit EventEmitter) Processor {
	return &processor{
		gen:     gen,
		store:   store,
		dataDir: dataDir,
		emit:    emit,
	}
}

func (p *processor) ProcessJob(ctx context.Context, job *Job) (*JobResult, error) {
	switch job.Type {
	case JobTxt2Img:
		return p.processTxt2Img(job)
	case JobFromImage:
		return p.processFromImage(job)
	case JobCompound:
		return p.processCompound(job)
	case JobCompareItem:
		return p.processCompareItem(ctx, job)
	default:
		return nil, fmt.Errorf("unknown job type: %s", job.Type)
	}
}

func (p *processor) processGeneration(job *Job, gen func() (*generation.GenerateImageResult, error)) (*JobResult, error) {
	p.emit.Emit("sd:progress:start")
	result, err := gen()
	p.emit.Emit("sd:progress:stop")
	if err != nil {
		return nil, err
	}
	return &JobResult{
		ImageBase64: result.Image,
		Info:        string(result.Info),
	}, nil
}

func (p *processor) processTxt2Img(job *Job) (*JobResult, error) {
	return p.processGeneration(job, func() (*generation.GenerateImageResult, error) {
		var params generation.GenerateImageParams
		if err := json.Unmarshal([]byte(job.Params), &params); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return p.gen.GenerateImage(params)
	})
}

func (p *processor) processFromImage(job *Job) (*JobResult, error) {
	return p.processGeneration(job, func() (*generation.GenerateImageResult, error) {
		var params generation.GenerateFromImageParams
		if err := json.Unmarshal([]byte(job.Params), &params); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return p.gen.GenerateFromImage(params)
	})
}

func (p *processor) processCompound(job *Job) (*JobResult, error) {
	return p.processGeneration(job, func() (*generation.GenerateImageResult, error) {
		var params generation.GenerateCompoundImageParams
		if err := json.Unmarshal([]byte(job.Params), &params); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return p.gen.GenerateCompoundImage(params)
	})
}

func (p *processor) processCompareItem(ctx context.Context, job *Job) (*JobResult, error) {
	var peek struct {
		Mode string `json:"mode"`
	}
	json.Unmarshal([]byte(job.Params), &peek)

	var results []generation.TestGenerateResultItem
	var err error

	if peek.Mode == "compounds" {
		var params generation.TestCompoundGenerateParams
		if err := json.Unmarshal([]byte(job.Params), &params); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		results, err = p.gen.TestCompoundGenerate(params)
	} else {
		var params generation.TestGenerateParams
		if err := json.Unmarshal([]byte(job.Params), &params); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		results, err = p.gen.TestGenerate(params)
	}

	if err != nil {
		return nil, err
	}

	resultJSON, _ := json.Marshal(results)

	var imageBase64 string
	if len(results) > 0 && results[0].Image != "" {
		imageBase64 = results[0].Image
	}

	return &JobResult{
		ImageBase64: imageBase64,
		Info:        string(resultJSON),
	}, nil
}

func decodeBase64Image(b64 string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(b64)
}
