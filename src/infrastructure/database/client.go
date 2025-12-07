package database

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

var (
	// ErrClientNotInitialized is returned when client is accessed before initialization
	ErrClientNotInitialized = errors.New("database client not initialized")
	// ErrDatabaseNameEmpty is returned when database name is empty
	ErrDatabaseNameEmpty = errors.New("database name cannot be empty")
	// ErrCollectionNameEmpty is returned when collection name is empty
	ErrCollectionNameEmpty = errors.New("collection name cannot be empty")
)

// Client is a singleton wrapper around MongoDB connection
type Client struct {
	connection   *Connection
	databaseName string
	mu           sync.RWMutex
}

var (
	instance *Client
	once     sync.Once
)

// GetClient returns the singleton database client instance
// It initializes the client on first call with the provided configuration
func GetClient(config *ConnectionConfig, logger *zap.Logger) *Client {
	once.Do(func() {
		instance = &Client{
			connection:   NewConnection(config, logger),
			databaseName: config.Database,
		}
	})
	return instance
}

// ResetClient resets the singleton instance (mainly for testing)
func ResetClient() {
	instance = nil
	once = sync.Once{}
}

// Connect establishes connection to MongoDB
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.connection.Connect(ctx)
}

// Disconnect gracefully closes the MongoDB connection
func (c *Client) Disconnect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.connection.Disconnect(ctx)
}

// GetDatabase returns a handle to the configured database
func (c *Client) GetDatabase(name string) (*mongo.Database, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.connection.GetClient() == nil {
		return nil, ErrClientNotInitialized
	}

	// Use provided name or default to configured database
	dbName := name
	if dbName == "" {
		dbName = c.databaseName
	}

	if dbName == "" {
		return nil, ErrDatabaseNameEmpty
	}

	return c.connection.GetClient().Database(dbName), nil
}

// GetDefaultDatabase returns a handle to the default configured database
func (c *Client) GetDefaultDatabase() (*mongo.Database, error) {
	return c.GetDatabase("")
}

// GetCollection returns a handle to a specific collection in the default database
func (c *Client) GetCollection(collectionName string) (*mongo.Collection, error) {
	if collectionName == "" {
		return nil, ErrCollectionNameEmpty
	}

	db, err := c.GetDefaultDatabase()
	if err != nil {
		return nil, err
	}

	return db.Collection(collectionName), nil
}

// GetCollectionInDatabase returns a handle to a specific collection in a named database
func (c *Client) GetCollectionInDatabase(databaseName, collectionName string) (*mongo.Collection, error) {
	if collectionName == "" {
		return nil, ErrCollectionNameEmpty
	}

	db, err := c.GetDatabase(databaseName)
	if err != nil {
		return nil, err
	}

	return db.Collection(collectionName), nil
}

// IsConnected checks if the client is connected to MongoDB
func (c *Client) IsConnected(ctx context.Context) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.connection.IsConnected(ctx)
}

// Reconnect attempts to reconnect to MongoDB
func (c *Client) Reconnect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.connection.Reconnect(ctx)
}

// WithTransaction executes a function within a MongoDB transaction
func (c *Client) WithTransaction(ctx context.Context, fn func(sessCtx mongo.SessionContext) error) error {
	c.mu.RLock()
	client := c.connection.GetClient()
	c.mu.RUnlock()

	if client == nil {
		return ErrClientNotInitialized
	}

	session, err := client.StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		return nil, fn(sessCtx)
	})

	return err
}

// GetMongoClient returns the underlying mongo.Client (use with caution)
func (c *Client) GetMongoClient() (*mongo.Client, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	client := c.connection.GetClient()
	if client == nil {
		return nil, ErrClientNotInitialized
	}

	return client, nil
}

// Check performs a health check on the database connection
func (c *Client) Check(ctx context.Context) *HealthCheckResult {
	start := time.Now()

	c.mu.RLock()
	client := c.connection.GetClient()
	c.mu.RUnlock()

	if client == nil {
		return &HealthCheckResult{
			Status:    HealthStatusUnhealthy,
			Latency:   time.Since(start),
			Error:     "client not initialized",
			Timestamp: time.Now(),
		}
	}

	// Ping the database
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := client.Ping(pingCtx, nil); err != nil {
		return &HealthCheckResult{
			Status:    HealthStatusUnhealthy,
			Latency:   time.Since(start),
			Error:     err.Error(),
			Timestamp: time.Now(),
		}
	}

	return &HealthCheckResult{
		Status:    HealthStatusHealthy,
		Latency:   time.Since(start),
		Timestamp: time.Now(),
	}
}
