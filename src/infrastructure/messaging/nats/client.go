package nats

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

var (
	// ErrInvalidURL indicates an invalid NATS URL
	ErrInvalidURL = errors.New("invalid NATS URL")
	// ErrAlreadyConnected indicates client is already connected
	ErrAlreadyConnected = errors.New("client already connected")
	// ErrNotConnected indicates client is not connected
	ErrNotConnected = errors.New("client not connected")
	// ErrConnectionTimeout indicates connection timeout
	ErrConnectionTimeout = errors.New("connection timeout")
	// ErrInvalidTimeout indicates invalid timeout value
	ErrInvalidTimeout = errors.New("invalid timeout value")
)

// NATSClient represents a NATS client connection wrapper
type NATSClient struct {
	url            string
	conn           *nats.Conn
	logger         *zap.Logger
	mu             sync.RWMutex
	maxReconnects  int
	reconnectDelay time.Duration
	drainTimeout   time.Duration
	connected      bool
	// Metrics counters
	connectionsTotal    int64
	disconnectionsTotal int64
	reconnectionsTotal  int64
}

// ClientConfig holds NATS client configuration
type ClientConfig struct {
	URL            string
	MaxReconnects  int
	ReconnectDelay time.Duration
	DrainTimeout   time.Duration
	Logger         *zap.Logger
}

// NewNATSClient creates a new NATS client instance
func NewNATSClient(config ClientConfig) (*NATSClient, error) {
	if config.URL == "" {
		return nil, ErrInvalidURL
	}

	// Validate URL format
	if !isValidNATSURL(config.URL) {
		return nil, ErrInvalidURL
	}

	// Set default logger if not provided
	logger := config.Logger
	if logger == nil {
		logger, _ = zap.NewProduction()
	}

	// Set default values
	if config.MaxReconnects == 0 {
		config.MaxReconnects = 3
	}
	if config.ReconnectDelay == 0 {
		config.ReconnectDelay = 2 * time.Second
	}
	if config.DrainTimeout == 0 {
		config.DrainTimeout = 5 * time.Second
	}

	return &NATSClient{
		url:            config.URL,
		logger:         logger,
		maxReconnects:  config.MaxReconnects,
		reconnectDelay: config.ReconnectDelay,
		drainTimeout:   config.DrainTimeout,
		connected:      false,
	}, nil
}

// Connect establishes connection to NATS server
func (c *NATSClient) Connect(ctx context.Context, timeout time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return ErrAlreadyConnected
	}

	if timeout <= 0 {
		return ErrInvalidTimeout
	}

	// Create connection options
	opts := []nats.Option{
		nats.Name("linkgen-nats-client"),
		nats.MaxReconnects(c.maxReconnects),
		nats.ReconnectWait(c.reconnectDelay),
		nats.DisconnectErrHandler(c.disconnectHandler),
		nats.ReconnectHandler(c.reconnectHandler),
		nats.ErrorHandler(c.errorHandler),
	}

	// Connect with timeout
	connectDone := make(chan error, 1)
	go func() {
		conn, err := nats.Connect(c.url, opts...)
		if err != nil {
			connectDone <- err
			return
		}
		c.conn = conn
		connectDone <- nil
	}()

	select {
	case err := <-connectDone:
		if err != nil {
			c.logger.Error("failed to connect to NATS", zap.Error(err))
			return fmt.Errorf("connect failed: %w", err)
		}
		c.connected = true
		c.connectionsTotal++
		c.logger.Info("connected to NATS", zap.String("url", c.url))
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(timeout):
		return ErrConnectionTimeout
	}
}

// Disconnect closes the NATS connection gracefully
func (c *NATSClient) Disconnect(timeout time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected || c.conn == nil {
		return nil // Already disconnected
	}

	var err error
	if timeout > 0 {
		// Drain connection (graceful shutdown)
		err = c.conn.Drain()
		if err != nil {
			c.logger.Warn("drain failed, forcing close", zap.Error(err))
			c.conn.Close()
		}

		// Wait for drain with timeout
		drainComplete := make(chan struct{})
		go func() {
			// Wait for drain to complete
			time.Sleep(timeout)
			close(drainComplete)
		}()
		<-drainComplete
	} else {
		c.conn.Close()
	}

	c.connected = false
	c.disconnectionsTotal++
	c.logger.Info("disconnected from NATS")

	return err
}

// IsConnected returns the connection status
func (c *NATSClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected && c.conn != nil && c.conn.IsConnected()
}

// GetConnection returns the underlying NATS connection
func (c *NATSClient) GetConnection() *nats.Conn {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conn
}

// GetMetrics returns client metrics
func (c *NATSClient) GetMetrics() map[string]int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]int64{
		"connections_total":    c.connectionsTotal,
		"disconnections_total": c.disconnectionsTotal,
		"reconnections_total":  c.reconnectionsTotal,
	}
}

// disconnectHandler is called when connection is lost
func (c *NATSClient) disconnectHandler(nc *nats.Conn, err error) {
	if err != nil {
		c.logger.Warn("disconnected from NATS", zap.Error(err))
	}
}

// reconnectHandler is called when connection is re-established
func (c *NATSClient) reconnectHandler(nc *nats.Conn) {
	c.mu.Lock()
	c.reconnectionsTotal++
	c.mu.Unlock()
	c.logger.Info("reconnected to NATS", zap.String("url", nc.ConnectedUrl()))
}

// errorHandler handles async errors
func (c *NATSClient) errorHandler(nc *nats.Conn, sub *nats.Subscription, err error) {
	c.logger.Error("NATS error", zap.Error(err))
}

// isValidNATSURL validates NATS URL format
func isValidNATSURL(url string) bool {
	if url == "" {
		return false
	}

	// Check for valid NATS protocol
	if len(url) < 7 {
		return false
	}

	prefix := url[:7]
	return prefix == "nats://" || url[:6] == "tls://"
}
