package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

var (
	// ErrConfigFileNotFound indicates configuration file was not found
	ErrConfigFileNotFound = errors.New("configuration file not found")
	// ErrInvalidYAML indicates invalid YAML format
	ErrInvalidYAML = errors.New("invalid YAML format")
)

var (
	globalConfig *Config
	configMutex  sync.RWMutex
)

// LoadFromEnvironment loads configuration from environment variables
func LoadFromEnvironment() (*Config, error) {
	cfg := NewDefaultConfig()

	// Server configuration
	if port := os.Getenv("LINKGEN_SERVER_PORT"); port != "" {
		p, err := strconv.Atoi(port)
		if err != nil {
			return nil, fmt.Errorf("invalid LINKGEN_SERVER_PORT: %w", err)
		}
		cfg.Server.Port = p
	}

	if host := os.Getenv("LINKGEN_SERVER_HOST"); host != "" {
		cfg.Server.Host = host
	}

	if readTimeout := os.Getenv("LINKGEN_SERVER_READ_TIMEOUT"); readTimeout != "" {
		d, err := time.ParseDuration(readTimeout + "s")
		if err == nil {
			cfg.Server.ReadTimeout = d
		}
	}

	if writeTimeout := os.Getenv("LINKGEN_SERVER_WRITE_TIMEOUT"); writeTimeout != "" {
		d, err := time.ParseDuration(writeTimeout + "s")
		if err == nil {
			cfg.Server.WriteTimeout = d
		}
	}

	// Database configuration
	if uri := os.Getenv("LINKGEN_MONGODB_URI"); uri != "" {
		cfg.Database.URI = uri
	}

	if database := os.Getenv("LINKGEN_MONGODB_DATABASE"); database != "" {
		cfg.Database.Database = database
	}

	if maxPoolSize := os.Getenv("LINKGEN_MONGODB_MAX_POOL_SIZE"); maxPoolSize != "" {
		size, err := strconv.Atoi(maxPoolSize)
		if err == nil {
			cfg.Database.MaxPoolSize = size
		}
	}

	if minPoolSize := os.Getenv("LINKGEN_MONGODB_MIN_POOL_SIZE"); minPoolSize != "" {
		size, err := strconv.Atoi(minPoolSize)
		if err == nil {
			cfg.Database.MinPoolSize = size
		}
	}

	// NATS configuration
	if natsURL := os.Getenv("LINKGEN_NATS_URL"); natsURL != "" {
		cfg.NATS.URL = natsURL
	}

	if queue := os.Getenv("LINKGEN_NATS_QUEUE"); queue != "" {
		cfg.NATS.Queue = queue
	}

	// LLM configuration
	if endpoint := os.Getenv("LINKGEN_LLM_ENDPOINT"); endpoint != "" {
		cfg.LLM.Endpoint = endpoint
	}

	if model := os.Getenv("LINKGEN_LLM_MODEL"); model != "" {
		cfg.LLM.Model = model
	}

	if apiKey := os.Getenv("LINKGEN_LLM_API_KEY"); apiKey != "" {
		cfg.LLM.APIKey = apiKey
	}

	if timeout := os.Getenv("LINKGEN_LLM_TIMEOUT"); timeout != "" {
		t, err := strconv.Atoi(timeout)
		if err == nil {
			cfg.LLM.Timeout = time.Duration(t) * time.Second
		}
	}

	if maxTokens := os.Getenv("LINKGEN_LLM_MAX_TOKENS"); maxTokens != "" {
		tokens, err := strconv.Atoi(maxTokens)
		if err == nil {
			cfg.LLM.MaxTokens = tokens
		}
	}

	if temperature := os.Getenv("LINKGEN_LLM_TEMPERATURE"); temperature != "" {
		temp, err := strconv.ParseFloat(temperature, 64)
		if err == nil {
			cfg.LLM.Temperature = temp
		}
	}

	// LinkedIn configuration
	if apiURL := os.Getenv("LINKGEN_LINKEDIN_API_URL"); apiURL != "" {
		cfg.LinkedIn.APIURL = apiURL
	}

	if clientID := os.Getenv("LINKGEN_LINKEDIN_CLIENT_ID"); clientID != "" {
		cfg.LinkedIn.ClientID = clientID
	}

	if clientSecret := os.Getenv("LINKGEN_LINKEDIN_CLIENT_SECRET"); clientSecret != "" {
		cfg.LinkedIn.ClientSecret = clientSecret
	}

	// Scheduler configuration
	if interval := os.Getenv("LINKGEN_SCHEDULER_INTERVAL"); interval != "" {
		d, err := time.ParseDuration(interval)
		if err == nil {
			cfg.Scheduler.Interval = d
		}
	}

	if batchSize := os.Getenv("LINKGEN_SCHEDULER_BATCH_SIZE"); batchSize != "" {
		size, err := strconv.Atoi(batchSize)
		if err == nil {
			cfg.Scheduler.BatchSize = size
		}
	}

	// Logging configuration
	if logLevel := os.Getenv("LINKGEN_LOG_LEVEL"); logLevel != "" {
		cfg.Logging.Level = logLevel
	}

	if logFormat := os.Getenv("LINKGEN_LOG_FORMAT"); logFormat != "" {
		cfg.Logging.Format = logFormat
	}

	// Validate configuration
	if err := ValidateRequiredFields(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// LoadFromFile loads configuration from a YAML file
func LoadFromFile(filePath string) (*Config, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrConfigFileNotFound, filePath)
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var rawConfig map[string]interface{}
	if err := yaml.Unmarshal(data, &rawConfig); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidYAML, err)
	}

	cfg := NewDefaultConfig()

	// Parse server configuration
	if server, ok := rawConfig["server"].(map[string]interface{}); ok {
		if host, ok := server["host"].(string); ok {
			cfg.Server.Host = host
		}
		if port, ok := server["port"].(int); ok {
			cfg.Server.Port = port
		}
		if readTimeout, ok := server["read_timeout"].(int); ok {
			cfg.Server.ReadTimeout = time.Duration(readTimeout) * time.Second
		}
		if writeTimeout, ok := server["write_timeout"].(int); ok {
			cfg.Server.WriteTimeout = time.Duration(writeTimeout) * time.Second
		}
	}

	// Parse database configuration
	if database, ok := rawConfig["database"].(map[string]interface{}); ok {
		if uri, ok := database["uri"].(string); ok {
			cfg.Database.URI = uri
		}
		if db, ok := database["database"].(string); ok {
			cfg.Database.Database = db
		}
		if maxPoolSize, ok := database["max_pool_size"].(int); ok {
			cfg.Database.MaxPoolSize = maxPoolSize
		}
		if minPoolSize, ok := database["min_pool_size"].(int); ok {
			cfg.Database.MinPoolSize = minPoolSize
		}
	}

	// Parse NATS configuration
	if nats, ok := rawConfig["nats"].(map[string]interface{}); ok {
		if url, ok := nats["url"].(string); ok {
			cfg.NATS.URL = url
		}
		if queue, ok := nats["queue"].(string); ok {
			cfg.NATS.Queue = queue
		}
	}

	// Parse LLM configuration
	if llm, ok := rawConfig["llm"].(map[string]interface{}); ok {
		if endpoint, ok := llm["endpoint"].(string); ok {
			cfg.LLM.Endpoint = endpoint
		}
		if model, ok := llm["model"].(string); ok {
			cfg.LLM.Model = model
		}
		if apiKey, ok := llm["api_key"].(string); ok {
			cfg.LLM.APIKey = apiKey
		}
		if timeout, ok := llm["timeout"].(int); ok {
			cfg.LLM.Timeout = time.Duration(timeout) * time.Second
		}
		if maxTokens, ok := llm["max_tokens"].(int); ok {
			cfg.LLM.MaxTokens = maxTokens
		}
		if temperature, ok := llm["temperature"].(float64); ok {
			cfg.LLM.Temperature = temperature
		}
	}

	// Parse LinkedIn configuration
	if linkedin, ok := rawConfig["linkedin"].(map[string]interface{}); ok {
		if apiURL, ok := linkedin["api_url"].(string); ok {
			cfg.LinkedIn.APIURL = apiURL
		}
		if clientID, ok := linkedin["client_id"].(string); ok {
			cfg.LinkedIn.ClientID = clientID
		}
		if clientSecret, ok := linkedin["client_secret"].(string); ok {
			cfg.LinkedIn.ClientSecret = clientSecret
		}
	}

	// Parse scheduler configuration
	if scheduler, ok := rawConfig["scheduler"].(map[string]interface{}); ok {
		if interval, ok := scheduler["interval"].(string); ok {
			d, err := time.ParseDuration(interval)
			if err == nil {
				cfg.Scheduler.Interval = d
			}
		}
		if batchSize, ok := scheduler["batch_size"].(int); ok {
			cfg.Scheduler.BatchSize = batchSize
		}
		if enabled, ok := scheduler["enabled"].(bool); ok {
			cfg.Scheduler.Enabled = enabled
		}
	}

	// Parse logging configuration
	if logging, ok := rawConfig["logging"].(map[string]interface{}); ok {
		if level, ok := logging["level"].(string); ok {
			cfg.Logging.Level = level
		}
		if format, ok := logging["format"].(string); ok {
			cfg.Logging.Format = format
		}
		if output, ok := logging["output"].(string); ok {
			cfg.Logging.Output = output
		}
	}

	return cfg, nil
}

