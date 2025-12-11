package valueobjects

import "fmt"

// DraftType represents the type of draft content
type DraftType string

const (
	// DraftTypePost represents a LinkedIn post
	DraftTypePost DraftType = "POST"

	// DraftTypeArticle represents a LinkedIn article
	DraftTypeArticle DraftType = "ARTICLE"
)

// String returns the string representation of DraftType
func (dt DraftType) String() string {
	return string(dt)
}

// IsValid checks if the draft type is valid
func (dt DraftType) IsValid() bool {
	return dt == DraftTypePost || dt == DraftTypeArticle
}

// Validate validates the draft type
func (dt DraftType) Validate() error {
	if !dt.IsValid() {
		return fmt.Errorf("invalid draft type: %s", dt)
	}
	return nil
}

// GetCharacterLimits returns min and max character limits for the type
func (dt DraftType) GetCharacterLimits() (min int, max int, err error) {
	switch dt {
	case DraftTypePost:
		return 10, 3000, nil
	case DraftTypeArticle:
		return 100, 110000, nil
	default:
		return 0, 0, fmt.Errorf("invalid draft type: %s", dt)
	}
}

// RequiresTitle checks if this draft type requires a title
func (dt DraftType) RequiresTitle() bool {
	return dt == DraftTypeArticle
}
