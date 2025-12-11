package valueobjects

import "fmt"

// DraftStatus represents the current state of a draft
type DraftStatus string

const (
	// DraftStatusDraft represents a draft in initial state
	DraftStatusDraft DraftStatus = "DRAFT"

	// DraftStatusRefined represents a draft that has been refined
	DraftStatusRefined DraftStatus = "REFINED"

	// DraftStatusPublished represents a successfully published draft
	DraftStatusPublished DraftStatus = "PUBLISHED"

	// DraftStatusFailed represents a draft that failed to publish
	DraftStatusFailed DraftStatus = "FAILED"
)

// String returns the string representation of DraftStatus
func (ds DraftStatus) String() string {
	return string(ds)
}

// IsValid checks if the draft status is valid
func (ds DraftStatus) IsValid() bool {
	return ds == DraftStatusDraft ||
		ds == DraftStatusRefined ||
		ds == DraftStatusPublished ||
		ds == DraftStatusFailed
}

// Validate validates the draft status
func (ds DraftStatus) Validate() error {
	if !ds.IsValid() {
		return fmt.Errorf("invalid draft status: %s", ds)
	}
	return nil
}

// CanRefine checks if drafts in this status can be refined
func (ds DraftStatus) CanRefine() bool {
	return ds == DraftStatusDraft || ds == DraftStatusRefined
}

// CanPublish checks if drafts in this status can be published
func (ds DraftStatus) CanPublish() bool {
	return ds == DraftStatusDraft || ds == DraftStatusRefined
}

// CanTransitionTo checks if transition to new status is allowed
func (ds DraftStatus) CanTransitionTo(newStatus DraftStatus) bool {
	allowedTransitions := map[DraftStatus][]DraftStatus{
		DraftStatusDraft: {
			DraftStatusRefined,
			DraftStatusPublished,
			DraftStatusFailed,
		},
		DraftStatusRefined: {
			DraftStatusDraft,
			DraftStatusPublished,
			DraftStatusFailed,
		},
		DraftStatusPublished: {},
		DraftStatusFailed: {
			DraftStatusDraft,
		},
	}

	allowed, exists := allowedTransitions[ds]
	if !exists {
		return false
	}

	for _, allowedStatus := range allowed {
		if allowedStatus == newStatus {
			return true
		}
	}

	return false
}
