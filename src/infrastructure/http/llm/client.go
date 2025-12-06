package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/linkgen-ai/backend/src/domain/interfaces"
)

// LLMHTTPClient implements the LLMService interface using HTTP
type LLMHTTPClient struct {
	baseURL     string
	httpClient  *http.Client
	retryConfig RetryConfig
	model       string
}

// Config holds configuration for LLM HTTP client
type Config struct {
	BaseURL     string
	Timeout     time.Duration
	MaxRetries  int
	Model       string
}

// LLMRequest represents a request to the LLM API
type LLMRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// LLMResponse represents a response from the LLM API
type LLMResponse struct {
	Choices []Choice `json:"choices"`
	Error   *APIError `json:"error,omitempty"`
}

// Choice represents a completion choice
type Choice struct {
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// APIError represents an error from the LLM API
type APIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

// IdeasResponse represents the response for idea generation
type IdeasResponse struct {
	Ideas []string `json:"ideas"`
}

// DraftsResponse represents the response for draft generation
type DraftsResponse struct {
	Posts    []string `json:"posts"`
	Articles []string `json:"articles"`
}

// RefinementResponse represents the response for draft refinement
type RefinementResponse struct {
	Refined string `json:"refined"`
}

// NewLLMHTTPClient creates a new LLM HTTP client
func NewLLMHTTPClient(config Config) (*LLMHTTPClient, error) {
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Ensure base URL doesn't have trailing slash
	baseURL := strings.TrimSuffix(config.BaseURL, "/")

	// Set default model if not provided
	model := config.Model
	if model == "" {
		model = "gpt-3.5-turbo" // Default model
	}

	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: config.Timeout,
	}

	// Create retry config
	retryConfig := DefaultRetryConfig()
	if config.MaxRetries > 0 {
		retryConfig.MaxRetries = config.MaxRetries
	}

	return &LLMHTTPClient{
		baseURL:     baseURL,
		httpClient:  httpClient,
		retryConfig: retryConfig,
		model:       model,
	}, nil
}

