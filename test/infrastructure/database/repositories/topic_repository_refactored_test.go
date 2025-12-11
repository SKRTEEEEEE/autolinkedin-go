package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/linkgen-ai/backend/src/domain/entities"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestTopicRepositoryRefactored(t *testing.T) {
	// Test for the new TopicRepository methods to handle the refactored structure:
	// - Topics now have: ideas, prompt, related_topics fields
	// - prompt field references a prompt by name
	// - Methods should handle the new structure correctly

	// Setup test database
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://localhost:27017"))
	require.NoError(t, err)
	defer client.Disconnect(context.Background())

	// Use a test database
	testDB := client.Database("test_topics_refactored")
	defer testDB.Drop(context.Background())

	// Create collection
	collection := testDB.Collection("topics")
	repo := NewTopicRepository(collection)

	// Clear collection before each test
	setup := func(t *testing.T) {
		_, err := collection.DeleteMany(context.Background(), bson.M{})
		require.NoError(t, err)
	}

	t.Run("should create topic with new fields (ideas, prompt, related_topics)", func(t *testing.T) {
		setup(t)

		// GIVEN a topic with all new fields
		userID := "user-123"
		topic := &entities.Topic{
			ID:            primitive.NewObjectID().Hex(),
			UserID:        userID,
			Name:          "Marketing Digital",
			Description:   "Contenido sobre estrategias de marketing digital",
			Category:      "Marketing",
			Priority:      7,
			Ideas:         3,           // NEW field
			Prompt:        "base1",      // NEW field
			RelatedTopics: []string{"SEO", "Social Media"}, // NEW field
			Active:        true,
			CreatedAt:     time.Now(),
		}

		// WHEN creating the topic
		id, err := repo.Create(context.Background(), topic)
		require.NoError(t, err)
		topic.ID = id

		// THEN the topic should be created correctly
		assert.NotEmpty(t, id)

		// AND it should be found with all fields preserved
		foundTopic, err := repo.FindByID(context.Background(), id)
		require.NoError(t, err)
		assert.NotNil(t, foundTopic)
		assert.Equal(t, topic.Ideas, foundTopic.Ideas)
		assert.Equal(t, topic.Prompt, foundTopic.Prompt)
		assert.Equal(t, topic.RelatedTopics, foundTopic.RelatedTopics)
	})

	t.Run("should find topics by prompt reference", func(t *testing.T) {
		setup(t)

		// GIVEN topics with different prompt references
		userID := "user-456"
		base1Topic := &entities.Topic{
			Name:          "Topic 1",
			UserID:        userID,
			Prompt:        "base1",
			Ideas:         2,
			Description:   "Description 1",
			Priority:      5,
			Active:        true,
			CreatedAt:     time.Now(),
		}

		creativeTopic := &entities.Topic{
			Name:          "Topic 2",
			UserID:        userID,
			Prompt:        "creative",
			Ideas:         5,
			Description:   "Description 2",
			Priority:      8,
			Active:        true,
			CreatedAt:     time.Now(),
		}

		base1Topic2 := &entities.Topic{
			Name:          "Topic 3",
			UserID:        userID,
			Prompt:        "base1",
			Ideas:         3,
			Description:   "Description 3",
			Priority:      6,
			Active:        true,
			CreatedAt:     time.Now(),
		}

		// Insert all topics
		_, err := repo.Create(context.Background(), base1Topic)
		require.NoError(t, err)
		_, err = repo.Create(context.Background(), creativeTopic)
		require.NoError(t, err)
		_, err = repo.Create(context.Background(), base1Topic2)
		require.NoError(t, err)

		// WHEN searching for topics using "base1" prompt
		base1Topics, err := repo.FindByPrompt(context.Background(), userID, "base1")

		// THEN the correct topics should be returned
		require.NoError(t, err)
		assert.Len(t, base1Topics, 2)
		for _, topic := range base1Topics {
			assert.Equal(t, "base1", topic.Prompt)
		}

		// AND when searching for "creative" prompt
		creativeTopics, err := repo.FindByPrompt(context.Background(), userID, "creative")
		require.NoError(t, err)
		assert.Len(t, creativeTopics, 1)
		assert.Equal(t, "creative", creativeTopics[0].Prompt)

		// AND when searching for non-existent prompt
		nonExistentTopics, err := repo.FindByPrompt(context.Background(), userID, "nonexistent")
		require.NoError(t, err)
		assert.Len(t, nonExistentTopics, 0)
	})

	t.Run("should find topics by ideas count within range", func(t *testing.T) {
		setup(t)

		// GIVEN topics with different ideas count
		userID := "user-789"
		topics := []*entities.Topic{
			{Name: "Topic 1", UserID: userID, Ideas: 1, Prompt: "base1", Description: "Desc 1", Priority: 5, Active: true, CreatedAt: time.Now()},
			{Name: "Topic 2", UserID: userID, Ideas: 3, Prompt: "base1", Description: "Desc 2", Priority: 5, Active: true, CreatedAt: time.Now()},
			{Name: "Topic 3", UserID: userID, Ideas: 5, Prompt: "base1", Description: "Desc 3", Priority: 5, Active: true, CreatedAt: time.Now()},
			{Name: "Topic 4", UserID: userID, Ideas: 10, Prompt: "base1", Description: "Desc 4", Priority: 5, Active: true, CreatedAt: time.Now()},
		}

		// Insert all topics
		for _, topic := range topics {
			_, err := repo.Create(context.Background(), topic)
			require.NoError(t, err)
		}

		// WHEN searching for topics with 1-3 ideas
		minMaxTopics, err := repo.FindByIdeasRange(context.Background(), userID, 1, 3)
		require.NoError(t, err)
		assert.Len(t, minMaxTopics, 2) // Topics with 1 and 3 ideas

		// WHEN searching for topics with 5-5 ideas (exact match)
		exactTopics, err := repo.FindByIdeasRange(context.Background(), userID, 5, 5)
		require.NoError(t, err)
		assert.Len(t, exactTopics, 1) // Only topic with exactly 5 ideas

		// WHEN searching for topics with 6-8 ideas (empty range)
		emptyTopics, err := repo.FindByIdeasRange(context.Background(), userID, 6, 8)
		require.NoError(t, err)
		assert.Len(t, emptyTopics, 0)
	})

	t.Run("should update topic with new fields", func(t *testing.T) {
		setup(t)

		// GIVEN a topic created with initial values
		userID := "user-999"
		topic := &entities.Topic{
			Name:        "Topic to Update",
			UserID:      userID,
			Prompt:      "base1",
			Ideas:       2,
			Description: "Original description",
			Priority:    5,
			Active:      true,
			CreatedAt:   time.Now(),
		}

		id, err := repo.Create(context.Background(), topic)
		require.NoError(t, err)
		topic.ID = id

		// WHEN updating the topic with new field values
		topic.Prompt = "creative"
		topic.Ideas = 5
		topic.RelatedTopics = []string{"SEO", "Content"}
		topic.Description = "Updated description"

		err = repo.Update(context.Background(), topic)
		require.NoError(t, err)

		// THEN the topic should be updated correctly
		updatedTopic, err := repo.FindByID(context.Background(), id)
		require.NoError(t, err)
		assert.Equal(t, "creative", updatedTopic.Prompt)
		assert.Equal(t, 5, updatedTopic.Ideas)
		assert.Equal(t, []string{"SEO", "Content"}, updatedTopic.RelatedTopics)
		assert.Equal(t, "Updated description", updatedTopic.Description)
	})

	t.Run("should handle related_topics field in CRUD operations", func(t *testing.T) {
		setup(t)

		// GIVEN a topic with related topics
		userID := "user-111"
		relatedTopics := []string{"Marketing", "SEO", "Analytics"}
		topic := &entities.Topic{
			Name:          "Marketing Strategy",
			UserID:        userID,
			Prompt:        "base1",
			Ideas:         3,
			RelatedTopics: relatedTopics,
			Description:   "Strategies for digital marketing",
			Priority:      8,
			Active:        true,
			CreatedAt:     time.Now(),
		}

		// WHEN creating the topic
		id, err := repo.Create(context.Background(), topic)
		require.NoError(t, err)
		topic.ID = id

		// THEN the related topics should be preserved
		foundTopic, err := repo.FindByID(context.Background(), id)
		require.NoError(t, err)
		assert.Equal(t, relatedTopics, foundTopic.RelatedTopics)

		// AND when updating the related topics
		newRelatedTopics := []string{"Marketing", "Social Media", "Content Strategy"}
		foundTopic.RelatedTopics = newRelatedTopics

		err = repo.Update(context.Background(), foundTopic)
		require.NoError(t, err)

		updatedTopic, err := repo.FindByID(context.Background(), id)
		require.NoError(t, err)
		assert.Equal(t, newRelatedTopics, updatedTopic.RelatedTopics)
	})

	t.Run("should find topics with multiple advanced filters", func(t *testing.T) {
		setup(t)

		// GIVEN topics with various combinations of the new fields
		userID := "user-222"
		topics := []*entities.Topic{
			{
				Name:          "Active Marketing",
				UserID:        userID,
				Prompt:        "base1",
				Ideas:         3,
				RelatedTopics: []string{"SEO", "Social Media"},
				Category:      "Marketing",
				Active:        true,
				CreatedAt:     time.Now(),
			},
			{
				Name:          "Inactive Marketing",
				UserID:        userID,
				Prompt:        "base1",
				Ideas:         5,
				RelatedTopics: []string{"Content"},
				Category:      "Marketing",
				Active:        false,
				CreatedAt:     time.Now(),
			},
			{
				Name:          "Active Technology",
				UserID:        userID,
				Prompt:        "creative",
				Ideas:         3,
				RelatedTopics: []string{"AI", "ML"},
				Category:      "Technology",
				Active:        true,
				CreatedAt:     time.Now(),
			},
		}

		// Insert all topics
		for _, topic := range topics {
			_, err := repo.Create(context.Background(), topic)
			require.NoError(t, err)
		}

		// WHEN searching with multiple filters (active, base1 prompt, 3-5 ideas)
		filter := TopicFilter{
			UserID: userID,
			Active: boolPtr(true),
			Prompt: stringPtr("base1"),
			IdeasMin: intPtr(3),
			IdeasMax: intPtr(5),
		}

		filteredTopics, err := repo.FindWithFilters(context.Background(), filter)
		require.NoError(t, err)
		assert.Len(t, filteredTopics, 1) // Only "Active Marketing" matches all criteria
		assert.Equal(t, "Active Marketing", filteredTopics[0].Name)
		assert.Equal(t, "base1", filteredTopics[0].Prompt)
		assert.Equal(t, 3, filteredTopics[0].Ideas)
	})

	t.Run("should validate prompt reference exists when creating topic", func(t *testing.T) {
		setup(t)

		// GIVEN a topic referencing a non-existent prompt
		userID := "user-333"
		topic := &entities.Topic{
			Name:        "Topic with Bad Reference",
			UserID:      userID,
			Prompt:      "nonexistent-prompt", // This prompt doesn't exist
			Ideas:       2,
			Description: "Description",
			Priority:    5,
			Active:      true,
			CreatedAt:   time.Now(),
		}

		// WHEN creating the topic
		// THEN it should succeed at repository level (validation handled elsewhere)
		id, err := repo.Create(context.Background(), topic)
		require.NoError(t, err)
		assert.NotEmpty(t, id)

		// AND the topic should be created with the reference
		foundTopic, err := repo.FindByID(context.Background(), id)
		require.NoError(t, err)
		assert.Equal(t, "nonexistent-prompt", foundTopic.Prompt)
	})

	t.Run("should maintain backward compatibility with existing methods", func(t *testing.T) {
		setup(t)

		// GIVEN a topic created with the new structure
		userID := "user-444"
		topic := &entities.Topic{
			ID:             primitive.NewObjectID().Hex(),
			UserID:         userID,
			Name:           "Compatibility Test",
			Description:    "Testing backward compatibility",
			Category:       "Test",
			Priority:       7,
			Ideas:          4,
			Prompt:         "base1",
			RelatedTopics:  []string{"Test1", "Test2"},
			Active:         true,
			CreatedAt:      time.Now(),
		}

		// WHEN using existing repository methods
		// THEN they should work with the new structure

		// Create
		id, err := repo.Create(context.Background(), topic)
		require.NoError(t, err)

		// FindByID
		foundByID, err := repo.FindByID(context.Background(), id)
		require.NoError(t, err)
		assert.NotNil(t, foundByID)

		// ListByUserID
		userTopics, err := repo.ListByUserID(context.Background(), userID)
		require.NoError(t, err)
		assert.Len(t, userTopics, 1)
		assert.Equal(t, topic.Prompt, userTopics[0].Prompt)
		assert.Equal(t, topic.Ideas, userTopics[0].Ideas)
		assert.Equal(t, topic.RelatedTopics, userTopics[0].RelatedTopics)

		// CountByUserID
		count, err := repo.CountByUserID(context.Background(), userID)
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)

		// Delete
		err = repo.Delete(context.Background(), id)
		require.NoError(t, err)

		// Verify deletion
		deletedTopic, err := repo.FindByID(context.Background(), id)
		require.NoError(t, err)
		assert.Nil(t, deletedTopic)
	})
}

