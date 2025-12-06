package services

import (
	"context"
	"fmt"
	"time"

	"github.com/linkgen-ai/backend/src/domain/entities"
	"github.com/linkgen-ai/backend/src/domain/interfaces"
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
	userRepo   interfaces.UserRepository
	topicRepo  interfaces.TopicRepository
	ideasRepo  interfaces.IdeasRepository
	llmService interfaces.LLMService
	logger     *zap.Logger
}

// NewDevSeeder creates a new development data seeder
func NewDevSeeder(
	userRepo interfaces.UserRepository,
	topicRepo interfaces.TopicRepository,
	ideasRepo interfaces.IdeasRepository,
	llmService interfaces.LLMService,
	logger *zap.Logger,
) *DevSeeder {
	return &DevSeeder{
		userRepo:   userRepo,
		topicRepo:  topicRepo,
		ideasRepo:  ideasRepo,
		llmService: llmService,
		logger:     logger,
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
			name:        "Technology",
			description: "Software development, AI, cloud computing, and tech innovations",
		},
		{
			name:        "Productivity",
			description: "Time management, work efficiency, and productivity hacks",
		},
		{
			name:        "Leadership",
			description: "Team management, leadership skills, and organizational culture",
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
			ID:          primitive.NewObjectID().Hex(),
			UserID:      DevUserID,
			Name:        topicData.name,
			Description: topicData.description,
			Keywords:    []string{},
			Category:    "General",
			Priority:    5,
			Active:      true,
			CreatedAt:   time.Now(),
		}

		// Validate topic
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

// generateMockIdeas creates mock ideas for development when LLM is unavailable
func (s *DevSeeder) generateMockIdeas(topicName string, count int) []string {
	mockIdeas := map[string][]string{
		"Technology": {
			"The future of AI-driven development: How machine learning is transforming software engineering workflows",
			"Building scalable microservices: Best practices for cloud-native architecture in 2025",
			"Cybersecurity trends: Protecting your applications from emerging threats",
			"The rise of serverless computing: When to use and when to avoid",
			"Developer productivity tools that changed the game this year",
		},
		"Productivity": {
			"Time-blocking techniques that actually work for software developers",
			"The Pomodoro Technique 2.0: Modern adaptations for remote teams",
			"Eliminating context switching: Strategies for deep work in a distracted world",
			"Automating your daily routine: Tools and workflows that save hours per week",
			"The science of focused work: How to maintain concentration for 4+ hours",
		},
		"Leadership": {
			"Leading remote engineering teams: Lessons learned from 100+ distributed projects",
			"The art of giving technical feedback: Building a culture of continuous improvement",
			"Hiring for culture add, not culture fit: Diversifying your tech team",
			"Conflict resolution in agile teams: Turning disagreements into innovations",
			"Mentoring junior developers: A framework for accelerating career growth",
		},
	}

	ideas, exists := mockIdeas[topicName]
	if !exists {
		// Generic fallback ideas
		ideas = []string{
			fmt.Sprintf("Exploring advanced concepts in %s", topicName),
			fmt.Sprintf("Best practices every professional should know about %s", topicName),
			fmt.Sprintf("Common mistakes to avoid when working with %s", topicName),
			fmt.Sprintf("How %s is evolving in the modern workplace", topicName),
			fmt.Sprintf("Building expertise in %s: A practical guide", topicName),
		}
	}

	if count > len(ideas) {
		count = len(ideas)
	}

	return ideas[:count]
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

	// Generate ideas for each topic
	const ideasPerTopic = 5
	totalGenerated := 0

	for _, topic := range topics {
		// Skip inactive topics
		if !topic.Active {
			s.logger.Info("Skipping inactive topic", zap.String("topic", topic.Name))
			continue
		}

		s.logger.Info("Generating ideas for topic", 
			zap.String("topic", topic.Name),
			zap.Int("count", ideasPerTopic))

		// Call LLM to generate ideas
		ideaContents, err := s.llmService.GenerateIdeas(ctx, topic.Name, ideasPerTopic)
		if err != nil {
			s.logger.Warn("Failed to generate ideas for topic, using fallback mock ideas",
				zap.String("topic", topic.Name),
				zap.Error(err),
			)
			// Fallback: Use mock ideas for development
			ideaContents = s.generateMockIdeas(topic.Name, ideasPerTopic)
		}

		if len(ideaContents) == 0 {
			s.logger.Warn("No ideas generated for topic", zap.String("topic", topic.Name))
			continue
		}

		// Create idea entities
		ideas := make([]*entities.Idea, 0, len(ideaContents))
		for _, content := range ideaContents {
			idea := &entities.Idea{
				ID:           primitive.NewObjectID().Hex(),
				UserID:       DevUserID,
				TopicID:      topic.ID,
				Content:      content,
				QualityScore: nil,
				Used:         false,
				CreatedAt:    time.Now(),
			}

			// Set expiration (30 days default)
			idea.CalculateExpiration(30)

			// Validate idea
			if err := idea.Validate(); err != nil {
				s.logger.Warn("Failed to validate idea, skipping",
					zap.String("topic", topic.Name),
					zap.Error(err),
				)
				continue
			}

			ideas = append(ideas, idea)
		}

		if len(ideas) == 0 {
			s.logger.Warn("No valid ideas created for topic", zap.String("topic", topic.Name))
			continue
		}

		// Save ideas batch
		if err := s.ideasRepo.CreateBatch(ctx, ideas); err != nil {
			s.logger.Warn("Failed to save ideas batch",
				zap.String("topic", topic.Name),
				zap.Error(err),
			)
			continue
		}

		totalGenerated += len(ideas)
		s.logger.Info("Ideas generated successfully",
			zap.String("topic", topic.Name),
			zap.Int("count", len(ideas)),
		)
	}

	s.logger.Info("Initial ideas seeding completed", zap.Int("total_generated", totalGenerated))
	return nil
}

// SeedAll seeds all development data
func (s *DevSeeder) SeedAll(ctx context.Context) error {
	if err := s.SeedDevUser(ctx); err != nil {
		return fmt.Errorf("failed to seed dev user: %w", err)
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
