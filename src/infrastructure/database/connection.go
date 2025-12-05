package database

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.uber.org/zap"
)

// ConnectionConfig holds MongoDB connection configuration
type ConnectionConfig struct {
	URI                string
	Database           string
	MinPoolSize        uint64
	MaxPoolSize        uint64
	MaxConnIdleTime    time.Duration
	ConnectTimeout     time.Duration
	ServerSelectTimeout time.Duration
	MaxRetries         int
	InitialBackoff     time.Duration
	MaxBackoff         time.Duration
}

// DefaultConnectionConfig returns default connection configuration
func DefaultConnectionConfig() *ConnectionConfig {
	return &ConnectionConfig{
		MinPoolSize:        5,
		MaxPoolSize:        100,
		MaxConnIdleTime:    30 * time.Second,
		ConnectTimeout:     10 * time.Second,
		ServerSelectTimeout: 5 * time.Second,
		MaxRetries:         3,
		InitialBackoff:     100 * time.Millisecond,
		MaxBackoff:         5 * time.Second,
	}
}

// Connection manages MongoDB connection with retry logic
type Connection struct {
	client *mongo.Client
	config *ConnectionConfig
	logger *zap.Logger
}

// NewConnection creates a new MongoDB connection manager
func NewConnection(config *ConnectionConfig, logger *zap.Logger) *Connection {
	if config == nil {
		config = DefaultConnectionConfig()
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Connection{
		config: config,
		logger: logger,
	}
}

// Connect establishes connection to MongoDB with retry logic
func (c *Connection) Connect(ctx context.Context) error {
	var lastErr error
	backoff := c.config.InitialBackoff

	for attempt := 1; attempt <= c.config.MaxRetries; attempt++ {
		c.logger.Info("Attempting to connect to MongoDB",
			zap.Int("attempt", attempt),
			zap.Int("max_retries", c.config.MaxRetries),
		)

		client, err := c.attemptConnection(ctx)
		if err == nil {
			c.client = client
			c.logger.Info("Successfully connected to MongoDB",
				zap.String("database", c.config.Database),
			)
			return nil
		}

		lastErr = err
		c.logger.Warn("Failed to connect to MongoDB",
			zap.Int("attempt", attempt),
			zap.Error(err),
			zap.Duration("retry_after", backoff),
		)

		if attempt < c.config.MaxRetries {
			select {
			case <-ctx.Done():
				return fmt.Errorf("connection cancelled: %w", ctx.Err())
			case <-time.After(backoff):
				// Exponential backoff with max limit
				backoff *= 2
				if backoff > c.config.MaxBackoff {
					backoff = c.config.MaxBackoff
				}
			}
		}
	}

	return fmt.Errorf("failed to connect after %d attempts: %w", c.config.MaxRetries, lastErr)
}

// attemptConnection tries to establish a single connection
func (c *Connection) attemptConnection(ctx context.Context) (*mongo.Client, error) {
	clientOptions := options.Client().
		ApplyURI(c.config.URI).
		SetMinPoolSize(c.config.MinPoolSize).
		SetMaxPoolSize(c.config.MaxPoolSize).
		SetMaxConnIdleTime(c.config.MaxConnIdleTime).
		SetConnectTimeout(c.config.ConnectTimeout).
		SetServerSelectionTimeout(c.config.ServerSelectTimeout)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to create MongoDB client: %w", err)
	}

	// Ping to verify connection
	pingCtx, cancel := context.WithTimeout(ctx, c.config.ConnectTimeout)
	defer cancel()

	if err := client.Ping(pingCtx, readpref.Primary()); err != nil {
		_ = client.Disconnect(ctx)
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	return client, nil
}

// Disconnect gracefully disconnects from MongoDB
func (c *Connection) Disconnect(ctx context.Context) error {
	if c.client == nil {
		return nil
	}

	c.logger.Info("Disconnecting from MongoDB")

	if err := c.client.Disconnect(ctx); err != nil {
		c.logger.Error("Failed to disconnect from MongoDB", zap.Error(err))
		return fmt.Errorf("failed to disconnect: %w", err)
	}

	c.logger.Info("Successfully disconnected from MongoDB")
	c.client = nil
	return nil
}

// GetClient returns the MongoDB client
func (c *Connection) GetClient() *mongo.Client {
	return c.client
}

// IsConnected checks if the connection is established
func (c *Connection) IsConnected(ctx context.Context) bool {
	if c.client == nil {
		return false
	}

	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	err := c.client.Ping(pingCtx, readpref.Primary())
	return err == nil
}

// Reconnect attempts to reconnect to MongoDB
func (c *Connection) Reconnect(ctx context.Context) error {
	c.logger.Info("Attempting to reconnect to MongoDB")

	// Disconnect existing connection if any
	if c.client != nil {
		if err := c.Disconnect(ctx); err != nil {
			c.logger.Warn("Error during disconnect before reconnect", zap.Error(err))
		}
	}

	// Establish new connection
	return c.Connect(ctx)
}
