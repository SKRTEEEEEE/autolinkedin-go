package entities

import (
	"fmt"
	"strings"
	"time"
)

// JobErrorStage represents the workflow stage where the job failed
type JobErrorStage string

const (
	// JobErrorStageDraftGeneration indicates a failure during draft generation
	JobErrorStageDraftGeneration JobErrorStage = "draft_generation"
)

// JobError captures detailed information about unexpected job failures
type JobError struct {
	ID          string
	JobID       string
	UserID      string
	IdeaID      *string
	Stage       JobErrorStage
	Error       string
	RawResponse string
	Prompt      string
	Attempt     int
	Metadata    map[string]interface{}
	CreatedAt   time.Time
}

// Validate validates the JobError entity
func (j *JobError) Validate() error {
	if j == nil {
		return fmt.Errorf("job error cannot be nil")
	}

	if strings.TrimSpace(j.JobID) == "" {
		return fmt.Errorf("job error job_id cannot be empty")
	}

	if strings.TrimSpace(j.UserID) == "" {
		return fmt.Errorf("job error user_id cannot be empty")
	}

	if strings.TrimSpace(string(j.Stage)) == "" {
		return fmt.Errorf("job error stage cannot be empty")
	}

	if strings.TrimSpace(j.Error) == "" {
		return fmt.Errorf("job error description cannot be empty")
	}

	if j.Attempt < 0 {
		return fmt.Errorf("job error attempt must be >= 0")
	}

	if j.CreatedAt.IsZero() {
		j.CreatedAt = time.Now()
	}

	return nil
}
