package queue

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

func TestNewQueue(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "queue.db")

	queue, err := NewQueue(dbPath)
	if err != nil {
		t.Fatalf("Failed to create queue: %v", err)
	}
	defer queue.Stop()

	if queue == nil {
		t.Error("Queue should not be nil")
	}
}

func TestAddJob(t *testing.T) {
	queue := setupTestQueue(t)
	defer queue.Stop()

	// Test adding a job
	job, err := queue.AddJob(JobTypeParse, "/test/file.txt", "dir1", 1, `{"test": "data"}`)
	if err != nil {
		t.Fatalf("Failed to add job: %v", err)
	}

	if job.ID == 0 {
		t.Error("Job ID should not be zero")
	}
	if job.Type != JobTypeParse {
		t.Errorf("Expected job type %s, got %s", JobTypeParse, job.Type)
	}
	if job.Status != JobStatusPending {
		t.Errorf("Expected status %s, got %s", JobStatusPending, job.Status)
	}
	if job.FilePath != "/test/file.txt" {
		t.Errorf("Expected file path '/test/file.txt', got '%s'", job.FilePath)
	}
	if job.Priority != 1 {
		t.Errorf("Expected priority 1, got %d", job.Priority)
	}
}

func TestGetJob(t *testing.T) {
	queue := setupTestQueue(t)
	defer queue.Stop()

	// Add a job first
	originalJob, err := queue.AddJob(JobTypeEmbed, "/test/embed.txt", "dir1", 2, `{"embed": "test"}`)
	if err != nil {
		t.Fatalf("Failed to add job: %v", err)
	}

	// Get the job
	retrievedJob, err := queue.GetJob(originalJob.ID)
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}

	if retrievedJob == nil {
		t.Fatal("Retrieved job should not be nil")
	}
	if retrievedJob.ID != originalJob.ID {
		t.Errorf("Expected job ID %d, got %d", originalJob.ID, retrievedJob.ID)
	}
	if retrievedJob.Type != JobTypeEmbed {
		t.Errorf("Expected job type %s, got %s", JobTypeEmbed, retrievedJob.Type)
	}

	// Test getting non-existent job
	nonExistentJob, err := queue.GetJob(9999)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if nonExistentJob != nil {
		t.Error("Non-existent job should return nil")
	}
}

func TestGetNextJob(t *testing.T) {
	queue := setupTestQueue(t)
	defer queue.Stop()

	// Add jobs with different types and priorities
	_, err := queue.AddJob(JobTypeParse, "/test/file1.txt", "dir1", 1, "")
	if err != nil {
		t.Fatalf("Failed to add job 1: %v", err)
	}

	_, err = queue.AddJob(JobTypeEmbed, "/test/file2.txt", "dir1", 3, "")
	if err != nil {
		t.Fatalf("Failed to add job 2: %v", err)
	}

	_, err = queue.AddJob(JobTypeParse, "/test/file3.txt", "dir1", 2, "")
	if err != nil {
		t.Fatalf("Failed to add job 3: %v", err)
	}

	// Get next job (should be highest priority)
	nextJob, err := queue.GetNextJob()
	if err != nil {
		t.Fatalf("Failed to get next job: %v", err)
	}

	if nextJob == nil {
		t.Fatal("Next job should not be nil")
	}
	if nextJob.Priority != 3 {
		t.Errorf("Expected highest priority job (3), got priority %d", nextJob.Priority)
	}
	if nextJob.Status != JobStatusProcessing {
		t.Errorf("Expected status %s, got %s", JobStatusProcessing, nextJob.Status)
	}

	// Get next job with specific type filter
	parseJob, err := queue.GetNextJob(JobTypeParse)
	if err != nil {
		t.Fatalf("Failed to get next parse job: %v", err)
	}

	if parseJob == nil {
		t.Fatal("Parse job should not be nil")
	}
	if parseJob.Type != JobTypeParse {
		t.Errorf("Expected job type %s, got %s", JobTypeParse, parseJob.Type)
	}
}

func TestCompleteJob(t *testing.T) {
	queue := setupTestQueue(t)
	defer queue.Stop()

	// Add a job
	job, err := queue.AddJob(JobTypeIndex, "/test/index.txt", "dir1", 1, "")
	if err != nil {
		t.Fatalf("Failed to add job: %v", err)
	}

	// Get and process the job
	_, err = queue.GetNextJob()
	if err != nil {
		t.Fatalf("Failed to get next job: %v", err)
	}

	// Complete the job
	err = queue.CompleteJob(job.ID)
	if err != nil {
		t.Fatalf("Failed to complete job: %v", err)
	}

	// Verify job status
	completedJob, err := queue.GetJob(job.ID)
	if err != nil {
		t.Fatalf("Failed to get completed job: %v", err)
	}

	if completedJob.Status != JobStatusCompleted {
		t.Errorf("Expected status %s, got %s", JobStatusCompleted, completedJob.Status)
	}
	if completedJob.CompletedAt == nil {
		t.Error("CompletedAt should not be nil")
	}
}

