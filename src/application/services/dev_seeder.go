package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/linkgen-ai/backend/src/application/usecases"
	"github.com/linkgen-ai/backend/src/domain/entities"
	"github.com/linkgen-ai/backend/src/domain/interfaces"
	"github.com/linkgen-ai/backend/src/infrastructure/services"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

const (
	// DevUserID is the fixed user ID for local development
	// Using a valid MongoDB ObjectID format (24 hex characters)
	DevUserID = "000000000000000000000001"
)

// DevSeeder handles seeding development data
type DevSeeder struct {
	userRepo      interfaces.UserRepository
	topicRepo     interfaces.TopicRepository
	ideasRepo     interfaces.IdeasRepository
	promptsRepo   interfaces.PromptsRepository
	promptEngine  *services.PromptEngine
	llmService    interfaces.LLMService
	logger        *zap.Logger
	promptDir     string
	topicSeedPath string
}

// DevSeederConfig allows customizing seed sources for testing
type DevSeederConfig struct {
	PromptDir     string
	TopicSeedPath string
}

// NewDevSeeder creates a new development data seeder
func NewDevSeeder(
	userRepo interfaces.UserRepository,
	topicRepo interfaces.TopicRepository,
	ideasRepo interfaces.IdeasRepository,
	promptsRepo interfaces.PromptsRepository,
	promptEngine *services.PromptEngine,
	llmService interfaces.LLMService,
	logger *zap.Logger,
	config *DevSeederConfig,
) *DevSeeder {
	promptDir := filepath.Join(".", "seed", "prompt")
	topicSeedPath := filepath.Join(".", "seed", "topic.json")

	if config != nil {
		if config.PromptDir != "" {
			promptDir = config.PromptDir
		}
		if config.TopicSeedPath != "" {
			topicSeedPath = config.TopicSeedPath
		}
	}

	return &DevSeeder{
		userRepo:      userRepo,
		topicRepo:     topicRepo,
		ideasRepo:     ideasRepo,
		promptsRepo:   promptsRepo,
		promptEngine:  promptEngine,
		llmService:    llmService,
		logger:        logger,
		promptDir:     promptDir,
		topicSeedPath: topicSeedPath,
	}
}

