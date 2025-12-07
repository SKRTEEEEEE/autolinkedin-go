package config

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

var (
	// ErrInvalidPort indicates an invalid port number
	ErrInvalidPort = errors.New("invalid port number")
	// ErrInvalidMongoDBURI indicates an invalid MongoDB URI
	ErrInvalidMongoDBURI = errors.New("invalid MongoDB URI")
	// ErrInvalidNATSURL indicates an invalid NATS URL
	ErrInvalidNATSURL = errors.New("invalid NATS URL")
	// ErrInvalidHTTPURL indicates an invalid HTTP/HTTPS URL
	ErrInvalidHTTPURL = errors.New("invalid HTTP/HTTPS URL")
	// ErrInvalidLogLevel indicates an invalid log level
	ErrInvalidLogLevel = errors.New("invalid log level")
	// ErrInvalidLogFormat indicates an invalid log format
	ErrInvalidLogFormat = errors.New("invalid log format")
	// ErrInvalidPoolSize indicates invalid pool size configuration
	ErrInvalidPoolSize = errors.New("invalid pool size configuration")
	// ErrInvalidSchedulerInterval indicates invalid scheduler interval
	ErrInvalidSchedulerInterval = errors.New("invalid scheduler interval")
	// ErrInvalidBatchSize indicates invalid batch size
	ErrInvalidBatchSize = errors.New("invalid batch size")
	// ErrInvalidTemperature indicates invalid LLM temperature
	ErrInvalidTemperature = errors.New("invalid LLM temperature")
	// ErrMissingRequiredField indicates a required field is missing
	ErrMissingRequiredField = errors.New("missing required field")
)

// ValidateRequiredFields validates that all required fields are present
func ValidateRequiredFields(cfg *Config) error {
	if cfg.Server.Port == 0 {
		return fmt.Errorf("%w: server_port", ErrMissingRequiredField)
	}
	if cfg.Database.URI == "" {
		return fmt.Errorf("%w: mongodb_uri", ErrMissingRequiredField)
	}
	if cfg.Database.Database == "" {
		return fmt.Errorf("%w: mongodb_database", ErrMissingRequiredField)
	}
	if cfg.NATS.URL == "" {
		return fmt.Errorf("%w: nats_url", ErrMissingRequiredField)
	}
	if cfg.LLM.Endpoint == "" {
		return fmt.Errorf("%w: llm_endpoint", ErrMissingRequiredField)
	}
	if cfg.LLM.APIKey == "" {
		return fmt.Errorf("%w: llm_api_key", ErrMissingRequiredField)
	}
	return nil
}

// ValidatePortRange validates port number is in valid range
func ValidatePortRange(port int) error {
	if port < 1024 || port > 65535 {
		return fmt.Errorf("%w: port must be between 1024 and 65535, got %d", ErrInvalidPort, port)
	}
	return nil
}

// ValidateMongoDBURI validates MongoDB connection string format
func ValidateMongoDBURI(uri string) error {
	if uri == "" {
		return fmt.Errorf("%w: empty URI", ErrInvalidMongoDBURI)
	}

	if !strings.HasPrefix(uri, "mongodb://") && !strings.HasPrefix(uri, "mongodb+srv://") {
		return fmt.Errorf("%w: must start with mongodb:// or mongodb+srv://", ErrInvalidMongoDBURI)
	}

	// Basic validation - just check it's not just the protocol
	if uri == "mongodb://" || uri == "mongodb+srv://" {
		return fmt.Errorf("%w: incomplete URI", ErrInvalidMongoDBURI)
	}

	// Check for invalid characters in database name (if present)
	if strings.Contains(uri, "/ ") || strings.Contains(uri, " /") {
		return fmt.Errorf("%w: invalid characters in database name", ErrInvalidMongoDBURI)
	}

	return nil
}

// ValidateNATSURL validates NATS connection URL format
func ValidateNATSURL(natsURL string) error {
	if natsURL == "" {
		return fmt.Errorf("%w: empty URL", ErrInvalidNATSURL)
	}

	if !strings.HasPrefix(natsURL, "nats://") && !strings.HasPrefix(natsURL, "tls://") {
		return fmt.Errorf("%w: must start with nats:// or tls://", ErrInvalidNATSURL)
	}

	if natsURL == "nats://" || natsURL == "tls://" {
		return fmt.Errorf("%w: incomplete URL", ErrInvalidNATSURL)
	}

	return nil
}

// ValidateHTTPURL validates HTTP/HTTPS URL format
func ValidateHTTPURL(httpURL string) error {
	if httpURL == "" {
		return fmt.Errorf("%w: empty URL", ErrInvalidHTTPURL)
	}

	if !strings.HasPrefix(httpURL, "http://") && !strings.HasPrefix(httpURL, "https://") {
		return fmt.Errorf("%w: must start with http:// or https://", ErrInvalidHTTPURL)
	}

	parsedURL, err := url.Parse(httpURL)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidHTTPURL, err)
	}

	if parsedURL.Host == "" {
		return fmt.Errorf("%w: missing host", ErrInvalidHTTPURL)
	}

	return nil
}

