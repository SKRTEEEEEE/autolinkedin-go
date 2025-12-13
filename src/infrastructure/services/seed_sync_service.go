package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/linkgen-ai/backend/src/domain/entities"
	"github.com/linkgen-ai/backend/src/infrastructure/database/repositories"
)

// SeedSyncService handles synchronization between seed files and database
type SeedSyncService struct {
	db                *mongo.Database
	promptRepo        repositories.PromptRepository
	topicRepo         repositories.TopicRepository
	promptLoader      *PromptLoader
	promptEngine      *PromptEngine
}

// NewSeedSyncService creates a new seed sync service
func NewSeedSyncService(db *mongo.Database, promptLoader *PromptLoader, promptEngine *PromptEngine) *SeedSyncService {
	return &SeedSyncService{
		db:           db,
		promptRepo:   repositories.NewPromptRepository(db),
		topicRepo:    repositories.NewTopicRepository(db),
		promptLoader: promptLoader,
		promptEngine: promptEngine,
	}
}

// SeedPromptsFromFiles loads prompts from seed/prompt/ directory and syncs with database
func (s *SeedSyncService) SeedPromptsFromFiles(ctx context.Context, userID string, seedDir string) error {
	log.Printf("Seeding prompts from directory: %s", seedDir)

	// Read all prompt files
	files, err := os.ReadDir(seedDir)
	if err != nil {
		return fmt.Errorf("failed to read seed directory: %v", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// Skip non-md files for now (assuming .md format)
		if filepath.Ext(file.Name()) != ".md" {
			log.Printf("Skipping non-markdown file: %s", file.Name())
			continue
		}

		filePath := filepath.Join(seedDir, file.Name())
		if err := s.processPromptFile(ctx, userID, filePath); err != nil {
			log.Printf("Error processing prompt file %s: %v", filePath, err)
			continue
		}
	}

	log.Println("Prompt seeding completed")
	return nil
}

// processPromptFile processes a single prompt file and syncs it to the database
func (s *SeedSyncService) processPromptFile(ctx context.Context, userID, filePath string) error {
	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read prompt file: %v", err)
	}

	// Parse as a structured prompt
	prompt, err := s.parsePromptFile(content, userID)
	if err != nil {
		return fmt.Errorf("failed to parse prompt file: %v", err)
	}

	// Check if prompt already exists for this user
	existing, err := s.promptRepo.GetByName(ctx, userID, prompt.Name)
	if err == nil && existing != nil {
		// Update existing prompt
		prompt.ID = existing.ID
		prompt.CreatedAt = existing.CreatedAt
		prompt.UpdatedAt = time.Now()
		
		if err := s.promptRepo.Update(ctx, prompt); err != nil {
			return fmt.Errorf("failed to update existing prompt: %v", err)
		}
		log.Printf("Updated existing prompt: %s", prompt.Name)
	} else {
		// Create new prompt
		prompt.ID = primitive.NewObjectID().Hex()
		prompt.CreatedAt = time.Now()
		prompt.UpdatedAt = time.Now()

		if err := s.promptRepo.Create(ctx, prompt); err != nil {
			return fmt.Errorf("failed to create new prompt: %v", err)
		}
		log.Printf("Created new prompt: %s", prompt.Name)
	}

	return nil
}

// parsePromptFile parses the content of a prompt file into a Prompt entity
func (s *SeedSyncService) parsePromptFile(content []byte, userID string) (*entities.Prompt, error) {
	// Extract metadata from filename
	// Format: base1.idea.md, pro.draft.md, etc.
	// name.type.md pattern
	
	// For now, we'll create a basic structure based on file name patterns
	// In a full implementation, this would need proper parsing of markdown files
	
	// Default values
	prompt := &entities.Prompt{
		UserID:         userID,
		Name:           "default",
		Type:           entities.PromptTypeIdeas,
		PromptTemplate: string(content),
		Active:         true,
	}

	// Simple parsing based on common patterns
	contentStr := string(content)
	
	// Detect type based on content or filename
	if contains(contentStr, []string{"draft", "article", "content", "post"}) {
		prompt.Type = entities.PromptTypeDrafts
	}
	
	// Try to extract name from file template
	if contains(contentStr, []string{"professional", "pro"}) {
		prompt.Name = "professional"
	} else if contains(contentStr, []string{"creative", "innovative"}) {
		prompt.Name = "creative"
	} else if contains(contentStr, []string{"base", "basic", "simple"}) {
		prompt.Name = "base1"
	}

	return prompt, nil
}

