package queue

import (
	"database/sql"
	"fmt"
	"math"
	"strings"
	"time"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

const jobColumns = `id, type, status, params, progress, progress_detail, result, error, source, created_at, started_at, completed_at, retry_count, max_retries, next_retry_at`

func scanJob(scanner interface{ Scan(...any) error }, j *Job) error {
	var nextRetry sql.NullString
	err := scanner.Scan(
		&j.ID, &j.Type, &j.Status, &j.Params, &j.Progress, &j.ProgressDetail,
		&j.Result, &j.Error, &j.Source, &j.CreatedAt, &j.StartedAt, &j.CompletedAt,
		&j.RetryCount, &j.MaxRetries, &nextRetry,
	)
	if err != nil {
		return err
	}
	if nextRetry.Valid {
		j.NextRetry = &nextRetry.String
	}
	return nil
}

func (s *Store) Create(jobType JobType, params, source string) (int64, error) {
	result, err := s.db.Exec(
		`INSERT INTO job_queue (type, status, params, source) VALUES (?, 'pending', ?, ?)`,
		jobType, params, source,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (s *Store) ClaimNext() (*Job, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}

	var job Job
	err = scanJob(tx.QueryRow(
		fmt.Sprintf(`SELECT %s FROM job_queue
			WHERE status = 'pending' AND (next_retry_at IS NULL OR next_retry_at <= CURRENT_TIMESTAMP)
			ORDER BY created_at ASC LIMIT 1`, jobColumns),
	), &job)

	if err == sql.ErrNoRows {
		tx.Rollback()
		return nil, nil
	}
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	_, err = tx.Exec(
		`UPDATE job_queue SET status = 'running', started_at = CURRENT_TIMESTAMP, next_retry_at = NULL WHERE id = ? AND status = 'pending'`,
		job.ID,
	)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	job.Status = StatusRunning
	return &job, nil
}

func (s *Store) UpdateStatus(id int64, status JobStatus, errMsg string) error {
	if errMsg != "" {
		_, err := s.db.Exec(
			`UPDATE job_queue SET status = ?, error = ?, completed_at = CURRENT_TIMESTAMP WHERE id = ?`,
			status, errMsg, id,
		)
		return err
	}
	_, err := s.db.Exec(
		`UPDATE job_queue SET status = ?, completed_at = CURRENT_TIMESTAMP WHERE id = ?`,
		status, id,
	)
	return err
}

func (s *Store) UpdateCompleted(id int64, result string) error {
	_, err := s.db.Exec(
		`UPDATE job_queue SET status = 'completed', result = ?, progress = 1, completed_at = CURRENT_TIMESTAMP WHERE id = ?`,
		result, id,
	)
	return err
}

func (s *Store) UpdateProgress(id int64, progress float64, detail string) error {
	_, err := s.db.Exec(
		`UPDATE job_queue SET progress = ?, progress_detail = ? WHERE id = ?`,
		progress, detail, id,
	)
	return err
}

func (s *Store) ListActive() ([]*Job, error) {
	rows, err := s.db.Query(
		fmt.Sprintf(`SELECT %s FROM job_queue WHERE status IN ('pending', 'running', 'paused') ORDER BY created_at ASC`, jobColumns),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*Job
	for rows.Next() {
		var j Job
		if err := scanJob(rows, &j); err != nil {
			return nil, err
		}
		jobs = append(jobs, &j)
	}
	return jobs, rows.Err()
}

func (s *Store) ListRecent(limit int) ([]*Job, error) {
	rows, err := s.db.Query(
		fmt.Sprintf(`SELECT %s FROM job_queue ORDER BY created_at DESC LIMIT ?`, jobColumns), limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*Job
	for rows.Next() {
		var j Job
		if err := scanJob(rows, &j); err != nil {
			return nil, err
		}
		jobs = append(jobs, &j)
	}
	return jobs, rows.Err()
}

func (s *Store) Get(id int64) (*Job, error) {
	var j Job
	err := s.db.QueryRow(
		fmt.Sprintf(`SELECT %s FROM job_queue WHERE id = ?`, jobColumns), id,
	).Scan(&j.ID, &j.Type, &j.Status, &j.Params, &j.Progress, &j.ProgressDetail,
		&j.Result, &j.Error, &j.Source, &j.CreatedAt, &j.StartedAt, &j.CompletedAt,
		&j.RetryCount, &j.MaxRetries, &j.NextRetry)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &j, nil
}

func (s *Store) Delete(id int64) error {
	_, err := s.db.Exec(
		`DELETE FROM job_queue WHERE id = ? AND status IN ('pending', 'failed', 'cancelled', 'paused')`, id,
	)
	if err != nil {
		return err
	}
	return nil
}

func (s *Store) CancelPending() error {
	_, err := s.db.Exec(
		`UPDATE job_queue SET status = 'cancelled', completed_at = CURRENT_TIMESTAMP WHERE status = 'pending'`,
	)
	return err
}

func (s *Store) ResetRunningToPending() error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	rows, err := tx.Query(
		`SELECT id, retry_count, max_retries FROM job_queue WHERE status = 'running'`,
	)
	if err != nil {
		tx.Rollback()
		return err
	}

	type runningJob struct {
		id         int64
		retryCount int
		maxRetries int
	}
	var running []runningJob
	for rows.Next() {
		var rj runningJob
		if err := rows.Scan(&rj.id, &rj.retryCount, &rj.maxRetries); err != nil {
			rows.Close()
			tx.Rollback()
			return err
		}
		running = append(running, rj)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		tx.Rollback()
		return err
	}

	for _, rj := range running {
		newRetryCount := rj.retryCount + 1
		if newRetryCount >= rj.maxRetries {
			if _, err := tx.Exec(
				`UPDATE job_queue SET status = 'paused', error = ?, retry_count = ? WHERE id = ?`,
				"paused: max retries reached after restart", newRetryCount, rj.id,
			); err != nil {
				tx.Rollback()
				return err
			}
		} else {
			nextRetry := time.Now().Add(backoffDuration(newRetryCount))
			if _, err := tx.Exec(
				`UPDATE job_queue SET status = 'pending', retry_count = ?, next_retry_at = ?, error = '' WHERE id = ?`,
				newRetryCount, nextRetry.UTC().Format("2006-01-02 15:04:05"), rj.id,
			); err != nil {
				tx.Rollback()
				return err
			}
		}
	}
	return tx.Commit()
}

func (s *Store) IncrementRetry(id int64, retryCount, maxRetries int) error {
	newRetryCount := retryCount + 1
	if newRetryCount >= maxRetries {
		_, err := s.db.Exec(
			`UPDATE job_queue SET status = 'paused', retry_count = ?, error = 'paused: max retries reached', completed_at = CURRENT_TIMESTAMP WHERE id = ?`,
			newRetryCount, id,
		)
		return err
	}
	nextRetry := time.Now().Add(backoffDuration(newRetryCount))
	_, err := s.db.Exec(
		`UPDATE job_queue SET status = 'pending', retry_count = ?, next_retry_at = ?, error = '' WHERE id = ?`,
		newRetryCount, nextRetry.UTC().Format("2006-01-02 15:04:05"), id,
	)
	return err
}

func (s *Store) ResumePausedJobs() (int, error) {
	result, err := s.db.Exec(
		`UPDATE job_queue SET status = 'pending', retry_count = 0, next_retry_at = NULL, error = '' WHERE status = 'paused'`,
	)
	if err != nil {
		return 0, err
	}
	affected, _ := result.RowsAffected()
	return int(affected), nil
}

func (s *Store) HasPausedJobs() (bool, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM job_queue WHERE status = 'paused'`).Scan(&count)
	return count > 0, err
}

func (s *Store) ClearCompleted() error {
	_, err := s.db.Exec(`DELETE FROM job_queue WHERE status IN ('completed', 'cancelled')`)
	return err
}

func backoffDuration(retryCount int) time.Duration {
	seconds := math.Pow(2, float64(retryCount)) * 5
	if seconds > 60 {
		seconds = 60
	}
	return time.Duration(seconds) * time.Second
}

func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "eof") ||
		strings.Contains(msg, "sd is not available") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "network is unreachable") ||
		strings.Contains(msg, "cannot assign requested address")
}

