// Package nats provides NATS-based messaging infrastructure for async operations.
// This package implements the NATS client, publisher, and consumer components
// for the LinkGen AI application.
//
// Components:
// - NATSClient: Connection management and health monitoring
// - Publisher: Message publishing with TTL and batch support
// - Consumer: Message consumption with retry logic and queue groups
//
// The implementation follows Clean Architecture principles and provides
// a simple, lightweight queue for draft generation jobs.
package nats
