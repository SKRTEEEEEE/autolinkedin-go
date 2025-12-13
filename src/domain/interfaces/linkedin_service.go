package interfaces

import (
	"context"
)

// LinkedInPostResponse represents the response from LinkedIn after publishing
type LinkedInPostResponse struct {
	ID      string `json:"id"`
	URN     string `json:"urn,omitempty"`
	Success bool   `json:"success"`
}

// LinkedInService defines the contract for LinkedIn API integration
type LinkedInService interface {
	// PublishPost publishes a post to LinkedIn UGC Posts API
	// Returns the LinkedIn post URN/ID or an error
	PublishPost(ctx context.Context, content string, accessToken string) (*LinkedInPostResponse, error)

	// PublishArticle publishes an article to LinkedIn Articles API
	// Returns the LinkedIn article URN/ID or an error
	PublishArticle(ctx context.Context, title string, content string, accessToken string) (*LinkedInPostResponse, error)

	// ValidateToken validates if a LinkedIn access token is valid and not expired
	ValidateToken(ctx context.Context, accessToken string) (bool, error)

	// RefreshToken refreshes an expired LinkedIn access token (if supported)
	RefreshToken(ctx context.Context, refreshToken string) (string, error)
}
