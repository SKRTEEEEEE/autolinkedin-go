// Package usecases contains the application use cases for LinkGen AI.
// Use cases orchestrate the flow of data to and from entities, and direct
// those entities to use their business rules to achieve the goals of the use case.
//
// Core Use Cases:
// - GenerateIdeasUseCase: Automated periodic idea generation from topics
// - GenerateDraftsUseCase: Create drafts from ideas (5 posts + 1 article)
// - RefineDraftUseCase: Refine existing drafts with user feedback
// - ListIdeasUseCase: List accumulated ideas with optional filters
// - ClearIdeasUseCase: Clear accumulated ideas for a user
//
// Each use case follows Clean Architecture principles:
// - Depends only on domain interfaces and entities
// - No dependencies on infrastructure or frameworks
// - Business logic orchestration with proper error handling
package usecases
