package queue

import (
	"testing"

	"go-sd/internal/generation"
)

func TestValidateJobParams(t *testing.T) {
	tests := []struct {
		name    string
		jobType JobType
		params  any
		wantErr string
	}{
		{"txt2img without preset", JobTxt2Img, generation.GenerateImageParams{}, "preset is required"},
		{"txt2img with preset", JobTxt2Img, generation.GenerateImageParams{PresetID: 5}, ""},
		{"compound without preset", JobCompound, generation.GenerateCompoundImageParams{}, "compound preset is required"},
		{"compound with preset", JobCompound, generation.GenerateCompoundImageParams{CompoundPresetID: 3}, ""},
		{"from image preset mode without preset", JobFromImage, generation.GenerateFromImageParams{GenMode: "preset"}, "preset is required"},
		{"from image preset mode with preset", JobFromImage, generation.GenerateFromImageParams{GenMode: "preset", PresetID: 5}, ""},
		{"from image compound mode without preset", JobFromImage, generation.GenerateFromImageParams{GenMode: "compound"}, "compound preset is required"},
		{"from image compound mode with preset", JobFromImage, generation.GenerateFromImageParams{GenMode: "compound", CompoundPresetID: 3}, ""},
		{"from image empty gen mode", JobFromImage, generation.GenerateFromImageParams{}, ""},
		{"from image unknown gen mode", JobFromImage, generation.GenerateFromImageParams{GenMode: "analyze"}, ""},
		{"from image remove object without preset", JobFromImage, generation.GenerateFromImageParams{GenMode: "preset", RemoveObject: true}, ""},
		{"compare item untouched", JobCompareItem, generation.TestGenerateParams{}, ""},
		{"unknown params type untouched", JobTxt2Img, nil, ""},
		{"pointer params type untouched", JobTxt2Img, &generation.GenerateImageParams{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateJobParams(tt.jobType, tt.params)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if err.Error() != tt.wantErr {
				t.Errorf("error = %q, want %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestEnqueue_RejectsInvalidParams(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	emitter := &mockEmitter{}
	proc := &mockProcessor{}
	store := NewStore(db)
	svc := NewService(store, proc, emitter)

	_, err := svc.Enqueue(JobTxt2Img, generation.GenerateImageParams{}, "generate")
	if err == nil || err.Error() != "preset is required" {
		t.Fatalf("error = %v, want %q", err, "preset is required")
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM job_queue`).Scan(&count); err != nil {
		t.Fatalf("count jobs: %v", err)
	}
	if count != 0 {
		t.Errorf("job count = %d, want 0", count)
	}

	if emitter.hasEvent(EventQueueChanged) {
		t.Error("expected no EventQueueChanged for rejected job")
	}
}
