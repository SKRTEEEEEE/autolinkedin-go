package llm

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"time"
)

// RetryConfig defines retry behavior configuration
type RetryConfig struct {
	MaxRetries      int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	BackoffFactor   float64
	RetryableStatus map[int]bool
}

// DefaultRetryConfig returns default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:    3,
		InitialDelay:  1 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		RetryableStatus: map[int]bool{
			http.StatusRequestTimeout:      true,
			http.StatusTooManyRequests:     true,
			http.StatusInternalServerError: true,
			http.StatusBadGateway:          true,
			http.StatusServiceUnavailable:  true,
			http.StatusGatewayTimeout:      true,
		},
	}
}

// IsRetryable determines if an error or status code should be retried
func (rc *RetryConfig) IsRetryable(statusCode int, err error) bool {
	// Network errors are always retryable
	if err != nil && statusCode == 0 {
		return true
	}

	// Check if status code is in retryable list
	if retryable, exists := rc.RetryableStatus[statusCode]; exists {
		return retryable
	}

	return false
}

// CalculateDelay calculates the delay for a given retry attempt
func (rc *RetryConfig) CalculateDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return rc.InitialDelay
	}

	// Exponential backoff: initialDelay * (backoffFactor ^ attempt)
	delay := float64(rc.InitialDelay) * math.Pow(rc.BackoffFactor, float64(attempt))

	// Cap at max delay
	if delay > float64(rc.MaxDelay) {
		return rc.MaxDelay
	}

	return time.Duration(delay)
}

// RetryableHTTPFunc represents a function that can be retried
type RetryableHTTPFunc func() (*http.Response, error)

// ExecuteWithRetry executes an HTTP function with retry logic
func ExecuteWithRetry(ctx context.Context, config RetryConfig, fn RetryableHTTPFunc) (*http.Response, error) {
	var lastErr error
	var lastStatusCode int

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		// Check context before attempting
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Execute the function
		resp, err := fn()

		// Success case
		if err == nil && resp != nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return resp, nil
		}

		// Record status code and error
		if resp != nil {
			lastStatusCode = resp.StatusCode
		} else {
			lastStatusCode = 0
		}
		lastErr = err

		// Check if we should retry
		if !config.IsRetryable(lastStatusCode, err) {
			// Non-retryable error, fail immediately
			if err != nil {
				return nil, fmt.Errorf("non-retryable error: %w", err)
			}
			return resp, nil
		}

		// If this was the last attempt, don't wait
		if attempt >= config.MaxRetries {
			break
		}

		// Calculate delay and wait
		delay := config.CalculateDelay(attempt)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	// All retries exhausted
	if lastErr != nil {
		return nil, fmt.Errorf("max retries (%d) exceeded: %w", config.MaxRetries, lastErr)
	}

	return nil, fmt.Errorf("max retries (%d) exceeded with status code %d", config.MaxRetries, lastStatusCode)
}
