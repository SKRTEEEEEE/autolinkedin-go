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
	"github.com/linkgen-ai/backend/src/domain/interfaces"
	"github.com/linkgen-ai/backend/src/infrastructure/database/repositories"
)

// SeedSyncService handles synchronization between seed files and database
type SeedSyncService struct {
	db           *mongo.Database
	promptRepo   interfaces.PromptsRepository
	topicRepo    interfaces.TopicRepository
	promptLoader *PromptLoader
	promptEngine *PromptEngine
}

// NewSeedSyncService creates a new seed sync service
func NewSeedSyncService(db *mongo.Database, promptLoader *PromptLoader, promptEngine *PromptEngine) *SeedSyncService {
	return &SeedSyncService{
		db:           db,
		promptRepo:   repositories.NewPromptsRepository(db.Collection("prompts")),
		topicRepo:    repositories.NewTopicRepository(db.Collection("topics")),
		promptLoader: promptLoader,
		promptEngine: promptEngine,
	}
}

// SeedPromptsFromFiles loads prompts from seed/prompt/ directory and syncs with database
func (s *SeedSyncService) SeedPromptsFromFiles(ctx context.Context, userID string, seedDir string) error {
	log.Printf("Seeding prompts from directory: %s", seedDir)

	// Load prompts from files using PromptLoader
	promptFiles, err := s.promptLoader.LoadPromptsFromDir(seedDir)
	if err != nil {
		return fmt.Errorf("failed to load prompts from directory: %v", err)
	}

	if len(promptFiles) == 0 {
		log.Printf("No prompt files found in directory: %s", seedDir)
		return nil
	}

	// Create prompt entities from files
	prompts, err := s.promptLoader.CreatePromptsFromFile(userID, promptFiles)
	if err != nil {
		return fmt.Errorf("failed to create prompt entities: %v", err)
	}

	if len(prompts) == 0 {
		log.Printf("No valid prompts were created from files")
		return nil
	}

	// Sync prompts to database
	for _, prompt := range prompts {
		// Check if prompt already exists for this user
		existing, err := s.promptRepo.FindByName(ctx, userID, prompt.Name)
		if err == nil && existing != nil {
			// Update existing prompt
			prompt.ID = existing.ID
			prompt.CreatedAt = existing.CreatedAt
			prompt.UpdatedAt = time.Now()

			if err := s.promptRepo.Update(ctx, prompt); err != nil {
				log.Printf("Failed to update existing prompt %s: %v", prompt.Name, err)
				continue
			}
			log.Printf("Updated existing prompt: %s (type: %s)", prompt.Name, prompt.Type)
		} else {
			// Create new prompt
			if _, err := s.promptRepo.Create(ctx, prompt); err != nil {
				log.Printf("Failed to create new prompt %s: %v", prompt.Name, err)
				continue
			}
			log.Printf("Created new prompt: %s (type: %s)", prompt.Name, prompt.Type)
		}
	}

	log.Println("Prompt seeding completed")
	return nil
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
		if topic.Prompt == "" {
			topic.Prompt = "base1"
		}
		if topic.Ideas == 0 {
			topic.Ideas = 3
		}
		if topic.Priority == 0 {
			topic.Priority = 5
		}

		// Check if topic already exists for this user
		// We need to list all topics and find by name since there's no FindByName method
		topics, err := s.topicRepo.ListByUserID(ctx, userID)
		if err != nil {
			log.Printf("Error listing topics: %v", err)
		} else {
			var existing *entities.Topic
			for _, t := range topics {
				if t.Name == topic.Name {
					existing = t
					break
				}
			}

			if existing != nil {
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

				if _, err := s.topicRepo.Create(ctx, topic); err != nil {
					log.Printf("Error creating topic %s: %v", topic.Name, err)
					continue
				}
				log.Printf("Created new topic: %s", topic.Name)
			}
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
	prompt, err := s.promptRepo.FindByName(ctx, userID, topic.Prompt)
	if err != nil {
		return fmt.Errorf("failed to get prompt %s: %v", topic.Prompt, err)
	}

	// Process the prompt template with topic variables
	_, err = s.promptEngine.ProcessTemplate(
		prompt.PromptTemplate,
		topic,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to process prompt template: %v", err)
	}

	// For now, we'll create mock ideas until LLM integration is complete
	// In a full implementation, this would call the LLM service
	generatedIdeas := s.createMockIdeas(topic, topic.Ideas)

	// Save ideas to database using batch API
	ideaRepo := repositories.NewIdeasRepository(s.db.Collection("ideas"))
	var ideasToStore []*entities.Idea
	for _, idea := range generatedIdeas {
		ideasToStore = append(ideasToStore, &entities.Idea{
			ID:        primitive.NewObjectID().Hex(),
			Content:   idea.Content,
			TopicID:   topic.ID,
			TopicName: topic.Name,
			UserID:    userID,
			Used:      false,
			CreatedAt: time.Now(),
		})
	}

	if len(ideasToStore) > 0 {
		if err := ideaRepo.CreateBatch(ctx, ideasToStore); err != nil {
			log.Printf("Failed to save ideas for topic %s: %v", topic.Name, err)
		}
	}

	log.Printf("Generated %d ideas for topic %s", len(ideasToStore), topic.Name)
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
			if prompt.Name == topic.Prompt {
				promptExists = true
				break
			}
		}

		if !promptExists {
			return fmt.Errorf("topic %s references non-existent prompt %s", topic.Name, topic.Prompt)
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
