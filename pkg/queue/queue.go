package queue

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// JobType represents the type of job
type JobType string

const (
	JobTypeParse  JobType = "parse"
	JobTypeEmbed  JobType = "embed"
	JobTypeIndex  JobType = "index"
)

// JobStatus represents the status of a job
type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusProcessing JobStatus = "processing"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
)

// Job represents a job in the queue
type Job struct {
	ID          int64     `json:"id"`
	Type        JobType   `json:"type"`
	Status      JobStatus `json:"status"`
	Priority    int       `json:"priority"`
	FilePath    string    `json:"file_path"`
	DirectoryID string    `json:"directory_id"`
	Payload     string    `json:"payload"` // JSON payload for job-specific data
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Error       string    `json:"error,omitempty"`
	Retries     int       `json:"retries"`
	MaxRetries  int       `json:"max_retries"`
}

// Queue manages job processing
type Queue struct {
	db     *sql.DB
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.RWMutex
}

// NewQueue creates a new job queue
func NewQueue(dbPath string) (*Queue, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	q := &Queue{
		db:     db,
		ctx:    ctx,
		cancel: cancel,
	}

	// Initialize database schema
	if err := q.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	// Resume any interrupted jobs
	if err := q.resumeInterruptedJobs(); err != nil {
		log.Printf("Warning: Failed to resume interrupted jobs: %v", err)
	}

	return q, nil
}

// initSchema creates the necessary tables
func (q *Queue) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS jobs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		type TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		priority INTEGER NOT NULL DEFAULT 0,
		file_path TEXT NOT NULL,
		directory_id TEXT NOT NULL,
		payload TEXT,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		started_at DATETIME,
		completed_at DATETIME,
		error TEXT,
		retries INTEGER NOT NULL DEFAULT 0,
		max_retries INTEGER NOT NULL DEFAULT 3
	);

	CREATE INDEX IF NOT EXISTS idx_jobs_status_priority ON jobs(status, priority DESC);
	CREATE INDEX IF NOT EXISTS idx_jobs_file_path ON jobs(file_path);
	CREATE INDEX IF NOT EXISTS idx_jobs_directory_id ON jobs(directory_id);
	CREATE INDEX IF NOT EXISTS idx_jobs_type ON jobs(type);
	`

	_, err := q.db.Exec(schema)
	return err
}

// AddJob adds a new job to the queue
func (q *Queue) AddJob(jobType JobType, filePath, directoryID string, priority int, payload string) (*Job, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Check if there's already a pending job for this file
	var existingID int64
	err := q.db.QueryRow(
		"SELECT id FROM jobs WHERE file_path = ? AND status IN ('pending', 'processing') AND type = ? LIMIT 1",
		filePath, jobType,
	).Scan(&existingID)

	if err == nil {
		// Job already exists, return existing job
		return q.GetJob(existingID)
	}

	// Insert new job
	result, err := q.db.Exec(`
		INSERT INTO jobs (type, file_path, directory_id, priority, payload, max_retries)
		VALUES (?, ?, ?, ?, ?, ?)
	`, jobType, filePath, directoryID, priority, payload, 3)

	if err != nil {
		return nil, fmt.Errorf("failed to insert job: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get job ID: %w", err)
	}

	return q.GetJob(id)
}

// GetJob retrieves a job by ID
func (q *Queue) GetJob(id int64) (*Job, error) {
	job := &Job{}
	var startedAt, completedAt sql.NullTime
	var errorMsg sql.NullString

	err := q.db.QueryRow(`
		SELECT id, type, status, priority, file_path, directory_id, payload,
		       created_at, updated_at, started_at, completed_at, error, retries, max_retries
		FROM jobs WHERE id = ?
	`, id).Scan(
		&job.ID, &job.Type, &job.Status, &job.Priority, &job.FilePath,
		&job.DirectoryID, &job.Payload, &job.CreatedAt, &job.UpdatedAt,
		&startedAt, &completedAt, &errorMsg, &job.Retries, &job.MaxRetries,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Return nil job and nil error when no job is found
		}
		return nil, err
	}

	if startedAt.Valid {
		job.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}
	if errorMsg.Valid {
		job.Error = errorMsg.String
	}

	return job, nil
}

// GetNextJob retrieves the next job to process
func (q *Queue) GetNextJob(jobTypes ...JobType) (*Job, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	typeFilter := ""
	args := []interface{}{}

	if len(jobTypes) > 0 {
		typeFilter = "AND type IN ("
		for i, jt := range jobTypes {
			if i > 0 {
				typeFilter += ","
			}
			typeFilter += "?"
			args = append(args, jt)
		}
		typeFilter += ")"
	}

	query := fmt.Sprintf(`
		SELECT id FROM jobs
		WHERE status = 'pending' %s
		ORDER BY priority DESC, created_at ASC
		LIMIT 1
	`, typeFilter)

	var id int64
	err := q.db.QueryRow(query, args...).Scan(&id)
	if err != nil {
		return nil, err
	}

	// Mark job as processing
	_, err = q.db.Exec(`
		UPDATE jobs
		SET status = 'processing', started_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, id)

	if err != nil {
		return nil, err
	}

	return q.GetJob(id)
}

