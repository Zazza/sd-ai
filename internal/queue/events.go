package queue

const (
	EventQueueStarted   = "queue:started"
	EventQueueProgress  = "queue:progress"
	EventQueueCompleted = "queue:completed"
	EventQueueFailed    = "queue:failed"
	EventQueuePaused    = "queue:paused"
	EventQueueChanged   = "queue:changed"
)

type EventPayload struct {
	JobID      int64      `json:"job_id"`
	Type       JobType    `json:"type,omitempty"`
	Progress   float64    `json:"progress,omitempty"`
	Detail     string     `json:"detail,omitempty"`
	Result     *JobResult `json:"result,omitempty"`
	Error      string     `json:"error,omitempty"`
	RetryCount int        `json:"retry_count,omitempty"`
	MaxRetries int        `json:"max_retries,omitempty"`
}
