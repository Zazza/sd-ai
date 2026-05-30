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
		return p.processTxt2Img(ctx, job)
	case JobFromImage:
		return p.processFromImage(ctx, job)
	case JobCompound:
		return p.processCompound(ctx, job)
	case JobCompareItem:
		return p.processCompareItem(ctx, job)
	default:
		return nil, fmt.Errorf("unknown job type: %s", job.Type)
	}
}

func (p *processor) processTxt2Img(ctx context.Context, job *Job) (*JobResult, error) {
	var params generation.GenerateImageParams
	if err := json.Unmarshal([]byte(job.Params), &params); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	p.emit.Emit("sd:progress:start")
	result, err := p.gen.GenerateImage(params)
	p.emit.Emit("sd:progress:stop")

	if err != nil {
		return nil, err
	}

	var imageBase64 string
	if img, ok := result.Image.(string); ok {
		imageBase64 = img
	}
	infoJSON, _ := json.Marshal(result.Info)

	return &JobResult{
		ImageBase64: imageBase64,
		Info:        string(infoJSON),
	}, nil
}

func (p *processor) processFromImage(ctx context.Context, job *Job) (*JobResult, error) {
	var params generation.GenerateFromImageParams
	if err := json.Unmarshal([]byte(job.Params), &params); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	p.emit.Emit("sd:progress:start")
	result, err := p.gen.GenerateFromImage(params)
	p.emit.Emit("sd:progress:stop")

	if err != nil {
		return nil, err
	}

	var imageBase64 string
	if img, ok := result.Image.(string); ok {
		imageBase64 = img
	}
	infoJSON, _ := json.Marshal(result.Info)

	return &JobResult{
		ImageBase64: imageBase64,
		Info:        string(infoJSON),
	}, nil
}

func (p *processor) processCompound(ctx context.Context, job *Job) (*JobResult, error) {
	var params generation.GenerateCompoundImageParams
	if err := json.Unmarshal([]byte(job.Params), &params); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	p.emit.Emit("sd:progress:start")
	result, err := p.gen.GenerateCompoundImage(params)
	p.emit.Emit("sd:progress:stop")

	if err != nil {
		return nil, err
	}

	var imageBase64 string
	if img, ok := result.Image.(string); ok {
		imageBase64 = img
	}
	infoJSON, _ := json.Marshal(result.Info)

	return &JobResult{
		ImageBase64: imageBase64,
		Info:        string(infoJSON),
	}, nil
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