// LoadFromFlags loads configuration from command-line flags
func LoadFromFlags() (*Config, error) {
	cfg := NewDefaultConfig()

	serverPort := flag.Int("server-port", cfg.Server.Port, "Server port")
	serverHost := flag.String("server-host", cfg.Server.Host, "Server host")
	mongodbURI := flag.String("mongodb-uri", cfg.Database.URI, "MongoDB URI")
	mongodbDatabase := flag.String("mongodb-database", cfg.Database.Database, "MongoDB database")
	natsURL := flag.String("nats-url", cfg.NATS.URL, "NATS URL")
	llmEndpoint := flag.String("llm-endpoint", cfg.LLM.Endpoint, "LLM endpoint")
	llmAPIKey := flag.String("llm-api-key", cfg.LLM.APIKey, "LLM API key")
	logLevel := flag.String("log-level", cfg.Logging.Level, "Log level")
	logFormat := flag.String("log-format", cfg.Logging.Format, "Log format")

	flag.Parse()

	cfg.Server.Port = *serverPort
	cfg.Server.Host = *serverHost
	cfg.Database.URI = *mongodbURI
	cfg.Database.Database = *mongodbDatabase
	cfg.NATS.URL = *natsURL
	cfg.LLM.Endpoint = *llmEndpoint
	cfg.LLM.APIKey = *llmAPIKey
	cfg.Logging.Level = *logLevel
	cfg.Logging.Format = *logFormat

	// Validate flags
	if cfg.Server.Port != 0 {
		if err := ValidatePortRange(cfg.Server.Port); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

// LoadWithPrecedence loads configuration with precedence: flags > env > file > defaults
func LoadWithPrecedence(filePath string) (*Config, error) {
	// Start with defaults
	cfg := NewDefaultConfig()

	// Load from file if provided
	if filePath != "" {
		fileCfg, err := LoadFromFile(filePath)
		if err != nil && !errors.Is(err, ErrConfigFileNotFound) {
			return nil, err
		}
		if err == nil {
			cfg = fileCfg
		}
	}

	// Override with environment variables
	envCfg, err := LoadFromEnvironment()
	if err == nil {
		mergeConfigs(cfg, envCfg)
	}

	// Override with flags (if parsed)
	if flag.Parsed() {
		flagCfg, err := LoadFromFlags()
		if err == nil {
			mergeConfigs(cfg, flagCfg)
		}
	}

	// Validate final configuration
	if err := ValidateCompleteConfig(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// AutoDetectConfigFile automatically detects configuration file based on environment
func AutoDetectConfigFile() (string, error) {
	environment := os.Getenv("LINKGEN_ENV")
	if environment == "" {
		environment = "development"
	}

	filename := fmt.Sprintf("%s.yaml", environment)
	searchPaths := []string{
		"./configs",
		"../configs",
		"../../configs",
		"/etc/linkgen",
	}

	for _, path := range searchPaths {
		fullPath := filepath.Join(path, filename)
		if _, err := os.Stat(fullPath); err == nil {
			return fullPath, nil
		}
	}

	return "", fmt.Errorf("%w: searched for %s in %v", ErrConfigFileNotFound, filename, searchPaths)
}

// NewDefaultConfig creates a new configuration with default values
func NewDefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:            "0.0.0.0",
			Port:            8000,
			ReadTimeout:     30 * time.Second,
			WriteTimeout:    30 * time.Second,
			ShutdownTimeout: 10 * time.Second,
		},
		Database: DatabaseConfig{
			Database:       "linkgenai",
			MaxPoolSize:    100,
			MinPoolSize:    10,
			ConnectTimeout: 10 * time.Second,
		},
		NATS: NATSConfig{
			Queue:         "linkgen-queue",
			MaxReconnects: 10,
			ReconnectWait: 2 * time.Second,
		},
		LLM: LLMConfig{
			Timeout:     60 * time.Second,
			MaxTokens:   2000,
			Temperature: 0.7,
		},
		LinkedIn: LinkedInAPIConfig{
			APIURL:          "https://api.linkedin.com/v2",
			Timeout:         30 * time.Second,
			RateLimit:       100,
			RateLimitWindow: 1 * time.Minute,
		},
		Scheduler: SchedulerConfig{
			Interval:   6 * time.Hour,
			BatchSize:  100,
			MaxRetries: 3,
			RetryDelay: 5 * time.Minute,
			Enabled:    true,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		},
		version: 1,
	}
}

// ReloadConfig reloads the configuration
func ReloadConfig(filePath string) error {
	newCfg, err := LoadWithPrecedence(filePath)
	if err != nil {
		return err
	}

	configMutex.Lock()
	defer configMutex.Unlock()

	if globalConfig != nil {
		newCfg.version = globalConfig.version + 1
	}

	globalConfig = newCfg
	return nil
}

// GetConfig returns the current global configuration (thread-safe)
func GetConfig() *Config {
	configMutex.RLock()
	defer configMutex.RUnlock()

	if globalConfig == nil {
		return NewDefaultConfig()
	}

	return globalConfig
}

// SetConfig sets the global configuration (thread-safe)
func SetConfig(cfg *Config) {
	configMutex.Lock()
	defer configMutex.Unlock()

	globalConfig = cfg
}

// mergeConfigs merges source config into destination config (non-zero values only)
func mergeConfigs(dst, src *Config) {
	// Server
	if src.Server.Host != "" && src.Server.Host != "0.0.0.0" {
		dst.Server.Host = src.Server.Host
	}
	if src.Server.Port != 0 && src.Server.Port != 8000 {
		dst.Server.Port = src.Server.Port
	}

	// Database
	if src.Database.URI != "" {
		dst.Database.URI = src.Database.URI
	}
	if src.Database.Database != "" && src.Database.Database != "linkgenai" {
		dst.Database.Database = src.Database.Database
	}

	// NATS
	if src.NATS.URL != "" {
		dst.NATS.URL = src.NATS.URL
	}

	// LLM
	if src.LLM.Endpoint != "" {
		dst.LLM.Endpoint = src.LLM.Endpoint
	}
	if src.LLM.Model != "" {
		dst.LLM.Model = src.LLM.Model
	}
	if src.LLM.APIKey != "" {
		dst.LLM.APIKey = src.LLM.APIKey
	}

	// Scheduler
	if src.Scheduler.Interval != 6*time.Hour && src.Scheduler.Interval != 0 {
		dst.Scheduler.Interval = src.Scheduler.Interval
	}
	if src.Scheduler.BatchSize != 100 && src.Scheduler.BatchSize != 0 {
		dst.Scheduler.BatchSize = src.Scheduler.BatchSize
	}

	// Logging
	if src.Logging.Level != "" && src.Logging.Level != "info" {
		dst.Logging.Level = src.Logging.Level
	}
	if src.Logging.Format != "" && src.Logging.Format != "json" {
		dst.Logging.Format = src.Logging.Format
	}
}
