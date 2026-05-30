package queue

type JobType string

const (
	JobTxt2Img      JobType = "txt2img"
	JobFromImage    JobType = "from_image"
	JobCompound     JobType = "compound"
	JobCompareItem  JobType = "compare_item"
)

type JobStatus string

const (
	StatusPending    JobStatus = "pending"
	StatusRunning    JobStatus = "running"
	StatusCompleted  JobStatus = "completed"
	StatusFailed     JobStatus = "failed"
	StatusCancelled  JobStatus = "cancelled"
	StatusPaused     JobStatus = "paused"
)

type Job struct {
	ID             int64     `json:"id"`
	Type           JobType   `json:"type"`
	Status         JobStatus `json:"status"`
	Params         string    `json:"params"`
	Progress       float64   `json:"progress"`
	ProgressDetail string    `json:"progress_detail"`
	Result         string    `json:"result"`
	Error          string    `json:"error"`
	Source         string    `json:"source"`
	CreatedAt      string    `json:"created_at"`
	StartedAt      *string   `json:"started_at,omitempty"`
	CompletedAt    *string   `json:"completed_at,omitempty"`
	RetryCount     int       `json:"retry_count"`
	MaxRetries     int       `json:"max_retries"`
	NextRetry      *string   `json:"next_retry,omitempty"`
}

type JobResult struct {
	ImageBase64 string `json:"image_base64,omitempty"`
	FilePath    string `json:"file_path,omitempty"`
	Info        string `json:"info,omitempty"`
}
