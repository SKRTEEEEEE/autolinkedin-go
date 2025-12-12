package services

import (
	"context"
	"fmt"
	"path/filepath"
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
	userRepo    interfaces.UserRepository
	topicRepo   interfaces.TopicRepository
	ideasRepo   interfaces.IdeasRepository
	promptsRepo interfaces.PromptsRepository
	llmService  interfaces.LLMService
	logger      *zap.Logger
	promptDir   string
}

// NewDevSeeder creates a new development data seeder
func NewDevSeeder(
	userRepo interfaces.UserRepository,
	topicRepo interfaces.TopicRepository,
	ideasRepo interfaces.IdeasRepository,
	promptsRepo interfaces.PromptsRepository,
	llmService interfaces.LLMService,
	logger *zap.Logger,
) *DevSeeder {
	return &DevSeeder{
		userRepo:    userRepo,
		topicRepo:   topicRepo,
		ideasRepo:   ideasRepo,
		promptsRepo: promptsRepo,
		llmService:  llmService,
		logger:      logger,
		promptDir:   filepath.Join(".", "seed", "prompt"),
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

	defaultTopics := []struct {
		name        string
		description string
	}{
		{
			name:        "Inteligencia Artificial",
			description: "Machine Learning, Deep Learning, aplicaciones de IA y tendencias tecnológicas",
		},
		{
			name:        "Desarrollo Backend",
			description: "Arquitecturas de servidor, APIs, bases de datos y mejores prácticas de backend",
		},
		{
			name:        "TypeScript",
			description: "Desarrollo con TypeScript, frameworks modernos y patrones de diseño",
		},
	}

	for _, topicData := range defaultTopics {
		// Check if topic already exists for this user
		topics, err := s.topicRepo.ListByUserID(ctx, DevUserID)
		if err != nil {
			s.logger.Warn("Failed to check existing topics", zap.Error(err))
		}

		// Check if this topic name already exists
		topicExists := false
		for _, t := range topics {
			if t.Name == topicData.name {
				topicExists = true
				break
			}
		}

		if topicExists {
			s.logger.Info("Topic already exists, skipping", zap.String("topic", topicData.name))
			continue
		}

		// Create new topic
		topic := &entities.Topic{
			ID:            primitive.NewObjectID().Hex(),
			UserID:        DevUserID,
			Name:          topicData.name,
			Description:   topicData.description,
			Category:      entities.DefaultCategory,
			Priority:      entities.DefaultPriority,
			Ideas:         entities.DefaultIdeasCount,
			Prompt:        entities.DefaultPrompt,
			RelatedTopics: []string{},
			Active:        true,
			CreatedAt:     time.Now(),
		}

		// Set defaults and validate
		topic.SetDefaults()
		if err := topic.Validate(); err != nil {
			s.logger.Warn("Failed to validate topic",
				zap.String("topic", topicData.name),
				zap.Error(err),
			)
			continue
		}

		// Save topic
		topicID, err := s.topicRepo.Create(ctx, topic)
		if err != nil {
			s.logger.Warn("Failed to save topic",
				zap.String("topic", topicData.name),
				zap.Error(err),
			)
			continue
		}

		s.logger.Info("Default topic created",
			zap.String("topic", topicData.name),
			zap.String("topic_id", topicID),
		)
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

	// Check if prompts already exist
	existingCount, err := s.promptsRepo.CountByUserID(ctx, DevUserID)
	if err != nil {
		s.logger.Warn("Failed to check existing prompts count", zap.Error(err))
	} else if existingCount > 0 {
		s.logger.Info("Prompts already exist, skipping default generation",
			zap.Int64("existing_count", existingCount))
		return nil
	}

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

	// Save prompts to database
	for _, prompt := range prompts {
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
