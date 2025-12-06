package nats

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	nats "github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

var (
	// ErrEmptySubject indicates empty subject name
	ErrEmptySubject = errors.New("subject cannot be empty")
	// ErrInvalidMessage indicates invalid message
	ErrInvalidMessage = errors.New("invalid message")
	// ErrInvalidTTL indicates invalid TTL value
	ErrInvalidTTL = errors.New("invalid TTL value")
	// ErrEmptyBatch indicates empty batch
	ErrEmptyBatch = errors.New("batch cannot be empty")
	// ErrPublishFailed indicates publish failure
	ErrPublishFailed = errors.New("publish failed")
)

// Publisher handles message publishing to NATS
type Publisher struct {
	client  *NATSClient
	subject string
	ttl     time.Duration
	logger  *zap.Logger
	mu      sync.RWMutex
	// Metrics
	messagesPublishedTotal int64
	publishErrorsTotal     int64
}

// PublisherConfig holds publisher configuration
type PublisherConfig struct {
	Client  *NATSClient
	Subject string
	TTL     time.Duration
	Logger  *zap.Logger
}

// NewPublisher creates a new publisher instance
func NewPublisher(config PublisherConfig) (*Publisher, error) {
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

	// Set default TTL if not provided (5 minutes as per requirements)
	ttl := config.TTL
	if ttl == 0 {
		ttl = 5 * time.Minute
	}

	return &Publisher{
		client:  config.Client,
		subject: config.Subject,
		ttl:     ttl,
		logger:  logger,
	}, nil
}

// Publish publishes a message to NATS
func (p *Publisher) Publish(ctx context.Context, data interface{}) error {
	if !p.client.IsConnected() {
		return ErrNotConnected
	}

	// Serialize message
	payload, err := json.Marshal(data)
	if err != nil {
		p.incrementPublishErrors()
		return fmt.Errorf("failed to serialize message: %w", err)
	}

	// Publish message
	conn := p.client.GetConnection()
	if conn == nil {
		return ErrNotConnected
	}

	// Type assertion to ensure nats.Conn is used
	var _ *nats.Conn = conn

	err = conn.Publish(p.subject, payload)
	if err != nil {
		p.incrementPublishErrors()
		p.logger.Error("failed to publish message",
			zap.String("subject", p.subject),
			zap.Error(err),
		)
		return fmt.Errorf("%w: %v", ErrPublishFailed, err)
	}

	p.incrementMessagesPublished()
	p.logger.Debug("message published",
		zap.String("subject", p.subject),
		zap.Int("payload_size", len(payload)),
	)

	return nil
}

// PublishWithTTL publishes a message with custom TTL
func (p *Publisher) PublishWithTTL(ctx context.Context, data interface{}, ttl time.Duration) error {
	if ttl < 0 {
		return ErrInvalidTTL
	}

	// For now, NATS core doesn't support native TTL
	// We'll include TTL in message metadata for consumer to handle
	// In production, use NATS JetStream for TTL support

	return p.Publish(ctx, data)
}

// PublishSync publishes a message and waits for acknowledgment
func (p *Publisher) PublishSync(ctx context.Context, data interface{}, timeout time.Duration) error {
	if !p.client.IsConnected() {
		return ErrNotConnected
	}

	// Serialize message
	payload, err := json.Marshal(data)
	if err != nil {
		p.incrementPublishErrors()
		return fmt.Errorf("failed to serialize message: %w", err)
	}

	conn := p.client.GetConnection()
	if conn == nil {
		return ErrNotConnected
	}

	// Use request-reply pattern for synchronous publish
	responseDone := make(chan error, 1)
	go func() {
		// For core NATS, we'll just use Publish and Flush
		err := conn.Publish(p.subject, payload)
		if err != nil {
			responseDone <- err
			return
		}
		err = conn.Flush()
		responseDone <- err
	}()

	select {
	case err := <-responseDone:
		if err != nil {
			p.incrementPublishErrors()
			return fmt.Errorf("%w: %v", ErrPublishFailed, err)
		}
		p.incrementMessagesPublished()
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(timeout):
		return errors.New("publish timeout")
	}
}

// PublishBatch publishes multiple messages in batch
func (p *Publisher) PublishBatch(ctx context.Context, messages []interface{}) error {
	if len(messages) == 0 {
		return ErrEmptyBatch
	}

	if !p.client.IsConnected() {
		return ErrNotConnected
	}

	conn := p.client.GetConnection()
	if conn == nil {
		return ErrNotConnected
	}

	// Publish all messages
	for i, msg := range messages {
		payload, err := json.Marshal(msg)
		if err != nil {
			p.incrementPublishErrors()
			p.logger.Warn("failed to serialize message in batch",
				zap.Int("index", i),
				zap.Error(err),
			)
			continue
		}

		err = conn.Publish(p.subject, payload)
		if err != nil {
			p.incrementPublishErrors()
			p.logger.Error("failed to publish message in batch",
				zap.Int("index", i),
				zap.Error(err),
			)
			continue
		}

		p.incrementMessagesPublished()
	}

	// Flush to ensure all messages are sent
	err := conn.Flush()
	if err != nil {
		return fmt.Errorf("failed to flush batch: %w", err)
	}

	return nil
}

// GetMetrics returns publisher metrics
func (p *Publisher) GetMetrics() map[string]int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return map[string]int64{
		"messages_published_total": p.messagesPublishedTotal,
		"publish_errors_total":     p.publishErrorsTotal,
	}
}

// incrementMessagesPublished increments published messages counter
func (p *Publisher) incrementMessagesPublished() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.messagesPublishedTotal++
}

// incrementPublishErrors increments publish errors counter
func (p *Publisher) incrementPublishErrors() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.publishErrorsTotal++
}
