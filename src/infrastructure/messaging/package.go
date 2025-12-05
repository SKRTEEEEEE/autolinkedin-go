// Package messaging provides NATS-based queue implementation for async operations.
// This package handles job queuing for long-running operations like draft generation.
//
// Components:
// - NATSQueueService: Queue service implementation
// - Worker: NATS message consumer for background jobs
// - Publisher: Message publishing for job submission
package messaging
