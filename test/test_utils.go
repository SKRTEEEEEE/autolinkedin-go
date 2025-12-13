package utils

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/linkgen-ai/backend/test/application/services"
	"github.com/linkgen-ai/backend/src/domain/entities"
	"github.com/linkgen-ai/backend/src/domain/interfaces"
	"github.com/linkgen-ai/backend/src/infrastructure/database/repositories"
	"github.com/linkgen-ai/backend/src/infrastructure/services"
)

// TestDatabase holds test database connection and repositories
type TestDatabase struct {
	DB                *mongo.Database
	Client            *mongo.Client
	PromptRepo        interfaces.PromptsRepository
	TopicRepo         interfaces.TopicRepository
	IdeaRepo          interfaces.IdeaRepository
	DraftRepo         interfaces.DraftRepository
	UserRepo          interfaces.UserRepository
	SeedSyncService   *services.SeedSyncService
	DataMigrator      *services.DataMigrator
}

// SetupTestDB creates a test database connection and initializes repositories
func SetupTestDB(t *testing.T) *TestDatabase {
	t.Helper()

	// Check if running in CI environment, skip database tests if not available
	if os.Getenv("CI") == "true" && os.Getenv("MONGODB_TEST_URI") == "" {
		t.Skip("Skipping database test - no MongoDB test URI configured")
	}

	// Get MongoDB connection string from environment or use default
	mongoURI := os.Getenv("MONGODB_TEST_URI")
	if mongoURI == "" {
		// Default test database URI
		mongoURI = "mongodb://localhost:27017"
	}

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	require.NoError(t, err, "Failed to connect to MongoDB")

	// Ping the database to verify connection
	err = client.Ping(ctx, nil)
	require.NoError(t, err, "Failed to ping MongoDB")

	// Create a unique test database name
	testDBName := fmt.Sprintf("linkgenai_test_%d", time.Now().UnixNano())
	db := client.Database(testDBName)

	// Set up repositories
	promptRepo := repositories.NewPromptRepository(db)
	topicRepo := repositories.NewTopicRepository(db)
	ideaRepo := repositories.NewIdeaRepository(db)
	draftRepo := repositories.NewDraftRepository(db)
	userRepo := repositories.NewUserRepository(db)

	// Set up additional services
	promptLoader := services.NewPromptLoader(db)
	promptEngine := services.NewPromptEngine(nil, &test.TestLogger{})
	seedSyncService := services.NewSeedSyncService(db, promptLoader, promptEngine)
	dataMigrator := services.NewDataMigrator(db)

	// Create test user
	user := &entities.User{
		ID:        "test-user-123",
		Name:      "Test User",
		Email:     "test@example.com",
		Active:    true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = userRepo.Create(ctx, user)
	require.NoError(t, err, "Failed to create test user")

	return &TestDatabase{
		DB:                db,
		Client:            client,
		PromptRepo:        promptRepo,
		TopicRepo:         topicRepo,
		IdeaRepo:          ideaRepo,
		DraftRepo:         draftRepo,
		UserRepo:          userRepo,
		SeedSyncService:   seedSyncService,
		DataMigrator:      dataMigrator,
	}
}

// CleanupTestDB drops the test database and closes the connection
func CleanupTestDB(t *testing.T, testDB *TestDatabase) {
	t.Helper()

	if testDB == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Drop the test database
	if testDB.DB != nil {
		err := testDB.DB.Drop(ctx)
		if err != nil {
			t.Logf("Warning: Failed to drop test database: %v", err)
		}
	}

	// Disconnect the client
	if testDB.Client != nil {
		err := testDB.Client.Disconnect(ctx)
		if err != nil {
			t.Logf("Warning: Failed to disconnect from MongoDB: %v", err)
		}
	}
}

// CreateTestUser creates a test user in the database
func CreateTestUser(t *testing.T, db *TestDatabase) *entities.User {
	t.Helper()

	ctx := context.Background()
	userID := fmt.Sprintf("test-user-%d", time.Now().UnixNano())

	user := &entities.User{
		ID:        userID,
		Name:      "Test User",
		Email:     fmt.Sprintf("test-%d@example.com", time.Now().UnixNano()),
		Active:    true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := db.UserRepo.Create(ctx, user)
	require.NoError(t, err, "Failed to create test user")

	return user
}

// CreateTestPrompt creates a test prompt in the database
func CreateTestPrompt(t *testing.T, db *TestDatabase, userID string, name, promptType, template string) *entities.Prompt {
	t.Helper()

	ctx := context.Background()

	prompt := &entities.Prompt{
		ID:             fmt.Sprintf("prompt-%d", time.Now().UnixNano()),
		UserID:         userID,
		Name:           name,
		Type:           entities.PromptType(promptType),
		PromptTemplate: template,
		Active:         true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	err := db.PromptRepo.Create(ctx, prompt)
	require.NoError(t, err, "Failed to create test prompt")

	return prompt
}

// CreateTestTopic creates a test topic in the database
func CreateTestTopic(t *testing.T, db *TestDatabase, userID string, name, promptName string) *entities.Topic {
	t.Helper()

	ctx := context.Background()

	topic := &entities.Topic{
		ID:             fmt.Sprintf("topic-%d", time.Now().UnixNano()),
		UserID:         userID,
		Name:           name,
		Description:    fmt.Sprintf("Test topic: %s", name),
		PromptName:     promptName,
		Category:       "Test",
		Priority:       5,
		IdeasCount:     3,
		Keywords:       []string{"test", "example"},
		RelatedTopics:  []string{},
		Active:         true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	err := db.TopicRepo.Create(ctx, topic)
	require.NoError(t, err, "Failed to create test topic")

	return topic
}

// CreateTestIdeas creates test ideas for a topic
func CreateTestIdeas(t *testing.T, db *TestDatabase, userID, topicID, topicName string, count int) []*entities.Idea {
	t.Helper()

	ctx := context.Background()
	ideas := make([]*entities.Idea, count)

	for i := 0; i < count; i++ {
		idea := &entities.Idea{
			ID:        fmt.Sprintf("idea-%d-%d", time.Now().UnixNano(), i),
			Content:   fmt.Sprintf("Test idea %d for topic %s", i+1, topicName),
			TopicID:   topicID,
			TopicName: topicName,
			UserID:    userID,
			Used:      false,
			CreatedAt: time.Now(),
		}

		err := db.IdeaRepo.Create(ctx, idea)
		require.NoError(t, err, "Failed to create test idea")

		ideas[i] = idea
	}

	return ideas
}

// SetupSeedTestData creates seed test data for the migration tests
func SetupSeedTestData(t *testing.T, db *TestDatabase) string {
	t.Helper()

	userID := CreateTestUser(t, db).ID

	// Create test prompts
	CreateTestPrompt(t, db, userID, "base1", "ideas", "Generate {ideas} ideas about {name} with keywords: {[keywords]}")
	CreateTestPrompt(t, db, userID, "professional", "drafts", "Create professional content based on:\nContent: {content}\nTopic: {topic_name}\nUser Context: {user_context}")
	CreateTestPrompt(t, db, userID, "creative", "ideas", "Generate innovative {ideas} ideas about {name} in category {category}. Focus on: {[related_topics]}")

	// Create test topics
	CreateTestTopic(t, db, userID, "Desarrollo Backend", "base1")
	CreateTestTopic(t, db, userID, "Inteligencia Artificial", "creative")
	CreateTestTopic(t, db, userID, "TypeScript", "base1")
	CreateTestTopic(t, db, userID, "DevOps y Cloud", "professional")

	return userID
}

// CreateOldStyleTestData creates data in the old format for migration testing
func CreateOldStyleTestData(t *testing.T, db *TestDatabase) string {
	t.Helper()

	ctx := context.Background()
	userID := CreateTestUser(t, db).ID

	// Connect directly to collections to create old-style data
	topicCollection := db.DB.Collection("topics")
	ideaCollection := db.DB.Collection("ideas")
	promptCollection := db.DB.Collection("prompts")

	// Create old topic structure (pre-refactor)
	oldTopics := []map[string]interface{}{
		{
			"_id":         fmt.Sprintf("old-topic-%d-1", time.Now().UnixNano()),
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
		},
		{
			"_id":         fmt.Sprintf("old-topic-%d-2", time.Now().UnixNano()),
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

	// Insert old topics
	for _, topic := range oldTopics {
		_, err := topicCollection.InsertOne(ctx, topic)
		require.NoError(t, err)
	}

	// Create old prompt structure with style_name
	oldPrompts := []map[string]interface{}{
		{
			"_id":            fmt.Sprintf("old-prompt-%d-1", time.Now().UnixNano()),
			"user_id":        userID,
			"style_name":     "professional", // Old field
			"type":           "drafts",
			"prompt_template": "Generate professional content about {topic}",
			"active":         true,
			"created_at":     time.Now().Add(-15 * 24 * time.Hour),
		},
		{
			"_id":            fmt.Sprintf("old-prompt-%d-2", time.Now().UnixNano()),
			"user_id":        userID,
			"style_name":     "creative",
			"type":           "ideas",
			"prompt_template": "Generate creative ideas about {name}",
			"active":         true,
			"created_at":     time.Now().Add(-8 * 24 * time.Hour),
		},
	}

	// Insert old prompts
	for _, prompt := range oldPrompts {
		_, err := promptCollection.InsertOne(ctx, prompt)
		require.NoError(t, err)
	}

	// Create old ideas structure (without topic_name)
	topicID1 := oldTopics[0]["_id"].(string)
	topicID2 := oldTopics[1]["_id"].(string)

	oldIdeas := []map[string]interface{}{
		{
			"_id":      fmt.Sprintf("old-idea-%d-1", time.Now().UnixNano()),
			"user_id":  userID,
			"topic_id": topicID1,
			"content":  "Building scalable GraphQL APIs with federation",
			"used":     false,
			"created_at": time.Now().Add(-7 * 24 * time.Hour),
		},
		{
			"_id":      fmt.Sprintf("old-idea-%d-2", time.Now().UnixNano()),
			"user_id":  userID,
			"topic_id": topicID2,
			"content":  "Index optimization strategies for large datasets",
			"used":     true,
			"created_at": time.Now().Add(-3 * 24 * time.Hour),
		},
	}

	// Insert old ideas
	for _, idea := range oldIdeas {
		_, err := ideaCollection.InsertOne(ctx, idea)
		require.NoError(t, err)
	}

	return userID
}

// TestLogger is a simple logger implementation for testing
type TestLogger struct{}

// Info logs an info message
func (l *TestLogger) Info(message string) {
	fmt.Printf("[INFO] %s\n", message)
}

// Error logs an error message
func (l *TestLogger) Error(message string) {
	fmt.Printf("[ERROR] %s\n", message)
}

// Debug logs a debug message
func (l *TestLogger) Debug(message string) {
	fmt.Printf("[DEBUG] %s\n", message)
}

// Warn logs a warning message
func (l *TestLogger) Warn(message string) {
	fmt.Printf("[WARN] %s\n", message)
}
