package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/linkgen-ai/backend/src/domain/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestPromptsRepositoryRefactored(t *testing.T) {
	// Test for the new PromptRepository methods to handle the refactored structure:
	// - FindByName: Find prompts by name (new field used as identifier)
	// - Other methods should work with the new entity structure

	// Setup test database
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://localhost:27017"))
	require.NoError(t, err)
	defer client.Disconnect(context.Background())

	// Use a test database
	testDB := client.Database("test_prompts_refactored")
	defer testDB.Drop(context.Background())

	// Create collection
	collection := testDB.Collection("prompts")
	repo := NewPromptsRepository(collection)

	// Clear collection before each test
	setup := func(t *testing.T) {
		_, err := collection.DeleteMany(context.Background(), bson.M{})
		require.NoError(t, err)
	}

	t.Run("should find prompt by name successfully", func(t *testing.T) {
		setup(t)

		// GIVEN a prompt in the database with a name
		userID := "user-123"
		promptName := "base1"
		expectedPrompt := &entities.Prompt{
			ID:             primitive.NewObjectID().Hex(),
			UserID:         userID,
			Name:           promptName, // NEW field used as identifier
			Type:           entities.PromptTypeIdeas,
			PromptTemplate: "Genera {ideas} ideas sobre el tema: {name}",
			Active:         true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		// Insert the prompt
		id, err := repo.Create(context.Background(), expectedPrompt)
		require.NoError(t, err)
		expectedPrompt.ID = id

		// WHEN finding the prompt by name and user
		foundPrompt, err := repo.FindByName(context.Background(), userID, promptName)

		// THEN the prompt should be found
		require.NoError(t, err)
		assert.NotNil(t, foundPrompt)
		assert.Equal(t, expectedPrompt.ID, foundPrompt.ID)
		assert.Equal(t, expectedPrompt.Name, foundPrompt.Name)
		assert.Equal(t, expectedPrompt.Type, foundPrompt.Type)
		assert.Equal(t, expectedPrompt.PromptTemplate, foundPrompt.PromptTemplate)
		assert.Equal(t, expectedPrompt.Active, foundPrompt.Active)
	})

	t.Run("should return nil when prompt not found by name", func(t *testing.T) {
		setup(t)

		// GIVEN no prompt with the specified name exists
		userID := "user-456"
		nonExistentPromptName := "nonexistent"

		// WHEN searching for the prompt
		foundPrompt, err := repo.FindByName(context.Background(), userID, nonExistentPromptName)

		// THEN no prompt should be found
		require.NoError(t, err)
		assert.Nil(t, foundPrompt)
	})

	t.Run("should not find prompt by name for different user", func(t *testing.T) {
		setup(t)

		// GIVEN a prompt belonging to one user
		userID1 := "user-789"
		userID2 := "user-999"
		promptName := "base1"

		prompt := &entities.Prompt{
			Name:           promptName,
			UserID:         userID1,
			Type:           entities.PromptTypeIdeas,
			PromptTemplate: "Template for user 1",
			Active:         true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		// Insert the prompt
		_, err := repo.Create(context.Background(), prompt)
		require.NoError(t, err)

		// WHEN another user searches for the same prompt name
		foundPrompt, err := repo.FindByName(context.Background(), userID2, promptName)

		// THEN the prompt should not be found
		require.NoError(t, err)
		assert.Nil(t, foundPrompt)
	})

	t.Run("should find multiple prompts when name exists for different users", func(t *testing.T) {
		setup(t)

		// GIVEN prompts with the same name but different users
		promptName := "professional"
		userID1 := "user-111"
		userID2 := "user-222"

		prompt1 := &entities.Prompt{
			Name:           promptName,
			UserID:         userID1,
			Type:           entities.PromptTypeDrafts,
			PromptTemplate: "Template for user 1",
			Active:         true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		prompt2 := &entities.Prompt{
			Name:           promptName,
			UserID:         userID2,
			Type:           entities.PromptTypeDrafts,
			PromptTemplate: "Template for user 2",
			Active:         true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		// Insert both prompts
		_, err := repo.Create(context.Background(), prompt1)
		require.NoError(t, err)
		_, err = repo.Create(context.Background(), prompt2)
		require.NoError(t, err)

		// WHEN searching for each user's prompt
		foundPrompt1, err := repo.FindByName(context.Background(), userID1, promptName)
		require.NoError(t, err)
		assert.NotNil(t, foundPrompt1)
		assert.Equal(t, userID1, foundPrompt1.UserID)
		assert.Equal(t, prompt1.PromptTemplate, foundPrompt1.PromptTemplate)

		foundPrompt2, err := repo.FindByName(context.Background(), userID2, promptName)
		require.NoError(t, err)
		assert.NotNil(t, foundPrompt2)
		assert.Equal(t, userID2, foundPrompt2.UserID)
		assert.Equal(t, prompt2.PromptTemplate, foundPrompt2.PromptTemplate)

		// AND they should have different IDs
		assert.NotEqual(t, foundPrompt1.ID, foundPrompt2.ID)
	})

	t.Run("should handle name field in all existing operations", func(t *testing.T) {
		setup(t)

		// GIVEN prompts with the new entity structure
		userID := "user-333"
		prompt := &entities.Prompt{
			ID:             primitive.NewObjectID().Hex(),
			UserID:         userID,
			Name:           "creative",
			Type:           entities.PromptTypeIdeas,
			PromptTemplate: "Creative template with {ideas} placeholders",
			Active:         true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		// WHEN using existing repository methods
		// THEN they should work with the new structure

		// Create
		id, err := repo.Create(context.Background(), prompt)
		require.NoError(t, err)
		prompt.ID = id

		// FindByID
		foundByID, err := repo.FindByID(context.Background(), id)
		require.NoError(t, err)
		assert.NotNil(t, foundByID)
		assert.Equal(t, prompt.Name, foundByID.Name)

		// ListByUserID
		userPrompts, err := repo.ListByUserID(context.Background(), userID)
		require.NoError(t, err)
		assert.Len(t, userPrompts, 1)
		assert.Equal(t, prompt.Name, userPrompts[0].Name)

		// ListByUserIDAndType
		ideasPrompts, err := repo.ListByUserIDAndType(context.Background(), userID, entities.PromptTypeIdeas)
		require.NoError(t, err)
		assert.Len(t, ideasPrompts, 1)
		assert.Equal(t, prompt.Name, ideasPrompts[0].Name)

		// Update
		prompt.Name = "updated_name"
		prompt.PromptTemplate = "Updated template"
		err = repo.Update(context.Background(), prompt)
		require.NoError(t, err)

		// Verify update
		updatedPrompt, err := repo.FindByID(context.Background(), id)
		require.NoError(t, err)
		assert.Equal(t, "updated_name", updatedPrompt.Name)
		assert.Equal(t, "Updated template", updatedPrompt.PromptTemplate)
	})

	t.Run("should find active prompts by name", func(t *testing.T) {
		setup(t)

		// GIVEN an active and an inactive prompt with the same name
		userID := "user-444"
		promptName := "base1"

		activePrompt := &entities.Prompt{
			Name:           promptName,
			UserID:         userID,
			Type:           entities.PromptTypeIdeas,
			PromptTemplate: "Active template",
			Active:         true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		inactivePrompt := &entities.Prompt{
			Name:           promptName,
			UserID:         userID,
			Type:           entities.PromptTypeIdeas,
			PromptTemplate: "Inactive template",
			Active:         false,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		// Insert both prompts
		_, err := repo.Create(context.Background(), activePrompt)
		require.NoError(t, err)
		_, err = repo.Create(context.Background(), inactivePrompt)
		require.NoError(t, err)

		// WHEN searching for active prompts by name
		foundPrompt, err := repo.FindActiveByName(context.Background(), userID, promptName)

		// THEN only the active prompt should be returned
		require.NoError(t, err)
		assert.NotNil(t, foundPrompt)
		assert.Equal(t, true, foundPrompt.Active)
		assert.Equal(t, "Active template", foundPrompt.PromptTemplate)
	})

	t.Run("should return nil when searching for active prompt but only inactive exists", func(t *testing.T) {
		setup(t)

		// GIVEN only an inactive prompt with the specified name
		userID := "user-555"
		promptName := "inactive-only"

		inactivePrompt := &entities.Prompt{
			Name:           promptName,
			UserID:         userID,
			Type:           entities.PromptTypeIdeas,
			PromptTemplate: "Inactive template",
			Active:         false,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		// Insert the inactive prompt
		_, err := repo.Create(context.Background(), inactivePrompt)
		require.NoError(t, err)

		// WHEN searching for active prompts by name
		foundPrompt, err := repo.FindActiveByName(context.Background(), userID, promptName)

		// THEN no prompt should be found
		require.NoError(t, err)
		assert.Nil(t, foundPrompt)
	})

	t.Run("should handle name uniqueness validation during creation", func(t *testing.T) {
		setup(t)

		// GIVEN a prompt with a specific name already exists
		userID := "user-666"
		promptName := "duplicate-test"

		existingPrompt := &entities.Prompt{
			Name:           promptName,
			UserID:         userID,
			Type:           entities.PromptTypeIdeas,
			PromptTemplate: "Existing template",
			Active:         true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		// Insert the existing prompt
		_, err := repo.Create(context.Background(), existingPrompt)
		require.NoError(t, err)

		// WHEN creating another prompt with the same name for the same user
		duplicatePrompt := &entities.Prompt{
			Name:           promptName,
			UserID:         userID,
			Type:           entities.PromptTypeIdeas,
			PromptTemplate: "Duplicate template",
			Active:         true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		// THEN the repository should allow insertion (uniqueness handled at application level)
		newID, err := repo.Create(context.Background(), duplicatePrompt)

		// Note: This test verifies the repository behavior
		// In the actual implementation, uniqueness validation should be handled
		// at the application layer since MongoDB doesn't enforce uniqueness across fields
		require.NoError(t, err)
		assert.NotEmpty(t, newID)

		// AND both prompts should exist
		foundPrompts, err := repo.FindByNameAllUsers(context.Background(), promptName)
		require.NoError(t, err)
		assert.Len(t, foundPrompts, 2)
	})
}

// NEW methods to be added to PromptsRepository

// FindByName finds a prompt by name and user ID
func (r *PromptsRepository) FindByName(ctx context.Context, userID string, name string) (*entities.Prompt, error) {
	filter := bson.M{
		"user_id": userID,
		"name":    name,
	}

	var doc promptDocument
	err := r.collection.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find prompt by name: %w", err)
	}

	return r.toEntity(&doc), nil
}

// FindActiveByName finds an active prompt by name and user ID
func (r *PromptsRepository) FindActiveByName(ctx context.Context, userID string, name string) (*entities.Prompt, error) {
	filter := bson.M{
		"user_id": userID,
		"name":    name,
		"active":  true,
	}

	var doc promptDocument
	err := r.collection.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find active prompt by name: %w", err)
	}

	return r.toEntity(&doc), nil
}

// FindByNameAllUsers finds all prompts with a specific name across all users
func (r *PromptsRepository) FindByNameAllUsers(ctx context.Context, name string) ([]*entities.Prompt, error) {
	filter := bson.M{
		"name": name,
	}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find prompts by name: %w", err)
	}
	defer cursor.Close(ctx)

	var prompts []*entities.Prompt
	for cursor.Next(ctx) {
		var doc promptDocument
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("failed to decode prompt: %w", err)
		}
		prompts = append(prompts, r.toEntity(&doc))
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return prompts, nil
}