// contains checks if content contains any of the substrings
func contains(content string, substrings []string) bool {
	for _, substr := range substrings {
		if len(content) > len(substr) {
			for i := 0; i <= len(content)-len(substr); i++ {
				if content[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}

// SeedTopicsFromFile loads topics from seed/topic.json and syncs with database
func (s *SeedSyncService) SeedTopicsFromFile(ctx context.Context, userID, topicFile string) error {
	log.Printf("Seeding topics from file: %s", topicFile)

	// Read file content
	content, err := os.ReadFile(topicFile)
	if err != nil {
		return fmt.Errorf("failed to read topic file: %v", err)
	}

	// Parse topics
	var seedTopics []entities.Topic
	if err := json.Unmarshal(content, &seedTopics); err != nil {
		return fmt.Errorf("failed to parse topic file: %v", err)
	}

	// Process each topic
	for i := range seedTopics {
		topic := &seedTopics[i]
		topic.UserID = userID
		topic.CreatedAt = time.Now()
		topic.UpdatedAt = time.Now()

		// Ensure required fields have default values
		if topic.PromptName == "" {
			topic.PromptName = "base1"
		}
		if topic.IdeasCount == 0 {
			topic.IdeasCount = 3
		}
		if topic.Priority == 0 {
			topic.Priority = 5
		}

		// Check if topic already exists for this user
		existing, err := s.topicRepo.GetByName(ctx, userID, topic.Name)
		if err == nil && existing != nil {
			// Update existing topic
			topic.ID = existing.ID
			topic.CreatedAt = existing.CreatedAt
			topic.UpdatedAt = time.Now()
			
			if err := s.topicRepo.Update(ctx, topic); err != nil {
				log.Printf("Error updating topic %s: %v", topic.Name, err)
				continue
			}
			log.Printf("Updated existing topic: %s", topic.Name)
		} else {
			// Create new topic
			topic.ID = primitive.NewObjectID().Hex()
			
			if err := s.topicRepo.Create(ctx, topic); err != nil {
				log.Printf("Error creating topic %s: %v", topic.Name, err)
				continue
			}
			log.Printf("Created new topic: %s", topic.Name)
		}

		// Generate ideas for this topic using its associated prompt
		if err := s.generateIdeasForTopic(ctx, userID, topic); err != nil {
			log.Printf("Error generating ideas for topic %s: %v", topic.Name, err)
		}
	}

	log.Println("Topic seeding completed")
	return nil
}

// generateIdeasForTopic generates ideas for a topic using its associated prompt
func (s *SeedSyncService) generateIdeasForTopic(ctx context.Context, userID string, topic *entities.Topic) error {
	// Get the prompt specified by the topic
	prompt, err := s.promptRepo.GetByName(ctx, userID, topic.PromptName)
	if err != nil {
		return fmt.Errorf("failed to get prompt %s: %v", topic.PromptName, err)
	}

	// Process the prompt template with topic variables
	processedPrompt, err := s.promptEngine.ProcessTemplate(
		prompt.PromptTemplate,
		topic,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to process prompt template: %v", err)
	}

	// For now, we'll create mock ideas until LLM integration is complete
	// In a full implementation, this would call the LLM service
	ideas := s.createMockIdeas(topic, topic.IdeasCount)

	// Save ideas to database
	ideaRepo := repositories.NewIdeaRepository(s.db)
	for _, idea := range ideas {
		ideaID := primitive.NewObjectID().Hex()
		idea := &entities.Idea{
			ID:        ideaID,
			Content:   idea.Content,
			TopicID:   topic.ID,
			TopicName: topic.Name,
			UserID:    userID,
			Used:      false,
			CreatedAt: time.Now(),
		}

		if err := ideaRepo.Create(ctx, idea); err != nil {
			log.Printf("Failed to save idea: %v", err)
		}
	}

	log.Printf("Generated %d ideas for topic %s", len(ideas), topic.Name)
	return nil
}

// createMockIdeas creates mock ideas for development purposes
func (s *SeedSyncService) createMockIdeas(topic *entities.Topic, count int) []*entities.Idea {
	ideas := make([]*entities.Idea, count)
	
	for i := 0; i < count; i++ {
		ideaNum := i + 1
		ideas[i] = &entities.Idea{
			Content: fmt.Sprintf("Idea #%d for %s: %s approach with %s", 
				ideaNum, topic.Name, 
				getRandomAdjective(), getRandomTechnology()),
		}
	}
	
	return ideas
}

// getRandomAdjective returns a random adjective for idea generation
func getRandomAdjective() string {
	adjectives := []string{"Innovative", "Practical", "Advanced", "Modern", "Efficient", "Scalable"}
	return adjectives[time.Now().Nanosecond()%len(adjectives)]
}

// getRandomTechnology returns a random technology keyword
func getRandomTechnology() string {
	technologies := []string{"microservices", "cloud-native", "AI-driven", "serverless", "containerized", "event-driven"}
	return technologies[time.Now().Nanosecond()%len(technologies)]
}

// ValidateSeedConfiguration validates that seed configuration matches database state
func (s *SeedSyncService) ValidateSeedConfiguration(ctx context.Context, userID string) error {
	log.Printf("Validating seed configuration for user %s", userID)

	// Get all prompts for user
	prompts, err := s.promptRepo.ListByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to list user prompts: %v", err)
	}

	// Get all topics for user
	topics, err := s.topicRepo.ListByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to list user topics: %v", err)
	}

	// Validate topic-prompt references
	for _, topic := range topics {
		// Check if referenced prompt exists
		promptExists := false
		for _, prompt := range prompts {
			if prompt.Name == topic.PromptName {
				promptExists = true
				break
			}
		}

		if !promptExists {
			return fmt.Errorf("topic %s references non-existent prompt %s", topic.Name, topic.PromptName)
		}
	}

	log.Printf("Seed configuration validation passed. Prompts: %d, Topics: %d", len(prompts), len(topics))
	return nil
}

// SyncStateToDatabase ensures database state is consistent with seed configuration
func (s *SeedSyncService) SyncStateToDatabase(ctx context.Context, userID, seedDir string) error {
	log.Printf("Syncing seed state to database for user %s", userID)

	// Get seed directory paths
	promptDir := filepath.Join(seedDir, "prompt")
	topicFile := filepath.Join(seedDir, "topic.json")

	// Seed prompts
	if err := s.SeedPromptsFromFiles(ctx, userID, promptDir); err != nil {
		return fmt.Errorf("failed to seed prompts: %v", err)
	}

	// Seed topics
	if err := s.SeedTopicsFromFile(ctx, userID, topicFile); err != nil {
		return fmt.Errorf("failed to seed topics: %v", err)
	}

	// Validate final state
	if err := s.ValidateSeedConfiguration(ctx, userID); err != nil {
		return fmt.Errorf("validation failed after sync: %v", err)
	}

	log.Println("Seed state sync completed successfully")
	return nil
}