// Helper function to create a string pointer
func stringPtr(s string) *string {
	return &s
}

// Helper function to create a bool pointer
func boolPtr(b bool) *bool {
	return &b
}

// Helper function to create an int pointer
func intPtr(i int) *int {
	return &i
}

// Filter structure for advanced topic searching
type TopicFilter struct {
	UserID    string
	Name      *string
	Category  *string
	Active    *bool
	Prompt    *string
	IdeasMin  *int
	IdeasMax  *int
}

// NEW methods to be added to TopicRepository

// FindByPrompt finds topics by prompt reference
func (r *TopicRepository) FindByPrompt(ctx context.Context, userID string, prompt string) ([]*entities.Topic, error) {
	filter := bson.M{
		"user_id": userID,
		"prompt":  prompt,
	}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find topics by prompt: %w", err)
	}
	defer cursor.Close(ctx)

	var topics []*entities.Topic
	for cursor.Next(ctx) {
		var doc topicDocument
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("failed to decode topic: %w", err)
		}
		topics = append(topics, r.toEntity(&doc))
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return topics, nil
}

// FindByIdeasRange finds topics with ideas count within specified range
func (r *TopicRepository) FindByIdeasRange(ctx context.Context, userID string, minIdeas, maxIdeas int) ([]*entities.Topic, error) {
	filter := bson.M{
		"user_id": userID,
		"ideas": bson.M{
			"$gte": minIdeas,
			"$lte": maxIdeas,
		},
	}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find topics by ideas range: %w", err)
	}
	defer cursor.Close(ctx)

	var topics []*entities.Topic
	for cursor.Next(ctx) {
		var doc topicDocument
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("failed to decode topic: %w", err)
		}
		topics = append(topics, r.toEntity(&doc))
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return topics, nil
}

// FindWithFilters finds topics using multiple filters
func (r *TopicRepository) FindWithFilters(ctx context.Context, filter TopicFilter) ([]*entities.Topic, error) {
	query := bson.M{"user_id": filter.UserID}

	if filter.Name != nil {
		query["name"] = *filter.Name
	}

	if filter.Category != nil {
		query["category"] = *filter.Category
	}

	if filter.Active != nil {
		query["active"] = *filter.Active
	}

	if filter.Prompt != nil {
		query["prompt"] = *filter.Prompt
	}

	if filter.IdeasMin != nil || filter.IdeasMax != nil {
		ideasRange := bson.M{}
		if filter.IdeasMin != nil {
			ideasRange["$gte"] = *filter.IdeasMin
		}
		if filter.IdeasMax != nil {
			ideasRange["$lte"] = *filter.IdeasMax
		}
		query["ideas"] = ideasRange
	}

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, query, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find topics with filters: %w", err)
	}
	defer cursor.Close(ctx)

	var topics []*entities.Topic
	for cursor.Next(ctx) {
		var doc topicDocument
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("failed to decode topic: %w", err)
		}
		topics = append(topics, r.toEntity(&doc))
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return topics, nil
}