func TestFailJob(t *testing.T) {
	queue := setupTestQueue(t)
	defer queue.Stop()

	// Add a job
	job, err := queue.AddJob(JobTypeParse, "/test/fail.txt", "dir1", 1, "")
	if err != nil {
		t.Fatalf("Failed to add job: %v", err)
	}

	// Set max_retries to 0 so it fails immediately
	_, err = queue.db.Exec("UPDATE jobs SET max_retries = 0 WHERE id = ?", job.ID)
	if err != nil {
		t.Fatalf("Failed to update max_retries: %v", err)
	}

	// Get and process the job
	_, err = queue.GetNextJob()
	if err != nil {
		t.Fatalf("Failed to get next job: %v", err)
	}

	// Fail the job
	errorMsg := "Test error message"
	err = queue.FailJob(job.ID, errorMsg)
	if err != nil {
		t.Fatalf("Failed to fail job: %v", err)
	}

	// Verify job status
	failedJob, err := queue.GetJob(job.ID)
	if err != nil {
		t.Fatalf("Failed to get failed job: %v", err)
	}

	if failedJob.Status != JobStatusFailed {
		t.Errorf("Expected status %s, got %s", JobStatusFailed, failedJob.Status)
	}
	if failedJob.Error != errorMsg {
		t.Errorf("Expected error '%s', got '%s'", errorMsg, failedJob.Error)
	}
}

func TestGetQueueStats(t *testing.T) {
	queue := setupTestQueue(t)
	defer queue.Stop()

	// Add jobs with different statuses
	job1, _ := queue.AddJob(JobTypeParse, "/test/1.txt", "dir1", 1, "")
	job2, _ := queue.AddJob(JobTypeEmbed, "/test/2.txt", "dir1", 1, "")
	_, _ = queue.AddJob(JobTypeIndex, "/test/3.txt", "dir1", 1, "")

	// Process and complete one job
	queue.GetNextJob()
	queue.CompleteJob(job1.ID)

	// Process and fail another job
	queue.GetNextJob()
	// Set max_retries to 0 so it fails immediately
	queue.db.Exec("UPDATE jobs SET max_retries = 0 WHERE id = ?", job2.ID)
	queue.FailJob(job2.ID, "test error")

	// Get stats
	stats, err := queue.GetQueueStats()
	if err != nil {
		t.Fatalf("Failed to get queue stats: %v", err)
	}

	if stats[JobStatusPending] != 1 {
		t.Errorf("Expected 1 pending job, got %d", stats[JobStatusPending])
	}
	if stats[JobStatusCompleted] != 1 {
		t.Errorf("Expected 1 completed job, got %d", stats[JobStatusCompleted])
	}
	if stats[JobStatusFailed] != 1 {
		t.Errorf("Expected 1 failed job, got %d", stats[JobStatusFailed])
	}
}

func TestGetPendingJobCount(t *testing.T) {
	queue := setupTestQueue(t)
	defer queue.Stop()

	// Initially should have 0 pending jobs
	count, err := queue.GetPendingJobCount()
	if err != nil {
		t.Fatalf("Failed to get pending job count: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 pending jobs, got %d", count)
	}

	// Add some jobs
	queue.AddJob(JobTypeParse, "/test/1.txt", "dir1", 1, "")
	queue.AddJob(JobTypeEmbed, "/test/2.txt", "dir1", 1, "")

	// Should have 2 pending jobs
	count, err = queue.GetPendingJobCount()
	if err != nil {
		t.Fatalf("Failed to get pending job count: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 pending jobs, got %d", count)
	}

	// Process one job
	job, _ := queue.GetNextJob()
	if job != nil {
		// Should still have 1 pending job (the processing one doesn't count as pending)
		count, err = queue.GetPendingJobCount()
		if err != nil {
			t.Fatalf("Failed to get pending job count after processing: %v", err)
		}
		if count != 1 {
			t.Errorf("Expected 1 pending job after processing, got %d", count)
		}
	}
}

func TestJobPriority(t *testing.T) {
	queue := setupTestQueue(t)
	defer queue.Stop()

	// Add jobs with different priorities
	_, err := queue.AddJob(JobTypeParse, "/test/low.txt", "dir1", 1, "")
	if err != nil {
		t.Fatalf("Failed to add low priority job: %v", err)
	}

	_, err = queue.AddJob(JobTypeParse, "/test/high.txt", "dir1", 5, "")
	if err != nil {
		t.Fatalf("Failed to add high priority job: %v", err)
	}

	_, err = queue.AddJob(JobTypeParse, "/test/medium.txt", "dir1", 3, "")
	if err != nil {
		t.Fatalf("Failed to add medium priority job: %v", err)
	}

	// Jobs should be processed in priority order (highest first)
	expectedPriorities := []int{5, 3, 1}
	for i, expectedPriority := range expectedPriorities {
		job, err := queue.GetNextJob()
		if err != nil {
			t.Fatalf("Failed to get job %d: %v", i, err)
		}
		if job == nil {
			t.Fatalf("Job %d should not be nil", i)
		}
		if job.Priority != expectedPriority {
			t.Errorf("Job %d: expected priority %d, got %d", i, expectedPriority, job.Priority)
		}
	}
}

