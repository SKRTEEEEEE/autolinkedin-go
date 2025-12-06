package nats

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

var (
	// ErrNilHandler indicates nil message handler
	ErrNilHandler = errors.New("message handler cannot be nil")
	// ErrAlreadySubscribed indicates already subscribed
	ErrAlreadySubscribed = errors.New("already subscribed")
	// ErrNotSubscribed indicates not subscribed
	ErrNotSubscribed = errors.New("not subscribed")
)

// MessageHandler is a function that processes consumed messages
type MessageHandler func(ctx context.Context, msg []byte) error

// Consumer handles message consumption from NATS
type Consumer struct {
	client        *NATSClient
	subject       string
	queueGroup    string
	handler       MessageHandler
	subscription  *nats.Subscription
	logger        *zap.Logger
	mu            sync.RWMutex
	maxConcurrent int
	maxRetries    int
	subscribed    bool
	// Metrics
	messagesConsumedTotal int64
	messagesAckedTotal    int64
	messagesNackedTotal   int64
	processingErrorsTotal int64
}

// ConsumerConfig holds consumer configuration
type ConsumerConfig struct {
	Client        *NATSClient
	Subject       string
	QueueGroup    string
	Handler       MessageHandler
	MaxConcurrent int
	MaxRetries    int
	Logger        *zap.Logger
}

// NewConsumer creates a new consumer instance
func NewConsumer(config ConsumerConfig) (*Consumer, error) {
	if config.Subject == "" {
		return nil, ErrEmptySubject
	}

	if config.Client == nil {
		return nil, errors.New("client cannot be nil")
	}

	logger := config.Logger
	if logger == nil {
		logger, _ = zap.NewProduction()
	}

	// Set defaults
	maxConcurrent := config.MaxConcurrent
	if maxConcurrent <= 0 {
		maxConcurrent = 1
	}

	maxRetries := config.MaxRetries
	if maxRetries < 0 {
		maxRetries = 2 // Default from requirements
	}

	return &Consumer{
		client:        config.Client,
		subject:       config.Subject,
		queueGroup:    config.QueueGroup,
		handler:       config.Handler,
		logger:        logger,
		maxConcurrent: maxConcurrent,
		maxRetries:    maxRetries,
		subscribed:    false,
	}, nil
}

// Subscribe subscribes to the subject and starts consuming messages
func (c *Consumer) Subscribe(ctx context.Context, handler MessageHandler) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if handler == nil {
		return ErrNilHandler
	}

	if c.subscribed {
		return ErrAlreadySubscribed
	}

	if !c.client.IsConnected() {
		return ErrNotConnected
	}

	c.handler = handler

	conn := c.client.GetConnection()
	if conn == nil {
		return ErrNotConnected
	}

	// Create message handler wrapper
	msgHandler := func(msg *nats.Msg) {
		c.handleMessage(ctx, msg)
	}

	// Subscribe with or without queue group
	var sub *nats.Subscription
	var err error

	if c.queueGroup != "" {
		sub, err = conn.QueueSubscribe(c.subject, c.queueGroup, msgHandler)
	} else {
		sub, err = conn.Subscribe(c.subject, msgHandler)
	}

	if err != nil {
		c.logger.Error("failed to subscribe",
			zap.String("subject", c.subject),
			zap.String("queue_group", c.queueGroup),
			zap.Error(err),
		)
		return fmt.Errorf("subscribe failed: %w", err)
	}

	c.subscription = sub
	c.subscribed = true

	c.logger.Info("subscribed to subject",
		zap.String("subject", c.subject),
		zap.String("queue_group", c.queueGroup),
	)

	return nil
}

// Unsubscribe unsubscribes from the subject
func (c *Consumer) Unsubscribe(drainTimeout time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.subscribed || c.subscription == nil {
		return nil // Already unsubscribed
	}

	var err error
	if drainTimeout > 0 {
		// Drain subscription gracefully
		err = c.subscription.Drain()
	} else {
		err = c.subscription.Unsubscribe()
	}

	if err != nil {
		c.logger.Warn("failed to unsubscribe",
			zap.String("subject", c.subject),
			zap.Error(err),
		)
		return err
	}

	c.subscribed = false
	c.logger.Info("unsubscribed from subject", zap.String("subject", c.subject))

	return nil
}

// handleMessage processes a single message
func (c *Consumer) handleMessage(ctx context.Context, msg *nats.Msg) {
	c.incrementMessagesConsumed()

	// Parse message to check retry count
	var msgData map[string]interface{}
	if err := json.Unmarshal(msg.Data, &msgData); err != nil {
		c.logger.Error("failed to parse message", zap.Error(err))
		c.incrementProcessingErrors()
		// Ack malformed messages to avoid infinite retries
		msg.Ack()
		c.incrementMessagesAcked()
		return
	}

	retryCount := 0
	if rc, ok := msgData["retry_count"].(float64); ok {
		retryCount = int(rc)
	}

	// Process message with handler
	err := c.handler(ctx, msg.Data)
	if err != nil {
		c.incrementProcessingErrors()
		c.logger.Error("message processing failed",
			zap.Error(err),
			zap.Int("retry_count", retryCount),
		)

		// Check if we should retry
		if retryCount < c.maxRetries {
			// Nack for redelivery
			msg.Nak()
			c.incrementMessagesNacked()
			c.logger.Info("message nacked for retry",
				zap.Int("retry_count", retryCount),
				zap.Int("max_retries", c.maxRetries),
			)
		} else {
			// Max retries reached, ack to remove from queue
			msg.Ack()
			c.incrementMessagesAcked()
			c.logger.Warn("max retries reached, message acked",
				zap.Int("retry_count", retryCount),
			)
		}
		return
	}

	// Success - ack message
	msg.Ack()
	c.incrementMessagesAcked()
	c.logger.Debug("message processed successfully")
}

// IsSubscribed returns subscription status
func (c *Consumer) IsSubscribed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.subscribed
}

// GetMetrics returns consumer metrics
func (c *Consumer) GetMetrics() map[string]int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]int64{
		"messages_consumed_total": c.messagesConsumedTotal,
		"messages_acked_total":    c.messagesAckedTotal,
		"messages_nacked_total":   c.messagesNackedTotal,
		"processing_errors_total": c.processingErrorsTotal,
	}
}

// incrementMessagesConsumed increments consumed messages counter
func (c *Consumer) incrementMessagesConsumed() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.messagesConsumedTotal++
}

// incrementMessagesAcked increments acked messages counter
func (c *Consumer) incrementMessagesAcked() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.messagesAckedTotal++
}

// incrementMessagesNacked increments nacked messages counter
func (c *Consumer) incrementMessagesNacked() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.messagesNackedTotal++
}

// incrementProcessingErrors increments processing errors counter
func (c *Consumer) incrementProcessingErrors() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.processingErrorsTotal++
}
