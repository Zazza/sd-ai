package queue

import (
	"context"
	"encoding/json"
	"fmt"
)

type Service struct {
	store      *Store
	worker     *Worker
	emit       EventEmitter
	interruptFn func()
}

func NewService(store *Store, processor Processor, emit EventEmitter) *Service {
	worker := NewWorker(store, processor, emit)
	return &Service{
		store:  store,
		worker: worker,
		emit:   emit,
	}
}

func (s *Service) Start(ctx context.Context) {
	s.store.ResetRunningToPending()
	s.worker.Start(ctx)
	s.worker.Notify()
}

func (s *Service) Enqueue(jobType JobType, params any, source string) (int64, error) {
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return 0, fmt.Errorf("marshal params: %w", err)
	}

	id, err := s.store.Create(jobType, string(paramsJSON), source)
	if err != nil {
		return 0, err
	}

	s.worker.Notify()
	s.emit.Emit(EventQueueChanged)
	return id, nil
}

func (s *Service) RemoveJob(id int64) error {
	err := s.store.Delete(id)
	if err != nil {
		return err
	}
	s.emit.Emit(EventQueueChanged)
	return nil
}

func (s *Service) SetInterruptFn(fn func()) {
	s.interruptFn = fn
}

func (s *Service) CancelJob(id int64) error {
	job, err := s.store.Get(id)
	if err != nil || job == nil {
		return fmt.Errorf("job not found")
	}

	if job.Status == StatusRunning {
		if s.interruptFn != nil {
			s.interruptFn()
		}
		s.worker.CancelCurrent()
		_ = s.store.UpdateStatus(id, StatusFailed, "cancelled by user")
		s.emit.Emit(EventQueueChanged)
		return nil
	}

	if job.Status == StatusPending {
		if err := s.store.UpdateStatus(id, StatusCancelled, ""); err != nil {
			return err
		}
		s.emit.Emit(EventQueueChanged)
		return nil
	}

	return fmt.Errorf("cannot cancel job with status %s", job.Status)
}

func (s *Service) PauseQueue() {
	s.worker.Pause()
}

func (s *Service) ResumeQueue() {
	s.worker.Resume()
}

func (s *Service) IsPaused() bool {
	return s.worker.IsPaused()
}

func (s *Service) CancelQueue() error {
	if s.interruptFn != nil {
		s.interruptFn()
	}
	s.worker.CancelCurrent()
	if cur := s.worker.CurrentJob(); cur != nil {
		_ = s.store.UpdateStatus(cur.ID, StatusFailed, "cancelled by user")
	}
	if err := s.store.CancelPending(); err != nil {
		return err
	}
	s.emit.Emit(EventQueueChanged)
	return nil
}

func (s *Service) GetQueue() ([]*Job, error) {
	return s.store.ListRecent(100)
}

func (s *Service) ClearCompleted() error {
	if err := s.store.ClearCompleted(); err != nil {
		return err
	}
	s.emit.Emit(EventQueueChanged)
	return nil
}

func (s *Service) ResumePausedJobs() (int, error) {
	count, err := s.store.ResumePausedJobs()
	if err != nil {
		return 0, err
	}
	if count > 0 {
		s.worker.Notify()
		s.emit.Emit(EventQueueChanged)
	}
	return count, nil
}

func (s *Service) HasPausedJobs() (bool, error) {
	return s.store.HasPausedJobs()
}