// CompleteJob marks a job as completed
func (q *Queue) CompleteJob(id int64) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	_, err := q.db.Exec(`
		UPDATE jobs
		SET status = 'completed', completed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, id)

	return err
}

// FailJob marks a job as failed
func (q *Queue) FailJob(id int64, errorMsg string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Check if we should retry
	var retries, maxRetries int
	err := q.db.QueryRow("SELECT retries, max_retries FROM jobs WHERE id = ?", id).Scan(&retries, &maxRetries)
	if err != nil {
		return err
	}

	if retries < maxRetries {
		// Retry the job
		_, err = q.db.Exec(`
			UPDATE jobs
			SET status = 'pending', retries = retries + 1, error = ?, updated_at = CURRENT_TIMESTAMP
			WHERE id = ?
		`, errorMsg, id)
	} else {
		// Mark as failed
		_, err = q.db.Exec(`
			UPDATE jobs
			SET status = 'failed', error = ?, completed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
			WHERE id = ?
		`, errorMsg, id)
	}

	return err
}

// GetQueueStats returns queue statistics
func (q *Queue) GetQueueStats() (map[JobStatus]int, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	stats := make(map[JobStatus]int)
	rows, err := q.db.Query("SELECT status, COUNT(*) FROM jobs GROUP BY status")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var status JobStatus
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		stats[status] = count
	}

	return stats, nil
}

// CleanupOldJobs removes completed and failed jobs older than the specified duration
func (q *Queue) CleanupOldJobs(olderThan time.Duration) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	cutoff := time.Now().Add(-olderThan)
	_, err := q.db.Exec(`
		DELETE FROM jobs
		WHERE status IN ('completed', 'failed')
		AND completed_at < ?
	`, cutoff)

	return err
}

// StartWorkers starts the specified number of worker goroutines
func (q *Queue) StartWorkers(numWorkers int, processor JobProcessor) {
	for i := 0; i < numWorkers; i++ {
		q.wg.Add(1)
		go q.worker(i, processor)
	}
}

// JobProcessor interface for processing jobs
type JobProcessor interface {
	ProcessJob(ctx context.Context, job *Job) error
}

// worker processes jobs from the queue
func (q *Queue) worker(id int, processor JobProcessor) {
	defer q.wg.Done()

	log.Printf("Worker %d started", id)
	defer log.Printf("Worker %d stopped", id)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-q.ctx.Done():
			return
		case <-ticker.C:
			job, err := q.GetNextJob()
			if err != nil {
				if err != sql.ErrNoRows {
					log.Printf("Worker %d: Error getting next job: %v", id, err)
				}
				continue
			}

			log.Printf("Worker %d: Processing job %d (%s) for file %s", id, job.ID, job.Type, job.FilePath)

			if err := processor.ProcessJob(q.ctx, job); err != nil {
				log.Printf("Worker %d: Job %d failed: %v", id, job.ID, err)
				if failErr := q.FailJob(job.ID, err.Error()); failErr != nil {
					log.Printf("Worker %d: Failed to mark job %d as failed: %v", id, job.ID, failErr)
				}
			} else {
				log.Printf("Worker %d: Job %d completed successfully", id, job.ID)
				if completeErr := q.CompleteJob(job.ID); completeErr != nil {
					log.Printf("Worker %d: Failed to mark job %d as completed: %v", id, job.ID, completeErr)
				}
			}
		}
	}
}

// resumeInterruptedJobs resets any jobs that were processing when the system shut down
func (q *Queue) resumeInterruptedJobs() error {
	// Reset any jobs that were marked as processing back to pending
	_, err := q.db.Exec(`
		UPDATE jobs
		SET status = 'pending', started_at = NULL, updated_at = CURRENT_TIMESTAMP
		WHERE status = 'processing'
	`)
	if err != nil {
		return fmt.Errorf("failed to resume interrupted jobs: %w", err)
	}

	// Log how many jobs were resumed
	var count int
	err = q.db.QueryRow("SELECT COUNT(*) FROM jobs WHERE status = 'pending'").Scan(&count)
	if err == nil && count > 0 {
		log.Printf("Resumed %d pending jobs from previous session", count)
	}

	return nil
}

// GetPendingJobCount returns the number of pending jobs
func (q *Queue) GetPendingJobCount() (int, error) {
	var count int
	err := q.db.QueryRow("SELECT COUNT(*) FROM jobs WHERE status = 'pending'").Scan(&count)
	return count, err
}

// Stop stops the queue and all workers
func (q *Queue) Stop() {
	q.cancel()
	q.wg.Wait()
	if q.db != nil {
		q.db.Close()
	}
}