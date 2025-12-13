package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/linkgen-ai/backend/src/domain/entities"
	"github.com/linkgen-ai/backend/src/infrastructure/database/repositories"
)

// TestProductionDataCompatibility tests that the refactored system
// maintains compatibility with production data structures
func TestProductionDataCompatibility(tt *testing.T) {
	ctx := context.Background()

	tt.Run("should handle existing topics without prompt references", func(t *testing.T) {
		// GIVEN existing production topics (without prompt field)
		db := setupTestDB(t)
		defer cleanupTestDB(t, db)

		userID := primitive.NewObjectID().Hex()
		topicRepo := repositories.NewTopicRepository(db)

		// Simulate legacy topic structure (old production data)
		legacyTopic := &entities.Topic{
			ID:          primitive.NewObjectID().Hex(),
			UserID:      userID,
			Name:        "Python Basics",
			Description: "Introduction to Python programming",
			Keywords:    []string{"python", "programming", "basics"},
			Category:    "",
			Priority:    5,
			IdeasCount:  0, // Legacy field
			Active:      true,
			// PromptName is empty (not in old structure)
			RelatedTopics: []string{},
			CreatedAt:     time.Now().Add(-30 * 24 * time.Hour), // 30 days ago
			UpdatedAt:     time.Now().Add(-30 * 24 * time.Hour),
		}

		// Simulate inserting legacy data directly into database
		collection := db.Collection("topics")
		legacyData := map[string]interface{}{
			"_id":          legacyTopic.ID,
			"user_id":      legacyTopic.UserID,
			"name":         legacyTopic.Name,
			"description":  legacyTopic.Description,
			"keywords":     legacyTopic.Keywords,
			"category":     legacyTopic.Category,
			"priority":     legacyTopic.Priority,
			"ideas":        legacyTopic.IdeasCount, // Legacy field name
			"active":       legacyTopic.Active,
			"related_topics": legacyTopic.RelatedTopics,
			"created_at":   legacyTopic.CreatedAt,
			"updated_at":   legacyTopic.UpdatedAt,
			// No prompt_name field
		}

		_, err := collection.InsertOne(ctx, legacyData)
		require.NoError(t, err)

		// WHEN fetching topic through new repository
		topic, err := topicRepo.GetByID(ctx, legacyTopic.ID)
		
		// This will fail until backward compatibility is implemented
		t.Fatal("implement backward compatibility for legacy topics - FAILING IN TDD RED PHASE")
		
		// THEN should:
		// 1. Handle missing prompt_name gracefully
		// 2. Map old 'ideas' field to new 'ideas_count' 
		// 3. Provide default prompt reference
		require.NoError(t, err)
		assert.NotNil(t, topic)
		assert.Equal(t, legacyTopic.Name, topic.Name)
		assert.Equal(t, "base1", topic.PromptName, "Should use default prompt for legacy topics")
		assert.Equal(t, 5, topic.IdeasCount, "Should map legacy ideas field")
	})

	tt.Run("should handle existing ideas without topic_name field", func(t *testing.T) {
		// GIVEN existing production ideas (without topic_name)
		db := setupTestDB(t)
		defer cleanupTestDB(t, db)

		userID := primitive.NewObjectID().Hex()
		ideaRepo := repositories.NewIdeaRepository(db)
		topicRepo := repositories.NewTopicRepository(db)

		// Create topic for reference
		topic := &entities.Topic{
			ID:          primitive.NewObjectID().Hex(),
			UserID:      userID,
			Name:        "Docker Containerization",
			Description: "Learning Docker fundamentals",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		createdTopic, err := topicRepo.Create(ctx, topic)
		require.NoError(t, err)

		// Simulate legacy idea structure
		legacyIdeaData := map[string]interface{}{
			"_id":      primitive.NewObjectID().Hex(),
			"user_id":  userID,
			"topic_id": createdTopic.ID,
			"content":  "Best practices for Dockerfile optimization",
			"used":     false,
			// No topic_name field in legacy data
			"created_at": time.Now().Add(-15 * 24 * time.Hour),
			"updated_at": time.Now().Add(-15 * 24 * time.Hour),
		}

		// Insert legacy idea directly
		ideaCollection := db.Collection("ideas")
		_, err = ideaCollection.InsertOne(ctx, legacyIdeaData)
		require.NoError(t, err)

		// WHEN fetching idea through new repository
		ideas, err := ideaRepo.ListByUserID(ctx, userID)
		
		// This will fail until idea backward compatibility is implemented
		t.Fatal("implement backward compatibility for legacy ideas - FAILING IN TDD RED PHASE")
		
		// THEN should:
		// 1. Populate topic_name from topic relationship
		// 2. Handle missing fields gracefully
		require.NoError(t, err)
		require.Len(t, ideas, 1)
		
		idea := ideas[0]
		assert.Equal(t, "Docker Containerization", idea.TopicName)
		assert.Equal(t, createdTopic.ID, idea.TopicID)
	})

	tt.Run("should handle existing prompts with style_name instead of name", func(t *testing.T) {
		// GIVEN existing production prompts (using style_name field)
		db := setupTestDB(t)
		defer cleanupTestDB(t, db)

		userID := primitive.NewObjectID().Hex()

		// Simulate legacy prompt structure
		legacyPromptData := map[string]interface{}{
			"_id":            primitive.NewObjectID().Hex(),
			"user_id":        userID,
			"style_name":     "creative", // Old field name
			"type":           "ideas",
			"prompt_template": "Generate creative ideas about {name}",
			"active":         true,
			"created_at":     time.Now().Add(-45 * 24 * time.Hour),
			"updated_at":     time.Now().Add(-45 * 24 * time.Hour),
			// No name field
		}

		// Insert legacy prompt directly
		promptCollection := db.Collection("prompts")
		_, err := promptCollection.InsertOne(ctx, legacyPromptData)
		require.NoError(t, err)

		// WHEN fetching prompts through new repository
		promptRepo := repositories.NewPromptRepository(db)
		prompts, err := promptRepo.ListByUserID(ctx, userID)
		
		// This will fail until prompt backward compatibility is implemented
		t.Fatal("implement backward compatibility for legacy prompts (style_name) - FAILING IN TDD RED PHASE")
		
		// THEN should:
		// 1. Map style_name to name field
		// 2. Handle missing new fields gracefully
		require.NoError(t, err)
		require.Len(t, prompts, 1)
		
		prompt := prompts[0]
		assert.Equal(t, "creative", prompt.Name, "Should map style_name to name")
		assert.Equal(t, entities.PromptTypeIdeas, prompt.Type)
	})

	tt.Run("should maintain production data integrity during migration", func(t *testing.T) {
		// GIVEN a mixed dataset of legacy and new structures
		db := setupTestDB(t)
		defer cleanupTestDB(t, db)

		userID := primitive.NewObjectID().Hex()

		// Create mixed legacy and new entities
		legacyTopicData := map[string]interface{}{
			"_id":       primitive.NewObjectID().Hex(),
			"user_id":   userID,
			"name":      "Legacy Topic",
			"ideas":     3, // Legacy field
			"active":    true,
			"created_at": time.Now().Add(-20 * 24 * time.Hour),
		}

		newTopicData := map[string]interface{}{
			"_id":          primitive.NewObjectID().Hex(),
			"user_id":      userID,
			"name":         "New Topic",
			"prompt_name":  "custom",
			"ideas_count":  5, // New field name
			"active":       true,
			"created_at":   time.Now(),
		}

		// Insert both types
		topicCollection := db.Collection("topics")
		_, err := topicCollection.InsertOne(ctx, legacyTopicData)
		require.NoError(t, err)
		_, err = topicCollection.InsertOne(ctx, newTopicData)
		require.NoError(t, err)

		// WHEN running migration or accessing data
		topicRepo := repositories.NewTopicRepository(db)
		topics, err := topicRepo.ListByUserID(ctx, userID)
		
		// This will fail until data migration/compatibility is implemented
		t.Fatal("implement production data migration logic - FAILING IN TDD RED PHASE")
		
		// THEN should:
		// 1. Successfully load both legacy and new topics
		// 2. Maintain data integrity
		// 3. Provide sensible defaults for missing fields
		require.NoError(t, err)
		require.Len(t, topics, 2)
		
		// Verify both are accessible
		hasLegacy := false
		hasNew := false
		for _, topic := range topics {
			if topic.Name == "Legacy Topic" {
				hasLegacy = true
				assert.Equal(t, 3, topic.IdeasCount)
				assert.Equal(t, "base1", topic.PromptName) // Default
			}
			if topic.Name == "New Topic" {
				hasNew = true
				assert.Equal(t, 5, topic.IdeasCount)
				assert.Equal(t, "custom", topic.PromptName)
			}
		}
		assert.True(t, hasLegacy, "Should include legacy topic")
		assert.True(t, hasNew, "Should include new topic")
	})
}

// TestProductionDataValidation validates production data against new constraints
func TestProductionDataValidation(tt *testing.T) {
	ctx := context.Background()

	tt.Run("should validate idea content length changes (10-5000 to 10-200)", func(t *testing.T) {
		// GIVEN production ideas with varied content lengths
		db := setupTestDB(t)
		defer cleanupTestDB(t, db)

		userID := primitive.NewObjectID().Hex()
		
		// Create ideas with different content lengths
		testIdeas := []map[string]interface{}{
			{
				"_id":      primitive.NewObjectID().Hex(),
				"user_id":  userID,
				"content":  "Short", // Valid (>=10)
				"topic_id": primitive.NewObjectID().Hex(),
				"created_at": time.Now(),
			},
			{
				"_id":      primitive.NewObjectID().Hex(),
				"user_id":  userID,
				"content": string(make([]byte, 150)), // Valid (<200)
				"topic_id": primitive.NewObjectID().Hex(),
				"created_at": time.Now(),
			},
			{
				"_id":      primitive.NewObjectID().Hex(),
				"user_id":  userID,
				"content":  "Too short", // Too short (<10)
				"topic_id": primitive.NewObjectID().Hex(),
				"created_at": time.Now(),
			},
			{
				"_id":      primitive.NewObjectID().Hex(),
				"user_id":  userID,
				"content": string(make([]byte, 300)), // Too long (>200)
				"topic_id": primitive.NewObjectID().Hex(),
				"created_at": time.Now(),
			},
		}

		ideaCollection := db.Collection("ideas")
		for _, idea := range testIdeas {
			_, err := ideaCollection.InsertOne(ctx, idea)
			require.NoError(t, err)
		}

		// WHEN validating production ideas against new constraints
		// This will fail until validation is implemented
		t.Fatal("implement production data validation for new content length constraints - FAILING IN TDD RED PHASE")
		
		// THEN should:
		// 1. Identify ideas that violate new constraints
		// 2. Provide recommendations for data cleanup
		// 3. Allow operation with warnings for invalid data
	})
}
