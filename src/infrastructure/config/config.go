package config

import (
	"fmt"
	"strings"
	"time"
)

// Config represents the complete application configuration
type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	NATS      NATSConfig
	LLM       LLMConfig
	LinkedIn  LinkedInAPIConfig
	Scheduler SchedulerConfig
	Logging   LoggingConfig
	version   int
}

// ServerConfig contains HTTP server configuration
type ServerConfig struct {
	Host            string
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

// DatabaseConfig contains MongoDB configuration
type DatabaseConfig struct {
	URI            string
	Database       string
	MaxPoolSize    int
	MinPoolSize    int
	ConnectTimeout time.Duration
}

// NATSConfig contains NATS messaging configuration
type NATSConfig struct {
	URL           string
	Queue         string
	MaxReconnects int
	ReconnectWait time.Duration
}

// LLMConfig contains LLM service configuration
type LLMConfig struct {
	Endpoint    string
	Model       string
	APIKey      string
	Timeout     time.Duration
	MaxTokens   int
	Temperature float64
}

// LinkedInAPIConfig contains LinkedIn API configuration
type LinkedInAPIConfig struct {
	APIURL          string
	ClientID        string
	ClientSecret    string
	RedirectURI     string
	Timeout         time.Duration
	RateLimit       int
	RateLimitWindow time.Duration
}

// SchedulerConfig contains scheduler configuration
type SchedulerConfig struct {
	Interval   time.Duration
	BatchSize  int
	MaxRetries int
	RetryDelay time.Duration
	Enabled    bool
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level      string
	Format     string
	Output     string
	FileOutput string
}

// String returns a string representation of the config with masked secrets
func (c *Config) String() string {
	return fmt.Sprintf(`Configuration:
  Server:
    Host: %s
    Port: %d
    ReadTimeout: %s
    WriteTimeout: %s
    ShutdownTimeout: %s
  Database:
    URI: %s
    Database: %s
    MaxPoolSize: %d
    MinPoolSize: %d
    ConnectTimeout: %s
  NATS:
    URL: %s
    Queue: %s
    MaxReconnects: %d
    ReconnectWait: %s
  LLM:
    Endpoint: %s
    Model: %s
    APIKey: %s
    Timeout: %s
    MaxTokens: %d
    Temperature: %.2f
  LinkedIn:
    APIURL: %s
    ClientID: %s
    ClientSecret: %s
    RedirectURI: %s
    Timeout: %s
    RateLimit: %d
    RateLimitWindow: %s
  Scheduler:
    Interval: %s
    BatchSize: %d
    MaxRetries: %d
    RetryDelay: %s
    Enabled: %t
  Logging:
    Level: %s
    Format: %s
    Output: %s
    FileOutput: %s
  Version: %d`,
		c.Server.Host,
		c.Server.Port,
		c.Server.ReadTimeout,
		c.Server.WriteTimeout,
		c.Server.ShutdownTimeout,
		maskSecret(c.Database.URI),
		c.Database.Database,
		c.Database.MaxPoolSize,
		c.Database.MinPoolSize,
		c.Database.ConnectTimeout,
		c.NATS.URL,
		c.NATS.Queue,
		c.NATS.MaxReconnects,
		c.NATS.ReconnectWait,
		c.LLM.Endpoint,
		c.LLM.Model,
		maskSecret(c.LLM.APIKey),
		c.LLM.Timeout,
		c.LLM.MaxTokens,
		c.LLM.Temperature,
		c.LinkedIn.APIURL,
		c.LinkedIn.ClientID,
		maskSecret(c.LinkedIn.ClientSecret),
		c.LinkedIn.RedirectURI,
		c.LinkedIn.Timeout,
		c.LinkedIn.RateLimit,
		c.LinkedIn.RateLimitWindow,
		c.Scheduler.Interval,
		c.Scheduler.BatchSize,
		c.Scheduler.MaxRetries,
		c.Scheduler.RetryDelay,
		c.Scheduler.Enabled,
		c.Logging.Level,
		c.Logging.Format,
		c.Logging.Output,
		c.Logging.FileOutput,
		c.version,
	)
}

// maskSecret masks sensitive information for display
func maskSecret(secret string) string {
	if secret == "" {
		return ""
	}

	// Mask passwords in URIs
	if strings.Contains(secret, "://") && strings.Contains(secret, ":") && strings.Contains(secret, "@") {
		parts := strings.Split(secret, "@")
		if len(parts) >= 2 {
			authPart := parts[0]
			hostPart := strings.Join(parts[1:], "@")

			authSubparts := strings.Split(authPart, ":")
			if len(authSubparts) >= 2 {
				protocol := authSubparts[0]
				username := authSubparts[1]
				if strings.HasPrefix(username, "//") {
					username = username[2:]
				}
				return fmt.Sprintf("%s://%s:***@%s", protocol, username, hostPart)
			}
		}
	}

	// Mask plain secrets
	return "***"
}

// GetVersion returns the current configuration version
func (c *Config) GetVersion() int {
	return c.version
}

// IncrementVersion increments the configuration version
func (c *Config) incrementVersion() {
	c.version++
}
