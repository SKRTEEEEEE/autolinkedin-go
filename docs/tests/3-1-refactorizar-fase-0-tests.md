# Tests Generated for Fase 0 Refactoring

This document outlines all tests generated for the Fase 0 refactoring task, which focuses on updating the entity structure, repositories, and use cases to support the new system described in the entity.md file.

## Test Files Created

### 1. Domain Entity Tests

#### Topic Entity Tests
- **File**: `test/domain/entities/topic_refactored_test.go`
- **Coverage**: Tests for the updated Topic entity with new fields
  - `ideas`: Number of ideas to generate from this topic (default 2)
  - `prompt`: Reference to the prompt to use (default "base1")
  - `related_topics`: Array of related topic names
- **Assertions**:
  - Topic validation with new required fields
  - Default values for new fields
  - Ideas count range validation
  - Prompt reference validation
  - Related topics handling and deduplication
  - Prompt context generation with new fields

#### Prompt Entity Tests
- **File**: `test/domain/entities/prompt_refactored_test.go`
- **Coverage**: Tests for the updated Prompt entity according to entity.md
  - `name`: [unique] identifier (replacing StyleName)
  - `type`: For what the prompt is used (ideas | draft)
  - `prompt_template`: Plain text template with variable placeholders
  - `active`: Boolean indicating if prompt is active
  - `user_id`: ID of the user using the prompt
- **Assertions**:
  - Prompt validation with new structure
  - Name field uniqueness and validation
  - Type field validation (ideas or draft)
  - Prompt template validation with placeholders
  - Active field default behavior
  - Template variable extraction

#### Idea Entity Tests
- **File**: `test/domain/entities/idea_refactored_test.go`
- **Coverage**: Tests for the updated Idea entity with the topic_name field
  - `content`: Text of the idea (10-200 characters, reduced from 5000)
  - `quality_score`: Optional score (0.0-1.0, default 0.0)
  - `topic_name`: (NEW) unique name of the related topic
- **Assertions**:
  - Idea validation with new structure and topic_name field
  - Content length validation with reduced limits
  - Quality score default behavior
  - Used field default behavior and expiration calculation
  - Topic name field validation and format
  - BelongsToTopicByName method functionality

### 2. Repository Tests

#### Prompt Repository Tests
- **File**: `test/infrastructure/database/repositories/prompts_repository_refactored_test.go`
- **Coverage**: Tests for the new PromptRepository methods to handle the refactored structure
- **New Methods**:
  - `FindByName`: Find prompts by name (new field used as identifier)
  - `FindActiveByName`: Find active prompts by name and user ID
  - `FindByNameAllUsers`: Find all prompts with a specific name across all users
- **Assertions**:
  - Prompt lookup by name and user ID
  - Active/inactive prompt filtering
  - Multi-user prompt handling
  - Integration with existing CRUD operations
  - Name uniqueness validation at repository level

#### Topic Repository Tests
- **File**: `test/infrastructure/database/repositories/topic_repository_refactored_test.go`
- **Coverage**: Tests for the new TopicRepository methods to handle the refactored structure
- **New Methods**:
  - `FindByPrompt`: Find topics by prompt reference
  - `FindByIdeasRange`: Find topics with ideas count within specified range
  - `FindWithFilters`: Find topics using multiple filters
- **Assertions**:
  - Topic CRUD operations with new fields (ideas, prompt, related_topics)
  - Prompt reference searching and filtering
  - Ideas count range filtering
  - Advanced multi-field filtering
  - Related topics field handling
  - Backward compatibility with existing methods

### 3. Services Tests

#### Dev Seeder Tests
- **File**: `test/application/services/dev_seeder_refactored_test.go`
- **Coverage**: Tests for the new DevSeeder functionality
  - Reading prompts from seed/prompt/*.md files
  - Parsing front-matter YAML (name, type, content)
  - Seeding topics with prompt references
  - Synchronizing file changes to database
- **New Methods**:
  - `ParsePromptFile`: Parse a prompt file with front-matter
  - `SeedPromptsFromFiles`: Seed prompts from the file system
  - `SeedDefaultTopics`: Seed default development topics
  - `CreateTopicWithConfig`: Create a topic with the given configuration
  - `SeedAll`: Run the complete seeding process
- **Assertions**:
  - Front-matter parsing for both idea and draft type prompts
  - Invalid front-matter handling
  - Prompt updating when files change
  - Topic creation with prompt references
  - Ideas count configuration
  - Prompt reference validation during seeding

### 4. Use Cases Tests

#### Generate Ideas UseCase Tests
- **File**: `test/application/usecases/generate_ideas_usecase_refactored_test.go`
- **Coverage**: Tests for the new GenerateIdeasUseCase functionality
  - Use specific prompt references from topics
  - Support the new prompt template format with variables
  - Create ideas with topic_name field
  - Generate correct number of ideas based on topic configuration
- **New Methods**:
  - Update to `GenerateIdeas` to use prompt references
  - `processPromptTemplate`: Replace template variables with actual values
  - `parseLLMResponse`: Parse and validate the LLM response
- **Assertions**:
  - Ideas generation using topic's specific prompt reference
  - Error handling for non-existent prompt references
  - Prompt template variable processing
  - Ideas count validation
  - Topic_name field in generated ideas
  - Default ideas count behavior

### 5. Integration Tests

#### Full Workflow Integration Tests
- **File**: `test/integration/refactoring_fase_0_test.go`
- **Coverage**: Tests the complete refactored flow from database seeding to idea generation
- **Test Scenarios**:
  - Complete refactored workflow from seeding to idea generation
  - Prompt reference constraints validation
  - Topic-idea relationships with new fields
  - HTTP endpoints integration (placeholder)
- **Components Tested**:
  - Database repositories with MongoDB
  - File system parsing and seeding
  - Use case orchestration
  - Entity relationship integrity

## Key Test Scenarios Covered

1. **Entity Structure Refactoring**
   - Validation of all new fields
   - Default values and constraints
   - Backward compatibility with existing code

2. **Repository Extensions**
   - New query methods for prompt references
   - Advanced filtering capabilities
   - Integration with existing CRUD operations

3. **File-Based Seeding System**
   - Front-matter parsing from Markdown files
   - Dynamic prompt loading from file system
   - Synchronization between files and database

4. **Use Case Integration**
   - Prompt reference resolution
   - Template variable processing
   - Topic-idea relationship maintenance

5. **End-to-End Workflows**
   - Complete system initialization
   - Idea generation with new structure
   - Data integrity across components

## Running the Tests

To run all tests for the Fase 0 refactoring:

```bash
# Run domain entity tests
go test ./test/domain/entities/... -v

# Run repository tests
go test ./test/infrastructure/database/repositories/... -v

# Run services tests
go test ./test/application/services/... -v

# Run use cases tests
go test ./test/application/usecases/... -v

# Run integration tests
go test ./test/integration/... -v

# Run all refactoring tests
go test ./test/... -run ".*Refactored.*" -v
```

## Test Dependencies

The tests require the following dependencies:
- MongoDB instance for repository tests
- Mock implementations for external dependencies
- Test data fixtures in seed/prompt/*.md format

## Next Steps

These tests are designed to ensure that the Fase 0 refactoring successfully:

1. Updates entity structures to match entity.md specifications
2. Implements proper relationships between topics and prompts
3. Enables file-based prompt management
4. Maintains backward compatibility with existing features
5. Provides a foundation for subsequent refactoring phases

The tests will fail initially (TDD Red phase) as the code implementation doesn't yet exist to support these requirements. As the implementation progresses, these tests should pass, confirming that the refactoring meets all specified requirements.
