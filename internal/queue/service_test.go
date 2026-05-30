package queue

import (
	"context"
	"database/sql"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS job_queue (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			type TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			params TEXT NOT NULL DEFAULT '{}',
			progress REAL NOT NULL DEFAULT 0,
			progress_detail TEXT NOT NULL DEFAULT '{}',
			result TEXT NOT NULL DEFAULT '',
			error TEXT NOT NULL DEFAULT '',
			source TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			started_at DATETIME,
			completed_at DATETIME,
			retry_count INTEGER NOT NULL DEFAULT 0,
			max_retries INTEGER NOT NULL DEFAULT 3,
			next_retry_at DATETIME
		);
		CREATE INDEX IF NOT EXISTS idx_job_queue_status ON job_queue(status);
	`)
	if err != nil {
		db.Close()
		t.Fatalf("migrate test db: %v", err)
	}
	return db
}

type mockProcessor struct {
	processFn func(ctx context.Context, job *Job) (*JobResult, error)
}

func (m *mockProcessor) ProcessJob(ctx context.Context, job *Job) (*JobResult, error) {
	if m.processFn != nil {
		return m.processFn(ctx, job)
	}
	return &JobResult{}, nil
}

type mockEmitter struct {
	mu     sync.Mutex
	events []string
}

func (m *mockEmitter) Emit(event string, data ...any) {
	m.mu.Lock()
	m.events = append(m.events, event)
	m.mu.Unlock()
}

func (m *mockEmitter) hasEvent(event string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range m.events {
		if e == event {
			return true
		}
	}
	return false
}

func (m *mockEmitter) countEvent(event string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, e := range m.events {
		if e == event {
			n++
		}
	}
	return n
}

func insertJob(t *testing.T, db *sql.DB, status JobStatus) int64 {
	t.Helper()
	result, err := db.Exec(
		`INSERT INTO job_queue (type, status, params, source) VALUES (?, ?, '{}', 'test')`,
		JobTxt2Img, status,
	)
	if err != nil {
		t.Fatalf("insert test job: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("last insert id: %v", err)
	}
	return id
}

func getJobStatus(t *testing.T, db *sql.DB, id int64) JobStatus {
	t.Helper()
	var status string
	err := db.QueryRow(`SELECT status FROM job_queue WHERE id = ?`, id).Scan(&status)
	if err != nil {
		t.Fatalf("get job status: %v", err)
	}
	return JobStatus(status)
}

func getJobError(t *testing.T, db *sql.DB, id int64) string {
	t.Helper()
	var errMsg string
	err := db.QueryRow(`SELECT error FROM job_queue WHERE id = ?`, id).Scan(&errMsg)
	if err != nil {
		t.Fatalf("get job error: %v", err)
	}
	return errMsg
}

func TestCancelJob_InterruptFnCalled(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	jobID := insertJob(t, db, StatusRunning)

	var interruptCalled atomic.Int32
	emitter := &mockEmitter{}

	proc := &mockProcessor{}
	store := NewStore(db)
	svc := NewService(store, proc, emitter)
	svc.SetInterruptFn(func() {
		interruptCalled.Add(1)
	})

	err := svc.CancelJob(jobID)
	if err != nil {
		t.Fatalf("CancelJob returned error: %v", err)
	}

	if interruptCalled.Load() != 1 {
		t.Errorf("interruptFn called %d times, want 1", interruptCalled.Load())
	}
}

func TestCancelQueue_InterruptFnCalled(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	var interruptCalled atomic.Int32
	emitter := &mockEmitter{}

	proc := &mockProcessor{}
	store := NewStore(db)
	svc := NewService(store, proc, emitter)
	svc.SetInterruptFn(func() {
		interruptCalled.Add(1)
	})

	err := svc.CancelQueue()
	if err != nil {
		t.Fatalf("CancelQueue returned error: %v", err)
	}

	if interruptCalled.Load() != 1 {
		t.Errorf("interruptFn called %d times, want 1", interruptCalled.Load())
	}
}

func TestCancelJob_RunningJobMarkedFailed(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	jobID := insertJob(t, db, StatusRunning)

	emitter := &mockEmitter{}
	proc := &mockProcessor{}
	store := NewStore(db)
	svc := NewService(store, proc, emitter)
	svc.SetInterruptFn(func() {})

	err := svc.CancelJob(jobID)
	if err != nil {
		t.Fatalf("CancelJob returned error: %v", err)
	}

	status := getJobStatus(t, db, jobID)
	if status != StatusFailed {
		t.Errorf("job status = %q, want %q", status, StatusFailed)
	}

	errMsg := getJobError(t, db, jobID)
	if errMsg != "cancelled by user" {
		t.Errorf("job error = %q, want %q", errMsg, "cancelled by user")
	}

	if !emitter.hasEvent(EventQueueChanged) {
		t.Error("expected EventQueueChanged to be emitted")
	}
}

func TestCancelJob_PendingJobMarkedCancelled(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	jobID := insertJob(t, db, StatusPending)

	emitter := &mockEmitter{}
	proc := &mockProcessor{}
	store := NewStore(db)
	svc := NewService(store, proc, emitter)

	err := svc.CancelJob(jobID)
	if err != nil {
		t.Fatalf("CancelJob returned error: %v", err)
	}

	status := getJobStatus(t, db, jobID)
	if status != StatusCancelled {
		t.Errorf("job status = %q, want %q", status, StatusCancelled)
	}
}

func TestCancelJob_CompletedJobReturnsError(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	jobID := insertJob(t, db, StatusCompleted)

	emitter := &mockEmitter{}
	proc := &mockProcessor{}
	store := NewStore(db)
	svc := NewService(store, proc, emitter)

	err := svc.CancelJob(jobID)
	if err == nil {
		t.Fatal("expected error when cancelling completed job, got nil")
	}
}

func TestCancelJob_NoInterruptFn_DoesNotPanic(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	jobID := insertJob(t, db, StatusRunning)

	emitter := &mockEmitter{}
	proc := &mockProcessor{}
	store := NewStore(db)
	svc := NewService(store, proc, emitter)

	err := svc.CancelJob(jobID)
	if err != nil {
		t.Fatalf("CancelJob returned error: %v", err)
	}

	status := getJobStatus(t, db, jobID)
	if status != StatusFailed {
		t.Errorf("job status = %q, want %q", status, StatusFailed)
	}
}

func TestCancelQueue_PendingJobsMarkedCancelled(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	id1 := insertJob(t, db, StatusPending)
	id2 := insertJob(t, db, StatusPending)
	_ = insertJob(t, db, StatusRunning)

	emitter := &mockEmitter{}
	proc := &mockProcessor{}
	store := NewStore(db)
	svc := NewService(store, proc, emitter)
	svc.SetInterruptFn(func() {})

	err := svc.CancelQueue()
	if err != nil {
		t.Fatalf("CancelQueue returned error: %v", err)
	}

	if getJobStatus(t, db, id1) != StatusCancelled {
		t.Errorf("job %d status = %q, want %q", id1, getJobStatus(t, db, id1), StatusCancelled)
	}
	if getJobStatus(t, db, id2) != StatusCancelled {
		t.Errorf("job %d status = %q, want %q", id2, getJobStatus(t, db, id2), StatusCancelled)
	}

	if !emitter.hasEvent(EventQueueChanged) {
		t.Error("expected EventQueueChanged to be emitted")
	}
}

func TestCancelJob_NonexistentJob(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	emitter := &mockEmitter{}
	proc := &mockProcessor{}
	store := NewStore(db)
	svc := NewService(store, proc, emitter)

	err := svc.CancelJob(99999)
	if err == nil {
		t.Fatal("expected error for nonexistent job, got nil")
	}
}

func TestWorkerProcessNext_CancelledCtx_MarksJobFailed(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	jobID := insertJob(t, db, StatusPending)

	emitter := &mockEmitter{}
	store := NewStore(db)

	procStarted := make(chan struct{})
	proc := &mockProcessor{
		processFn: func(ctx context.Context, job *Job) (*JobResult, error) {
			close(procStarted)
			<-ctx.Done()
			return &JobResult{}, nil
		},
	}

	worker := NewWorker(store, proc, emitter)

	parentCtx, parentCancel := context.WithCancel(context.Background())
	defer parentCancel()

	doneCh := make(chan struct{})
	go func() {
		worker.loop(parentCtx)
		close(doneCh)
	}()

	worker.Notify()
	<-procStarted

	worker.CancelCurrent()
	time.Sleep(200 * time.Millisecond)

	parentCancel()
	<-doneCh

	status := getJobStatus(t, db, jobID)
	if status != StatusFailed {
		t.Errorf("job status = %q, want %q", status, StatusFailed)
	}
}

func TestWorkerProcessNext_Success(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	jobID := insertJob(t, db, StatusPending)

	emitter := &mockEmitter{}
	store := NewStore(db)

	proc := &mockProcessor{
		processFn: func(ctx context.Context, job *Job) (*JobResult, error) {
			return &JobResult{Info: `{"test":true}`}, nil
		},
	}

	worker := NewWorker(store, proc, emitter)

	parentCtx, parentCancel := context.WithCancel(context.Background())
	defer parentCancel()

	doneCh := make(chan struct{})
	go func() {
		worker.loop(parentCtx)
		close(doneCh)
	}()

	worker.Notify()
	time.Sleep(300 * time.Millisecond)
	parentCancel()
	<-doneCh

	status := getJobStatus(t, db, jobID)
	if status != StatusCompleted {
		t.Errorf("job status = %q, want %q", status, StatusCompleted)
	}
	if !emitter.hasEvent(EventQueueCompleted) {
		t.Error("expected EventQueueCompleted to be emitted")
	}
}

func TestWorkerProcessNext_ProcessorError(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	jobID := insertJob(t, db, StatusPending)

	emitter := &mockEmitter{}
	store := NewStore(db)

	proc := &mockProcessor{
		processFn: func(ctx context.Context, job *Job) (*JobResult, error) {
			return nil, &testError{msg: "fatal generation error"}
		},
	}

	worker := NewWorker(store, proc, emitter)

	parentCtx, parentCancel := context.WithCancel(context.Background())
	defer parentCancel()

	doneCh := make(chan struct{})
	go func() {
		worker.loop(parentCtx)
		close(doneCh)
	}()

	worker.Notify()
	time.Sleep(300 * time.Millisecond)
	parentCancel()
	<-doneCh

	status := getJobStatus(t, db, jobID)
	if status != StatusFailed {
		t.Errorf("job status = %q, want %q", status, StatusFailed)
	}
	if !emitter.hasEvent(EventQueueFailed) {
		t.Error("expected EventQueueFailed to be emitted")
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
