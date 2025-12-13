package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/linkgen-ai/backend/src/domain/entities"
	"github.com/linkgen-ai/backend/src/infrastructure/database/repositories"
	"github.com/linkgen-ai/backend/src/infrastructure/services"
	"github.com/linkgen-ai/backend/test/utils"
)

// TestDataMigration tests migration from old to new data structures
func TestDataMigration(tt *testing.T) {
	ctx := context.Background()

	tt.Run("should migrate topics from old schema to new schema", func(t *testing.T) {
		// GIVEN database with old topic structure
		db := setupTestDB(t)
		defer cleanupTestDB(t, db)

		userID := primitive.NewObjectID().Hex()

		// Create old topic structure (pre-refactor)
		oldTopics := []map[string]interface{}{
			{
				"_id":         primitive.NewObjectID().Hex(),
				"user_id":     userID,
				"name":        "React Hooks",
				"description": "Understanding React Hooks",
				"keywords":    []string{"react", "hooks"},
				"category":    "",
				"priority":    5,
				"ideas":       4, // Old field name
				"active":      true,
				"created_at":  time.Now().Add(-10 * 24 * time.Hour),
				"updated_at":  time.Now().Add(-10 * 24 * time.Hour),
				// Missing fields: prompt_name, ideas_count
			},
			{
				"_id":         primitive.NewObjectID().Hex(),
				"user_id":     userID,
				"name":        "Vue Composition API",
				"description": "Vue's new composition API",
				"keywords":    []string{"vue", "composition"},
				"category":    "",
				"priority":    7,
				"ideas":       3,
				"active":      true,
				"created_at":  time.Now().Add(-5 * 24 * time.Hour),
				"updated_at":  time.Now().Add(-5 * 24 * time.Hour),
			},
		}

		// Insert old topics directly to simulate production data
		topicCollection := db.Collection("topics")
		for _, topic := range oldTopics {
			_, err := topicCollection.InsertOne(ctx, topic)
			require.NoError(t, err)
		}

		// WHEN running data migration
		migrator := NewDataMigrator(db)
		migrationReport, err := migrator.MigrateTopics(ctx)

		// This will fail until migration logic is implemented
		t.Fatal("implement topic data migration from old to new schema - FAILING IN TDD RED PHASE")

		// THEN should:
		// 1. Create new fields (prompt_name, ideas_count)
		// 2. Map old 'ideas' to 'ideas_count'
		// 3. Set default prompt_name = "base1"
		// 4. Provide migration report
		require.NoError(t, err)
		assert.NotZero(t, migrationReport.MigratedCount)
		assert.Equal(t, 2, migrationReport.MigratedCount)

		// Verify migration results
		topicRepo := repositories.NewTopicRepository(db)
		topics, err := topicRepo.ListByUserID(ctx, userID)
		require.NoError(t, err)
		require.Len(t, topics, 2)

		for _, topic := range topics {
			assert.Equal(t, "base1", topic.PromptName, "Should set default prompt")
			assert.Greater(t, topic.IdeasCount, 0, "Should map old ideas field")
			assert.Equal(t, topic.UpdatedAt, migrationReport.MigrationDate)
		}
	})

	tt.Run("should migrate ideas without topic_name to include topic names", func(t *testing.T) {
		// GIVEN database with ideas lacking topic_name field
		db := setupTestDB(t)
		defer cleanupTestDB(t, db)

		userID := primitive.NewObjectID().Hex()

		// Create topics first (with new structure)
		topicRepo := repositories.NewTopicRepository(db)
		topic1 := &entities.Topic{
			ID:        primitive.NewObjectID().Hex(),
			UserID:    userID,
			Name:      "GraphQL APIs",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		topic2 := &entities.Topic{
			ID:        primitive.NewObjectID().Hex(),
			UserID:    userID,
			Name:      "SQL Optimization",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		createdTopic1, err := topicRepo.Create(ctx, topic1)
		require.NoError(t, err)
		createdTopic2, err := topicRepo.Create(ctx, topic2)
		require.NoError(t, err)

		// Create old ideas structure (without topic_name)
		oldIdeas := []map[string]interface{}{
			{
				"_id":        primitive.NewObjectID().Hex(),
				"user_id":    userID,
				"topic_id":   createdTopic1.ID,
				"content":    "Building scalable GraphQL APIs with federation",
				"used":       false,
				"created_at": time.Now().Add(-7 * 24 * time.Hour),
				// Missing: topic_name
			},
			{
				"_id":        primitive.NewObjectID().Hex(),
				"user_id":    userID,
				"topic_id":   createdTopic2.ID,
				"content":    "Index optimization strategies for large datasets",
				"used":       true,
				"created_at": time.Now().Add(-3 * 24 * time.Hour),
			},
		}

		ideaCollection := db.Collection("ideas")
		for _, idea := range oldIdeas {
			_, err := ideaCollection.InsertOne(ctx, idea)
			require.NoError(t, err)
		}

		// WHEN running idea migration
		migrator := NewDataMigrator(db)
		report, err := migrator.MigrateIdeas(ctx)

		// This will fail until idea migration is implemented
		t.Fatal("implement idea migration to populate topic_name - FAILING IN TDD RED PHASE")

		// THEN should populate topic_name for all ideas
		require.NoError(t, err)
		assert.Equal(t, 2, report.MigratedCount)

		// Verify results
		ideaRepo := repositories.NewIdeaRepository(db)
		ideas, err := ideaRepo.ListByUserID(ctx, userID)
		require.NoError(t, err)
		require.Len(t, ideas, 2)

		for _, idea := range ideas {
			assert.NotEmpty(t, idea.TopicName, "Should have topic name populated")
			if idea.TopicID == createdTopic1.ID {
				assert.Equal(t, "GraphQL APIs", idea.TopicName)
			}
			if idea.TopicID == createdTopic2.ID {
				assert.Equal(t, "SQL Optimization", idea.TopicName)
			}
		}
	})

	tt.Run("should migrate prompts from style_name to name field", func(t *testing.T) {
		// GIVEN database with old prompt structure
		db := setupTestDB(t)
		defer cleanupTestDB(t, db)

		userID := primitive.NewObjectID().Hex()

		// Old prompt structure with style_name
		oldPrompts := []map[string]interface{}{
			{
				"_id":             primitive.NewObjectID().Hex(),
				"user_id":         userID,
				"style_name":      "professional", // Old field
				"type":            "drafts",
				"prompt_template": "Generate professional content about {topic}",
				"active":          true,
				"created_at":      time.Now().Add(-15 * 24 * time.Hour),
				// Missing: name field
			},
			{
				"_id":             primitive.NewObjectID().Hex(),
				"user_id":         userID,
				"style_name":      "creative",
				"type":            "ideas",
				"prompt_template": "Generate creative ideas about {name}",
				"active":          true,
				"created_at":      time.Now().Add(-8 * 24 * time.Hour),
			},
		}

		promptCollection := db.Collection("prompts")
		for _, prompt := range oldPrompts {
			_, err := promptCollection.InsertOne(ctx, prompt)
			require.NoError(t, err)
		}

		// WHEN running prompt migration
		migrator := NewDataMigrator(db)
		report, err := migrator.MigratePrompts(ctx)

		// This will fail until prompt migration is implemented
		t.Fatal("implement prompt migration from style_name to name - FAILING IN TDD RED PHASE")

		// THEN should map style_name to name field
		require.NoError(t, err)
		assert.Equal(t, 2, report.MigratedCount)

		// Verify results
		promptRepo := repositories.NewPromptRepository(db)
		prompts, err := promptRepo.ListByUserID(ctx, userID)
		require.NoError(t, err)
		require.Len(t, prompts, 2)

		hasProfessional := false
		hasCreative := false
		for _, prompt := range prompts {
			if prompt.Name == "professional" {
				hasProfessional = true
				assert.Equal(t, entities.PromptTypeDrafts, prompt.Type)
			}
			if prompt.Name == "creative" {
				hasCreative = true
				assert.Equal(t, entities.PromptTypeIdeas, prompt.Type)
			}
		}
		assert.True(t, hasProfessional, "Should migrate professional prompt")
		assert.True(t, hasCreative, "Should migrate creative prompt")
	})
}

// TestMigrationRollback tests migration rollback capabilities
func TestMigrationRollback(tt *testing.T) {
	ctx := context.Background()

	tt.Run("should support rolling back topic migration", func(t *testing.T) {
		// GIVEN a migrated state
		testDB := utils.SetupTestDB(t)
		defer utils.CleanupTestDB(t, testDB)

		userID := utils.CreateTestUser(t, testDB).ID

		// Start with new structure (post-migration)
		newTopic := &entities.Topic{
			ID:         primitive.NewObjectID().Hex(),
			UserID:     userID,
			Name:       "Topic for rollback test",
			PromptName: "custom",
			IdeasCount: 5,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		createdTopic, err := testDB.TopicRepo.Create(ctx, newTopic)
		require.NoError(t, err)

		// WHEN rolling back migration
		migrator := services.NewDataMigrator(testDB.DB)
		rollbackReport, err := migrator.RollbackTopics(ctx, userID)

		require.NoError(t, err)
		assert.Equal(t, 1, rollbackReport.RolledBackCount)

		// THEN should:
		// 1. Remove new fields (prompt_name)
		// 2. Restore old field name (ideas instead of ideas_count)
		// 3. Keep existing data intact

		// Verify rollback worked directly in collection
		topicCollection := testDB.DB.Collection("topics")
		var rolledBackTopic map[string]interface{}
		err = topicCollection.FindOne(ctx, map[string]interface{}{
			"_id": createdTopic.ID,
		}).Decode(&rolledBackTopic)
		require.NoError(t, err)

		assert.NotNil(t, rolledBackTopic["ideas"])
		assert.Nil(t, rolledBackTopic["prompt_name"])
		assert.Nil(t, rolledBackTopic["ideas_count"])
	})

	tt.Run("should provide migration validation before execution", func(t *testing.T) {
		// GIVEN database with mixed old and new structures
		testDB := utils.SetupTestDB(t)
		defer utils.CleanupTestDB(t, testDB)

		userID := utils.CreateTestUser(t, testDB).ID

		// Mix of old and new data
		oldTopic := map[string]interface{}{
			"_id":        primitive.NewObjectID().Hex(),
			"user_id":    userID,
			"name":       "Old Topic",
			"ideas":      3,
			"created_at": time.Now(),
		}

		newTopic := map[string]interface{}{
			"_id":         primitive.NewObjectID().Hex(),
			"user_id":     userID,
			"name":        "New Topic",
			"prompt_name": "base1",
			"ideas_count": 4,
			"created_at":  time.Now(),
		}

		topicCollection := testDB.DB.Collection("topics")
		_, err := topicCollection.InsertOne(ctx, oldTopic)
		require.NoError(t, err)
		_, err = topicCollection.InsertOne(ctx, newTopic)
		require.NoError(t, err)

		// WHEN validating migration
		migrator := services.NewDataMigrator(testDB.DB)
		validationReport, err := migrator.ValidateMigration(ctx, userID)

		require.NoError(t, err)

		// THEN should:
		// 1. Identify which topics need migration
		// 2. Detect conflicts or issues
		// 3. Provide detailed report
		assert.Equal(t, 1, validationReport.ItemsNeedingMigration)
		assert.Equal(t, 1, validationReport.ItemsAlreadyMigrated)
		assert.Equal(t, 0, validationReport.ConflictCount)
	})
}

// MigrationReport represents the results of a migration operation
type MigrationReport struct {
	MigratedCount int       `json:"migrated_count"`
	FailedCount   int       `json:"failed_count"`
	MigrationDate time.Time `json:"migration_date"`
	Errors        []string  `json:"errors,omitempty"`
}

// RollbackReport represents the results of a rollback operation
type RollbackReport struct {
	RolledBackCount int       `json:"rolled_back_count"`
	FailedCount     int       `json:"failed_count"`
	RollbackDate    time.Time `json:"rollback_date"`
	Errors          []string  `json:"errors,omitempty"`
}

// ValidationReport represents migration validation results
type ValidationReport struct {
	ItemsNeedingMigration int       `json:"items_needing_migration"`
	ItemsAlreadyMigrated  int       `json:"items_already_migrated"`
	ConflictCount         int       `json:"conflict_count"`
	ValidationDate        time.Time `json:"validation_date"`
}

// TestDataMigration tests migration from old to new data structures
func TestDataMigration(tt *testing.T) {
	ctx := context.Background()

	tt.Run("should migrate topics from old schema to new schema", func(t *testing.T) {
		// GIVEN database with old topic structure
		testDB := utils.SetupTestDB(t)
		defer utils.CleanupTestDB(t, testDB)

		userID := utils.CreateOldStyleTestData(t, testDB)

		// WHEN running data migration
		migrator := services.NewDataMigrator(testDB.DB)
		migrationReport, err := migrator.MigrateTopics(ctx)

		require.NoError(t, err)
		assert.NotZero(t, migrationReport.MigratedCount)
		assert.Equal(t, 2, migrationReport.MigratedCount)

		// THEN should:
		// 1. Create new fields (prompt_name, ideas_count)
		// 2. Map old 'ideas' to 'ideas_count'
		// 3. Set default prompt_name = "base1"
		// 4. Provide migration report

		// Verify migration results
		topicRepo := repositories.NewTopicRepository(testDB.DB)
		topics, err := topicRepo.ListByUserID(ctx, userID)
		require.NoError(t, err)
		require.Len(t, topics, 2)

		for _, topic := range topics {
			assert.Equal(t, "base1", topic.PromptName, "Should set default prompt")
			assert.Greater(t, topic.IdeasCount, 0, "Should map old ideas field")
			assert.Equal(t, migrationReport.MigrationDate, topic.UpdatedAt)
		}
	})

	tt.Run("should migrate ideas without topic_name to include topic names", func(t *testing.T) {
		// GIVEN database with ideas lacking topic_name field
		testDB := utils.SetupTestDB(t)
		defer utils.CleanupTestDB(t, testDB)

		userID := utils.CreateOldStyleTestData(t, testDB)

		// WHEN running idea migration
		migrator := services.NewDataMigrator(testDB.DB)
		report, err := migrator.MigrateIdeas(ctx)

		require.NoError(t, err)
		assert.Equal(t, 2, report.MigratedCount)

		// THEN should populate topic_name for all ideas
		ideaRepo := repositories.NewIdeaRepository(testDB.DB)
		ideas, err := ideaRepo.ListByUserID(ctx, userID)
		require.NoError(t, err)
		require.Len(t, ideas, 2)

		for _, idea := range ideas {
			assert.NotEmpty(t, idea.TopicName, "Should have topic name populated")
		}
	})

	tt.Run("should migrate prompts from style_name to name field", func(t *testing.T) {
		// GIVEN database with old prompt structure
		testDB := utils.SetupTestDB(t)
		defer utils.CleanupTestDB(t, testDB)

		userID := utils.CreateOldStyleTestData(t, testDB)

		// WHEN running prompt migration
		migrator := services.NewDataMigrator(testDB.DB)
		report, err := migrator.MigratePrompts(ctx)

		require.NoError(t, err)
		assert.Equal(t, 2, report.MigratedCount)

		// THEN should map style_name to name field
		promptRepo := repositories.NewPromptRepository(testDB.DB)
		prompts, err := promptRepo.ListByUserID(ctx, userID)
		require.NoError(t, err)
		require.Len(t, prompts, 2)

		hasProfessional := false
		hasCreative := false
		for _, prompt := range prompts {
			if prompt.Name == "professional" {
				hasProfessional = true
				assert.Equal(t, entities.PromptTypeDrafts, prompt.Type)
			}
			if prompt.Name == "creative" {
				hasCreative = true
				assert.Equal(t, entities.PromptTypeIdeas, prompt.Type)
			}
		}
		assert.True(t, hasProfessional, "Should migrate professional prompt")
		assert.True(t, hasCreative, "Should migrate creative prompt")
	})
}
