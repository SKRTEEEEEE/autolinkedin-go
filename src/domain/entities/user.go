package entities

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// User represents a LinkedIn user in the system
type User struct {
	ID            string
	Email         string
	LinkedInToken string
	APIKeys       map[string]string
	Configuration map[string]interface{}
	CreatedAt     time.Time
	UpdatedAt     time.Time
	Active        bool
}

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// ValidateCredentials checks if user has valid credentials for publishing
func (u *User) ValidateCredentials() error {
	if u.LinkedInToken == "" {
		return fmt.Errorf("missing LinkedIn token")
	}

	if u.APIKeys == nil || len(u.APIKeys) == 0 {
		return fmt.Errorf("missing API keys")
	}

	return nil
}

// CanPublish checks if user can publish to LinkedIn
func (u *User) CanPublish() bool {
	return u.Active && u.LinkedInToken != ""
}

// IsActive checks if user is active
func (u *User) IsActive() bool {
	return u.Active
}

// UpdateConfiguration updates user configuration
func (u *User) UpdateConfiguration(newConfig map[string]interface{}) error {
	if newConfig == nil || len(newConfig) == 0 {
		return fmt.Errorf("configuration cannot be empty or nil")
	}

	// Merge configurations
	if u.Configuration == nil {
		u.Configuration = make(map[string]interface{})
	}

	for key, value := range newConfig {
		u.Configuration[key] = value
	}

	u.UpdatedAt = time.Now()
	return nil
}

// Validate validates the user entity
func (u *User) Validate() error {
	if u.ID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	if u.Email == "" {
		return fmt.Errorf("email cannot be empty")
	}

	if !isValidEmail(u.Email) {
		return fmt.Errorf("invalid email format")
	}

	if u.CreatedAt.IsZero() {
		return fmt.Errorf("created timestamp cannot be zero")
	}

	if !u.UpdatedAt.IsZero() && u.UpdatedAt.Before(u.CreatedAt) {
		return fmt.Errorf("updated timestamp cannot be before created timestamp")
	}

	return nil
}

// isValidEmail validates email format
func isValidEmail(email string) bool {
	if strings.Contains(email, " ") {
		return false
	}

	if strings.Count(email, "@") != 1 {
		return false
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return false
	}

	return emailRegex.MatchString(email)
}
