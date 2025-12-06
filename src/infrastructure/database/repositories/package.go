// Package repositories provides concrete MongoDB implementations of domain repository interfaces.
// These repositories are responsible for persisting and retrieving domain entities from MongoDB,
// following the Clean Architecture principle of dependency inversion.
//
// Repository Implementations:
// - UserRepository: Manages user persistence and authentication tokens
// - TopicRepository: Handles user topics for content generation
// - IdeasRepository: Manages the idea backlog with batch operations
// - DraftRepository: Persists drafts with refinement history and status tracking
//
// All repositories implement their corresponding interfaces defined in domain/interfaces
// and use the BaseRepository from infrastructure/database for common CRUD operations.
package repositories