// ValidateLogLevel validates logging level values
func ValidateLogLevel(level string) error {
	if level == "" {
		return fmt.Errorf("%w: empty log level", ErrInvalidLogLevel)
	}

	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
		"fatal": true,
	}

	if !validLevels[level] {
		return fmt.Errorf("%w: must be one of [debug, info, warn, error, fatal], got %s", ErrInvalidLogLevel, level)
	}

	return nil
}

// ValidateLogFormat validates logging format values
func ValidateLogFormat(format string) error {
	if format == "" {
		return fmt.Errorf("%w: empty log format", ErrInvalidLogFormat)
	}

	if format != "json" && format != "text" {
		return fmt.Errorf("%w: must be 'json' or 'text', got %s", ErrInvalidLogFormat, format)
	}

	return nil
}

// ValidatePoolSize validates database connection pool size
func ValidatePoolSize(minPoolSize, maxPoolSize int) error {
	if minPoolSize < 0 {
		return fmt.Errorf("%w: minPoolSize cannot be negative", ErrInvalidPoolSize)
	}
	if maxPoolSize <= 0 {
		return fmt.Errorf("%w: maxPoolSize must be positive", ErrInvalidPoolSize)
	}
	if minPoolSize > maxPoolSize {
		return fmt.Errorf("%w: minPoolSize (%d) cannot be greater than maxPoolSize (%d)", ErrInvalidPoolSize, minPoolSize, maxPoolSize)
	}
	if maxPoolSize > 1000 {
		return fmt.Errorf("%w: maxPoolSize (%d) exceeds maximum allowed (1000)", ErrInvalidPoolSize, maxPoolSize)
	}
	return nil
}

// ValidateSchedulerInterval validates scheduler interval format
func ValidateSchedulerInterval(interval string) error {
	if interval == "" {
		return fmt.Errorf("%w: empty interval", ErrInvalidSchedulerInterval)
	}

	duration, err := time.ParseDuration(interval)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidSchedulerInterval, err)
	}

	if duration < 60*time.Second {
		return fmt.Errorf("%w: interval too short (minimum 60s), got %s", ErrInvalidSchedulerInterval, interval)
	}

	return nil
}

// ValidateBatchSize validates scheduler batch size
func ValidateBatchSize(batchSize int) error {
	if batchSize <= 0 {
		return fmt.Errorf("%w: must be positive, got %d", ErrInvalidBatchSize, batchSize)
	}
	if batchSize > 1000 {
		return fmt.Errorf("%w: exceeds maximum (1000), got %d", ErrInvalidBatchSize, batchSize)
	}
	return nil
}

// ValidateLLMTemperature validates LLM temperature parameter
func ValidateLLMTemperature(temperature float64) error {
	if temperature < 0.0 || temperature > 2.0 {
		return fmt.Errorf("%w: must be between 0.0 and 2.0, got %.2f", ErrInvalidTemperature, temperature)
	}
	return nil
}

// ValidateCompleteConfig validates entire configuration object
func ValidateCompleteConfig(cfg *Config) error {
	// Required fields
	if err := ValidateRequiredFields(cfg); err != nil {
		return err
	}

	// Port validation
	if err := ValidatePortRange(cfg.Server.Port); err != nil {
		return err
	}

	// MongoDB URI validation
	if err := ValidateMongoDBURI(cfg.Database.URI); err != nil {
		return err
	}

	// Pool size validation
	if err := ValidatePoolSize(cfg.Database.MinPoolSize, cfg.Database.MaxPoolSize); err != nil {
		return err
	}

	// NATS URL validation
	if err := ValidateNATSURL(cfg.NATS.URL); err != nil {
		return err
	}

	// LLM endpoint validation
	if err := ValidateHTTPURL(cfg.LLM.Endpoint); err != nil {
		return err
	}

	// LLM temperature validation
	if err := ValidateLLMTemperature(cfg.LLM.Temperature); err != nil {
		return err
	}

	// Log level validation
	if err := ValidateLogLevel(cfg.Logging.Level); err != nil {
		return err
	}

	// Log format validation
	if err := ValidateLogFormat(cfg.Logging.Format); err != nil {
		return err
	}

	// Scheduler interval validation (if enabled)
	if cfg.Scheduler.Enabled && cfg.Scheduler.Interval > 0 {
		intervalStr := cfg.Scheduler.Interval.String()
		if err := ValidateSchedulerInterval(intervalStr); err != nil {
			return err
		}
	}

	// Batch size validation (if scheduler enabled)
	if cfg.Scheduler.Enabled {
		if err := ValidateBatchSize(cfg.Scheduler.BatchSize); err != nil {
			return err
		}
	}

	return nil
}
