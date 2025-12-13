# Prompt System Tests Generated (TDD Red)

This document summarizes all test files generated for the prompt system implementation following the TDD Red pattern. These tests will FAIL until the corresponding implementation exists.

## Test Files Generated

### 1. PromptLoader Service Tests (3-3-1)
**File**: `test/infrastructure/services/prompt_loader_test.go`

Coverage:
- Loading prompts from seed/prompts/*.md files
- Parsing front-matter (name, type) and content
- Filtering out .old.md files
- Detecting changes between seed and database
- Synchronizing seed files with database

### 2. Refactored PromptRepository Tests (3-3-2)
**File**: `test/infrastructure/database/repositories/prompt_repository_synchronization_test.go`

Coverage:
- Validating prompt template syntax
- Extracting variables from templates
- Validating required variables by prompt type
- Activating/deactivating prompts by name
- Finding prompts by name with type validation
- Resetting user prompts to defaults from seed files

### 3. Enhanced PromptHandler API Tests (3-3-3)
**File**: `test/interfaces/handlers/prompts_handler_enhanced_test.go`

Coverage:
- Getting specific prompt by name
- Creating custom prompts with names
- Resetting user prompts to defaults
- Validating prompt template syntax via API
- Listing available default prompts
- Activating/deactivating specific prompts
- Handling duplicate prompt names

### 4. PromptEngine Service Tests (3-3-4)
**File**: `test/infrastructure/services/prompt_engine_test.go`

Coverage:
- Processing prompts with variable substitution for ideas and drafts
- Caching processed prompts
- Falling back to default prompts for missing custom prompts
- Building user context from profile data
- Handling empty related topics arrays
- Validating required variables before processing
- Initializing prompt engine with seed defaults

### 5. Integration Tests for Generation Flow (3-3-5)
**File**: `test/application/usecases/prompt_system_integration_test.go`

Coverage:
- GenerateIdeasUseCase with PromptEngine integration
- GenerateDraftsUseCase with PromptEngine integration
- Fallback to default prompts when user has no prompts
- Logging prompt system usage and errors
- Handling prompt errors gracefully
- Using cached prompts when available
- Diagnostic endpoints for prompt system

## Test Pattern

All tests follow the TDD Red pattern:
1. Tests are written to FAIL intentionally
2. Tests specify the exact behavior expected from the implementation
3. Tests include comprehensive edge cases and error scenarios
4. Tests adhere to the project's testing guidelines in AGENTS.md

## Next Steps

To implement the prompt system:
1. Implement PromptLoader service in `src/infrastructure/services/`
2. Add required methods to PromptRepository in `src/infrastructure/database/repositories/`
3. Enhance PromptHandler in `src/interfaces/handlers/`
4. Implement PromptEngine in `src/infrastructure/services/`
5. Update GenerateIdeasUseCase and GenerateDraftsUseCase to use PromptEngine
6. Ensure all tests pass (TDD Green phase)
