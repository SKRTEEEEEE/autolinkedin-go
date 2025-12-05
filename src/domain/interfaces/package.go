// Package interfaces defines the abstract ports (interfaces) for the domain layer.
// These interfaces are implemented by the infrastructure layer, ensuring that
// the domain layer has no dependencies on external concerns.
//
// Core Interfaces:
// - LLMService: Interface for LLM interactions
// - DraftRepository: Interface for draft persistence
// - IdeasRepository: Interface for ideas persistence
// - TopicsRepository: Interface for topics persistence
// - UserRepository: Interface for user persistence
// - PublisherService: Interface for LinkedIn publishing
// - QueueService: Interface for async job queuing
package interfaces