func TestJobRetries(t *testing.T) {
	queue := setupTestQueue(t)
	defer queue.Stop()

	// Add a job
	job, err := queue.AddJob(JobTypeParse, "/test/retry.txt", "dir1", 1, "")
	if err != nil {
		t.Fatalf("Failed to add job: %v", err)
	}

	// Initial retries should be 0
	if job.Retries != 0 {
		t.Errorf("Expected 0 retries, got %d", job.Retries)
	}

	// Process and fail the job multiple times
	for i := 1; i <= 3; i++ {
		// Get the job
		processingJob, err := queue.GetNextJob()
		if err != nil {
			t.Fatalf("Failed to get job for retry %d: %v", i, err)
		}

		if processingJob == nil {
			t.Fatalf("Job should not be nil for retry %d", i)
		}

		// Fail the job
		err = queue.FailJob(processingJob.ID, "retry test")
		if err != nil {
			t.Fatalf("Failed to fail job for retry %d: %v", i, err)
		}

		// Check retry count
		failedJob, err := queue.GetJob(processingJob.ID)
		if err != nil {
			t.Fatalf("Failed to get failed job for retry %d: %v", i, err)
		}

		if failedJob.Retries != i {
			t.Errorf("Retry %d: expected %d retries, got %d", i, i, failedJob.Retries)
		}
	}
}

func TestCleanupOldJobs(t *testing.T) {
	queue := setupTestQueue(t)
	defer queue.Stop()

	// Add and complete some jobs
	job1, _ := queue.AddJob(JobTypeParse, "/test/old1.txt", "dir1", 1, "")
	job2, _ := queue.AddJob(JobTypeEmbed, "/test/old2.txt", "dir1", 1, "")

	// Process and complete jobs
	queue.GetNextJob()
	queue.CompleteJob(job1.ID)
	queue.GetNextJob()
	queue.CompleteJob(job2.ID)

	// Initially should have jobs
	stats, _ := queue.GetQueueStats()
	if stats[JobStatusCompleted] != 2 {
		t.Errorf("Expected 2 completed jobs before cleanup, got %d", stats[JobStatusCompleted])
	}

	// Cleanup old jobs (cleanup anything older than 1 nanosecond)
	err := queue.CleanupOldJobs(1 * time.Nanosecond)
	if err != nil {
		t.Fatalf("Failed to cleanup old jobs: %v", err)
	}

	// Should have fewer completed jobs after cleanup
	statsAfter, _ := queue.GetQueueStats()
	if statsAfter[JobStatusCompleted] >= stats[JobStatusCompleted] {
		t.Error("Cleanup should have removed some completed jobs")
	}
}

func TestConcurrentJobProcessing(t *testing.T) {
	queue := setupTestQueue(t)
	defer queue.Stop()

	// Add multiple jobs
	jobCount := 10
	for i := 0; i < jobCount; i++ {
		_, err := queue.AddJob(JobTypeParse, fmt.Sprintf("/test/file%d.txt", i), "dir1", 1, "")
		if err != nil {
			t.Fatalf("Failed to add job %d: %v", i, err)
		}
	}

	// Process jobs concurrently
	done := make(chan bool, jobCount)
	for i := 0; i < 3; i++ { // 3 concurrent workers
		go func() {
			for {
				job, err := queue.GetNextJob()
				if err != nil {
					break
				}
				if job == nil {
					break
				}
				// Simulate work
				time.Sleep(1 * time.Millisecond)
				queue.CompleteJob(job.ID)
				done <- true
			}
		}()
	}

	// Wait for all jobs to complete
	completedJobs := 0
	timeout := time.After(5 * time.Second)
	for completedJobs < jobCount {
		select {
		case <-done:
			completedJobs++
		case <-timeout:
			t.Fatalf("Timeout waiting for jobs to complete. Completed: %d/%d", completedJobs, jobCount)
		}
	}

	// Verify all jobs were completed
	stats, err := queue.GetQueueStats()
	if err != nil {
		t.Fatalf("Failed to get final stats: %v", err)
	}

	if stats[JobStatusCompleted] != jobCount {
		t.Errorf("Expected %d completed jobs, got %d", jobCount, stats[JobStatusCompleted])
	}
}

// Helper function to set up a test queue
func setupTestQueue(t *testing.T) *Queue {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_queue.db")

	queue, err := NewQueue(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test queue: %v", err)
	}

	return queue
}