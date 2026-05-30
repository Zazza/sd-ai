package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

type Worker struct {
	store     *Store
	processor Processor
	emit      EventEmitter
	notify    chan struct{}

	mu       sync.Mutex
	paused   bool
	current  *Job
	cancelFn context.CancelFunc
}

func NewWorker(store *Store, processor Processor, emit EventEmitter) *Worker {
	return &Worker{
		store:     store,
		processor: processor,
		emit:      emit,
		notify:    make(chan struct{}, 1),
	}
}

func (w *Worker) Start(ctx context.Context) {
	go w.loop(ctx)
}

func (w *Worker) Notify() {
	select {
	case w.notify <- struct{}{}:
	default:
	}
}

func (w *Worker) Pause() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.paused = true
}

func (w *Worker) Resume() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.paused = false
	w.Notify()
}

func (w *Worker) IsPaused() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.paused
}

func (w *Worker) CancelCurrent() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.cancelFn != nil {
		w.cancelFn()
	}
}

func (w *Worker) CurrentJob() *Job {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.current
}

func (w *Worker) loop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.mu.Lock()
			if !w.paused {
				w.mu.Unlock()
				w.processNext(ctx)
				w.checkMoreJobs()
			} else {
				w.mu.Unlock()
			}
		case <-w.notify:
			w.mu.Lock()
			if w.paused {
				w.mu.Unlock()
				continue
			}
			w.mu.Unlock()

			w.processNext(ctx)
			w.checkMoreJobs()
		}
	}
}

func (w *Worker) checkMoreJobs() {
	if jobs, _ := w.store.ListActive(); len(jobs) > 0 {
		for _, j := range jobs {
			if j.Status == StatusPending {
				w.Notify()
				break
			}
		}
	}
}

func (w *Worker) processNext(parentCtx context.Context) {
	job, err := w.store.ClaimNext()
	if err != nil || job == nil {
		return
	}

	jobCtx, cancel := context.WithCancel(parentCtx)
	w.mu.Lock()
	w.current = job
	w.cancelFn = cancel
	w.mu.Unlock()

	defer func() {
		if r := recover(); r != nil {
			log.Printf("[queue] panic in processNext: %v", r)
			w.store.UpdateStatus(job.ID, StatusFailed, fmt.Sprintf("panic: %v", r))
			w.emit.Emit(EventQueueFailed, EventPayload{JobID: job.ID, Error: fmt.Sprintf("panic: %v", r)})
			w.emit.Emit(EventQueueChanged)
		}
		cancel()
		w.mu.Lock()
		w.current = nil
		w.cancelFn = nil
		w.mu.Unlock()
	}()

	w.emit.Emit(EventQueueStarted, EventPayload{JobID: job.ID, Type: job.Type})
	w.emit.Emit(EventQueueChanged)

	progressCh := make(chan progressUpdate, 16)
	go w.watchProgress(jobCtx, job.ID, progressCh)

	result, procErr := w.processor.ProcessJob(jobCtx, job)

	close(progressCh)

	if jobCtx.Err() != nil {
		if current, _ := w.store.Get(job.ID); current != nil && current.Status == StatusRunning {
			w.store.UpdateStatus(job.ID, StatusFailed, "cancelled")
		}
		w.emit.Emit(EventQueueFailed, EventPayload{JobID: job.ID, Error: "cancelled"})
	} else if procErr != nil {
		errMsg := procErr.Error()
		if IsRetryableError(procErr) && job.RetryCount < job.MaxRetries {
			w.store.IncrementRetry(job.ID, job.RetryCount, job.MaxRetries)
			w.emit.Emit(EventQueuePaused, EventPayload{
				JobID:      job.ID,
				Error:      errMsg,
				RetryCount: job.RetryCount + 1,
				MaxRetries: job.MaxRetries,
			})
			w.Notify()
		} else {
			w.store.UpdateStatus(job.ID, StatusFailed, errMsg)
			w.emit.Emit(EventQueueFailed, EventPayload{JobID: job.ID, Error: errMsg})
		}
	} else {
		resultJSON, _ := json.Marshal(result)
		w.store.UpdateCompleted(job.ID, string(resultJSON))
		w.emit.Emit(EventQueueCompleted, EventPayload{JobID: job.ID, Result: result})
	}
	w.emit.Emit(EventQueueChanged)
}

type progressUpdate struct {
	progress float64
	detail   string
}

func (w *Worker) watchProgress(ctx context.Context, jobID int64, ch chan progressUpdate) {
	for {
		select {
		case <-ctx.Done():
			return
		case upd, ok := <-ch:
			if !ok {
				return
			}
			w.store.UpdateProgress(jobID, upd.progress, upd.detail)
			w.emit.Emit(EventQueueProgress, EventPayload{JobID: jobID, Progress: upd.progress, Detail: upd.detail})
		}
	}
}
