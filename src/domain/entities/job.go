package entities

import (
	"fmt"
	"time"
)

// JobStatus represents the current state of a job
type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusProcessing JobStatus = "processing"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
)

// JobType represents the type of job
type JobType string

const (
	JobTypeDraftGeneration JobType = "draft_generation"
)

// Job represents an asynchronous job execution
type Job struct {
	ID        string
	UserID    string
	Type      JobType
	Status    JobStatus
	IdeaID    *string
	DraftIDs  []string
	Error     string
	CreatedAt time.Time
	UpdatedAt time.Time
	StartedAt *time.Time
	CompletedAt *time.Time
}

// Validate validates the job entity
func (j *Job) Validate() error {
	if j.ID == "" {
		return fmt.Errorf("job ID cannot be empty")
	}

	if j.UserID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	if !j.isValidType() {
		return fmt.Errorf("invalid job type: %s", j.Type)
	}

	if !j.isValidStatus() {
		return fmt.Errorf("invalid job status: %s", j.Status)
	}

	return nil
}

// isValidType checks if job type is valid
func (j *Job) isValidType() bool {
	return j.Type == JobTypeDraftGeneration
}

// isValidStatus checks if job status is valid
func (j *Job) isValidStatus() bool {
	return j.Status == JobStatusPending ||
		j.Status == JobStatusProcessing ||
		j.Status == JobStatusCompleted ||
		j.Status == JobStatusFailed
}

// MarkAsProcessing marks the job as processing
func (j *Job) MarkAsProcessing() error {
	if j.Status != JobStatusPending {
		return fmt.Errorf("cannot mark job as processing: current status is %s", j.Status)
	}

	now := time.Now()
	j.Status = JobStatusProcessing
	j.StartedAt = &now
	j.UpdatedAt = now

	return nil
}

// MarkAsCompleted marks the job as completed
func (j *Job) MarkAsCompleted(draftIDs []string) error {
	if j.Status != JobStatusProcessing {
		return fmt.Errorf("cannot mark job as completed: current status is %s", j.Status)
	}

	now := time.Now()
	j.Status = JobStatusCompleted
	j.DraftIDs = draftIDs
	j.CompletedAt = &now
	j.UpdatedAt = now

	return nil
}

// MarkAsFailed marks the job as failed
func (j *Job) MarkAsFailed(errorMessage string) error {
	if j.Status == JobStatusCompleted {
		return fmt.Errorf("cannot mark completed job as failed")
	}

	now := time.Now()
	j.Status = JobStatusFailed
	j.Error = errorMessage
	j.CompletedAt = &now
	j.UpdatedAt = now

	return nil
}

// BelongsToUser checks if the job belongs to a specific user
func (j *Job) BelongsToUser(userID string) bool {
	return j.UserID == userID
}

// IsCompleted checks if the job has finished (completed or failed)
func (j *Job) IsCompleted() bool {
	return j.Status == JobStatusCompleted || j.Status == JobStatusFailed
}

// IsPending checks if the job is pending
func (j *Job) IsPending() bool {
	return j.Status == JobStatusPending
}

// IsProcessing checks if the job is processing
func (j *Job) IsProcessing() bool {
	return j.Status == JobStatusProcessing
}

// IsFailed checks if the job failed
func (j *Job) IsFailed() bool {
	return j.Status == JobStatusFailed
}