// validateConfig validates the client configuration
func validateConfig(config Config) error {
	if config.BaseURL == "" {
		return fmt.Errorf("base URL cannot be empty")
	}

	// Validate URL format
	parsedURL, err := url.Parse(config.BaseURL)
	if err != nil {
		return fmt.Errorf("invalid base URL: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("base URL must use http or https scheme")
	}

	if config.Timeout <= 0 {
		return fmt.Errorf("timeout must be greater than 0")
	}

	if config.MaxRetries < 0 {
		return fmt.Errorf("max retries cannot be negative")
	}

	return nil
}

// GenerateIdeas implements LLMService.GenerateIdeas
func (c *LLMHTTPClient) GenerateIdeas(ctx context.Context, topic string, count int) ([]string, error) {
	if err := validateIdeaRequest(topic, count); err != nil {
		return nil, err
	}

	prompt := BuildIdeasPrompt(topic, count)
	
	response, err := c.sendRequest(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ideas: %w", err)
	}

	var ideasResp IdeasResponse
	if err := json.Unmarshal([]byte(response), &ideasResp); err != nil {
		return nil, fmt.Errorf("failed to parse ideas response: %w", err)
	}

	if len(ideasResp.Ideas) == 0 {
		return nil, fmt.Errorf("LLM returned empty ideas list")
	}

	return ideasResp.Ideas, nil
}

// GenerateDrafts implements LLMService.GenerateDrafts
func (c *LLMHTTPClient) GenerateDrafts(ctx context.Context, idea string, userContext string) (interfaces.DraftSet, error) {
	if err := validateDraftRequest(idea, userContext); err != nil {
		return interfaces.DraftSet{}, err
	}

	prompt := BuildDraftsPrompt(idea, userContext)
	
	response, err := c.sendRequest(ctx, prompt)
	if err != nil {
		return interfaces.DraftSet{}, fmt.Errorf("failed to generate drafts: %w", err)
	}

	var draftsResp DraftsResponse
	if err := json.Unmarshal([]byte(response), &draftsResp); err != nil {
		return interfaces.DraftSet{}, fmt.Errorf("failed to parse drafts response: %w", err)
	}

	if len(draftsResp.Posts) == 0 {
		return interfaces.DraftSet{}, fmt.Errorf("LLM returned empty posts list")
	}

	if len(draftsResp.Articles) == 0 {
		return interfaces.DraftSet{}, fmt.Errorf("LLM returned empty articles list")
	}

	return interfaces.DraftSet{
		Posts:    draftsResp.Posts,
		Articles: draftsResp.Articles,
	}, nil
}

// RefineDraft implements LLMService.RefineDraft
func (c *LLMHTTPClient) RefineDraft(ctx context.Context, draft string, userPrompt string, history []string) (string, error) {
	if err := validateRefinementRequest(draft, userPrompt); err != nil {
		return "", err
	}

	prompt := BuildRefinementPrompt(draft, userPrompt, history)
	
	response, err := c.sendRequest(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("failed to refine draft: %w", err)
	}

	var refinementResp RefinementResponse
	if err := json.Unmarshal([]byte(response), &refinementResp); err != nil {
		return "", fmt.Errorf("failed to parse refinement response: %w", err)
	}

	if strings.TrimSpace(refinementResp.Refined) == "" {
		return "", fmt.Errorf("LLM returned empty refined content")
	}

	return refinementResp.Refined, nil
}

// sendRequest sends a request to the LLM API with retry logic
func (c *LLMHTTPClient) sendRequest(ctx context.Context, prompt string) (string, error) {
	endpoint := c.baseURL + "/v1/chat/completions"

	llmReq := LLMRequest{
		Model: c.model,
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.7,
		MaxTokens:   2000,
	}

	reqBody, err := json.Marshal(llmReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	var resp *http.Response
	resp, err = ExecuteWithRetry(ctx, c.retryConfig, func() (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(reqBody))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")

		return c.httpClient.Do(req)
	})

	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode >= 400 {
		var apiErr APIError
		if json.Unmarshal(body, &apiErr) == nil && apiErr.Message != "" {
			return "", fmt.Errorf("LLM API error (%d): %s", resp.StatusCode, apiErr.Message)
		}
		return "", fmt.Errorf("LLM API error (%d): %s", resp.StatusCode, string(body))
	}

	var llmResp LLMResponse
	if err := json.Unmarshal(body, &llmResp); err != nil {
		return "", fmt.Errorf("failed to parse LLM response: %w", err)
	}

	if llmResp.Error != nil {
		return "", fmt.Errorf("LLM API error: %s", llmResp.Error.Message)
	}

	if len(llmResp.Choices) == 0 {
		return "", fmt.Errorf("LLM returned no choices")
	}

	content := strings.TrimSpace(llmResp.Choices[0].Message.Content)
	if content == "" {
		return "", fmt.Errorf("LLM returned empty content")
	}

	return content, nil
}

// validateIdeaRequest validates idea generation request parameters
func validateIdeaRequest(topic string, count int) error {
	if strings.TrimSpace(topic) == "" {
		return fmt.Errorf("topic cannot be empty")
	}

	if count <= 0 {
		return fmt.Errorf("count must be greater than 0")
	}

	return nil
}

// validateDraftRequest validates draft generation request parameters
func validateDraftRequest(idea string, userContext string) error {
	if strings.TrimSpace(idea) == "" {
		return fmt.Errorf("idea cannot be empty")
	}

	if strings.TrimSpace(userContext) == "" {
		return fmt.Errorf("user context cannot be empty")
	}

	return nil
}

// validateRefinementRequest validates refinement request parameters
func validateRefinementRequest(draft string, userPrompt string) error {
	if strings.TrimSpace(draft) == "" {
		return fmt.Errorf("draft cannot be empty")
	}

	if strings.TrimSpace(userPrompt) == "" {
		return fmt.Errorf("user prompt cannot be empty")
	}

	return nil
}
