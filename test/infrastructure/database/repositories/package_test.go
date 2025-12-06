// Package repositories contains integration tests for MongoDB repository implementations.
// These tests validate the concrete implementations of domain repository interfaces.
//
// Test Coverage:
// - UserRepository: User CRUD operations, LinkedIn token management, email lookup
// - TopicRepository: Topic CRUD operations, user filtering, random selection
// - IdeasRepository: Batch creation, user filtering, clearing, counting
// - DraftRepository: Draft CRUD, status management, refinement history, publishing workflow
//
// All tests follow TDD Red phase - they will FAIL until implementations exist.
package repositories
