package factories

import (
	"fmt"
	"time"

	"github.com/linkgen-ai/backend/domain/entities"
)

// NewUser creates a new User entity with validation
func NewUser(id, email string) (*entities.User, error) {
	now := time.Now()

	user := &entities.User{
		ID:            id,
		Email:         email,
		LinkedInToken: "",
		APIKeys:       make(map[string]string),
		Configuration: make(map[string]interface{}),
		CreatedAt:     now,
		UpdatedAt:     now,
		Active:        true,
	}

	if err := user.Validate(); err != nil {
		return nil, fmt.Errorf("user validation failed: %w", err)
	}

	return user, nil
}

// NewUserWithCredentials creates a new User with credentials
func NewUserWithCredentials(id, email, linkedInToken string, apiKeys map[string]string) (*entities.User, error) {
	user, err := NewUser(id, email)
	if err != nil {
		return nil, err
	}

	user.LinkedInToken = linkedInToken

	if apiKeys != nil {
		user.APIKeys = apiKeys
	}

	if err := user.ValidateCredentials(); err != nil {
		return nil, fmt.Errorf("credentials validation failed: %w", err)
	}

	return user, nil
}

// NewUserWithConfig creates a new User with configuration
func NewUserWithConfig(id, email string, config map[string]interface{}) (*entities.User, error) {
	user, err := NewUser(id, email)
	if err != nil {
		return nil, err
	}

	if config != nil {
		user.Configuration = config
	}

	return user, nil
}