// SeedDevUser creates or updates the development user
func (s *DevSeeder) SeedDevUser(ctx context.Context) error {
	s.logger.Info("Seeding development user...")

	// Check if dev user already exists
	existingUser, err := s.userRepo.FindByID(ctx, DevUserID)
	if err == nil && existingUser != nil {
		s.logger.Info("Development user already exists", zap.String("user_id", DevUserID))
		return nil
	}

	// Create new dev user
	user := &entities.User{
		ID:            DevUserID,
		Email:         "dev@local.linkgen.ai",
		Language:      "es", // Spanish by default
		LinkedInToken: "dev-token-not-needed-for-local",
		APIKeys:       map[string]string{"dev": "key"},
		Configuration: map[string]interface{}{
			"auto_publish": false,
			"environment":  "development",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Active:    true,
	}

	// Validate user
	if err := user.Validate(); err != nil {
		return fmt.Errorf("failed to validate dev user: %w", err)
	}

	// Save user
	userID, err := s.userRepo.Create(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to save dev user: %w", err)
	}

	s.logger.Info("Development user created successfully", zap.String("user_id", userID))
	return nil
}

// SeedDefaultTopics creates default topics for the dev user if they don't exist
func (s *DevSeeder) SeedDefaultTopics(ctx context.Context) error {
	s.logger.Info("Seeding default topics for development user...")

	seedTopics, err := s.loadTopicsFromFile()
	if err != nil {
		return fmt.Errorf("failed to load topics from seed: %w", err)
	}

	existingTopics, err := s.topicRepo.ListByUserID(ctx, DevUserID)
	if err != nil {
		s.logger.Warn("Failed to check existing topics", zap.Error(err))
		existingTopics = []*entities.Topic{}
	}

	existingByName := make(map[string]bool)
	for _, t := range existingTopics {
		existingByName[strings.ToLower(t.Name)] = true
	}

	for _, seedTopic := range seedTopics {
		normalizedName := strings.ToLower(seedTopic.Name)
		if existingByName[normalizedName] {
			s.logger.Info("Topic already exists, skipping", zap.String("topic", seedTopic.Name))
			continue
		}

		seedTopic.ID = primitive.NewObjectID().Hex()
		seedTopic.UserID = DevUserID
		seedTopic.NormalizeRelatedTopics()
		seedTopic.SetDefaults()
		if seedTopic.Prompt == "" {
			seedTopic.Prompt = entities.DefaultPrompt
		}

		if seedTopic.CreatedAt.IsZero() {
			seedTopic.CreatedAt = time.Now()
		}
		if seedTopic.UpdatedAt.IsZero() {
			seedTopic.UpdatedAt = seedTopic.CreatedAt
		}

		if err := seedTopic.Validate(); err != nil {
			s.logger.Warn("Failed to validate topic", zap.String("topic", seedTopic.Name), zap.Error(err))
			continue
		}

		topicID, err := s.topicRepo.Create(ctx, seedTopic)
		if err != nil {
			s.logger.Warn("Failed to save topic", zap.String("topic", seedTopic.Name), zap.Error(err))
			continue
		}

		s.logger.Info("Default topic created", zap.String("topic", seedTopic.Name), zap.String("topic_id", topicID))
	}

	s.logger.Info("Default topics seeding completed")
	return nil
}

// SeedInitialIdeas generates initial ideas for all dev user topics
func (s *DevSeeder) SeedInitialIdeas(ctx context.Context) error {
	s.logger.Info("Seeding initial ideas for development user...")

	// Check if ideas already exist
	existingCount, err := s.ideasRepo.CountByUserID(ctx, DevUserID)
	if err != nil {
		s.logger.Warn("Failed to check existing ideas count", zap.Error(err))
	} else if existingCount > 0 {
		s.logger.Info("Ideas already exist, skipping initial generation",
			zap.Int64("existing_count", existingCount))
		return nil
	}

	// Get all topics for the dev user
	topics, err := s.topicRepo.ListByUserID(ctx, DevUserID)
	if err != nil {
		return fmt.Errorf("failed to list topics: %w", err)
	}

	if len(topics) == 0 {
		s.logger.Warn("No topics found, skipping idea generation")
		return nil
	}

	generateIdeasUC := usecases.NewGenerateIdeasUseCase(
		s.userRepo,
		s.topicRepo,
		s.ideasRepo,
		s.promptsRepo,
		s.promptEngine,
		s.llmService,
	)

	totalGenerated := 0

	for _, topic := range topics {
		// Skip inactive topics
		if !topic.Active {
			s.logger.Info("Skipping inactive topic", zap.String("topic", topic.Name))
			continue
		}

		s.logger.Info("Generating ideas for topic",
			zap.String("topic", topic.Name),
			zap.Int("count", topic.Ideas))

		generated, err := generateIdeasUC.GenerateIdeasForTopic(ctx, topic.ID)
		if err != nil {
			s.logger.Warn("Failed to generate ideas for topic",
				zap.String("topic", topic.Name),
				zap.Error(err),
			)
			continue
		}

		totalGenerated += len(generated)
		s.logger.Info("Ideas generated successfully",
			zap.String("topic", topic.Name),
			zap.Int("count", len(generated)),
		)
	}

	s.logger.Info("Initial ideas seeding completed", zap.Int("total_generated", totalGenerated))
	return nil
}

// SeedDefaultPrompts creates default prompts for the dev user if they don't exist
func (s *DevSeeder) SeedDefaultPrompts(ctx context.Context) error {
	s.logger.Info("Seeding default prompts for development user...")

	// Create prompt loader using zap adapter for interfaces.Logger
	loggerAdapter := services.NewZapLoggerAdapter(s.logger)
	promptLoader := services.NewPromptLoader(loggerAdapter)

	// Load prompts from files
	promptFiles, err := promptLoader.LoadPromptsFromDir(s.promptDir)
	if err != nil {
		return fmt.Errorf("failed to load prompts from directory %s: %w", s.promptDir, err)
	}

	if len(promptFiles) == 0 {
		s.logger.Warn("No prompt files found in directory", zap.String("dir", s.promptDir))
		return nil
	}

	// Create prompt entities from files
	prompts, err := promptLoader.CreatePromptsFromFile(DevUserID, promptFiles)
	if err != nil {
		return fmt.Errorf("failed to create prompt entities: %w", err)
	}

	if len(prompts) == 0 {
		s.logger.Warn("No valid prompts were created from files")
		return nil
	}

	existingPrompts, err := s.promptsRepo.ListByUserID(ctx, DevUserID)
	if err != nil {
		s.logger.Warn("Failed to list existing prompts", zap.Error(err))
	}

	existingByName := make(map[string]bool)
	for _, p := range existingPrompts {
		existingByName[strings.ToLower(p.Name)] = true
	}

	// Save prompts to database
	for _, prompt := range prompts {
		if existingByName[strings.ToLower(prompt.Name)] {
			s.logger.Info("Prompt already exists, skipping", zap.String("name", prompt.Name))
			continue
		}

		promptID, err := s.promptsRepo.Create(ctx, prompt)
		if err != nil {
			s.logger.Warn("Failed to save prompt",
				zap.String("name", prompt.Name),
				zap.Error(err),
			)
			continue
		}

		s.logger.Info("Prompt created successfully",
			zap.String("name", prompt.Name),
			zap.String("type", string(prompt.Type)),
			zap.String("prompt_id", promptID),
		)
	}

	s.logger.Info("Default prompts seeding completed", zap.Int("total_created", len(prompts)))
	return nil
}

func (s *DevSeeder) loadTopicsFromFile() ([]*entities.Topic, error) {
	content, err := os.ReadFile(s.topicSeedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read topic seed file: %w", err)
	}

	var rawTopics []map[string]interface{}
	if err := json.Unmarshal(content, &rawTopics); err != nil {
		return nil, fmt.Errorf("failed to parse topic seed JSON: %w", err)
	}

	topics := make([]*entities.Topic, 0, len(rawTopics))
	for _, raw := range rawTopics {
		name, _ := raw["name"].(string)
		description, _ := raw["description"].(string)
		category, _ := raw["category"].(string)
		promptName, _ := raw["prompt"].(string)

		priority := entities.DefaultPriority
		if value, ok := raw["priority"].(float64); ok {
			priority = int(value)
		}

		ideas := entities.DefaultIdeasCount
		if value, ok := raw["ideas"].(float64); ok {
			ideas = int(value)
		}

		active := true
		if value, ok := raw["active"].(bool); ok {
			active = value
		}

		var relatedTopics []string
		if list, ok := raw["related_topics"].([]interface{}); ok {
			for _, item := range list {
				if str, ok := item.(string); ok {
					relatedTopics = append(relatedTopics, str)
				}
			}
		}

		topic := &entities.Topic{
			Name:          name,
			Description:   description,
			Category:      category,
			Priority:      priority,
			Ideas:         ideas,
			Prompt:        promptName,
			RelatedTopics: relatedTopics,
			Active:        active,
		}

		topics = append(topics, topic)
	}

	return topics, nil
}

// SeedAll seeds all development data
func (s *DevSeeder) SeedAll(ctx context.Context) error {
	if err := s.SeedDevUser(ctx); err != nil {
		return fmt.Errorf("failed to seed dev user: %w", err)
	}

	if err := s.SeedDefaultPrompts(ctx); err != nil {
		s.logger.Warn("Failed to seed default prompts", zap.Error(err))
		// Don't fail if prompts seeding fails
	}

	if err := s.SeedDefaultTopics(ctx); err != nil {
		return fmt.Errorf("failed to seed default topics: %w", err)
	}

	if err := s.SeedInitialIdeas(ctx); err != nil {
		s.logger.Warn("Failed to seed initial ideas", zap.Error(err))
		// Don't fail if ideas seeding fails
	}

	return nil
}
