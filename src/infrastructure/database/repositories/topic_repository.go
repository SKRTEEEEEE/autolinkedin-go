package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/linkgen-ai/backend/src/domain/entities"
	"github.com/linkgen-ai/backend/src/domain/interfaces"
	"github.com/linkgen-ai/backend/src/infrastructure/database"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// topicRepository implements the TopicRepository interface for MongoDB
type topicRepository struct {
	*database.BaseRepository
	collection *mongo.Collection
}

// NewTopicRepository creates a new MongoDB topic repository
func NewTopicRepository(collection *mongo.Collection) interfaces.TopicRepository {
	return &topicRepository{
		BaseRepository: database.NewBaseRepository(collection),
		collection:     collection,
	}
}

// topicDocument represents the MongoDB document structure for Topic
type topicDocument struct {
	ID            primitive.ObjectID `bson:"_id,omitempty"`
	UserID        primitive.ObjectID `bson:"user_id"`
	Name          string             `bson:"name"`
	Description   string             `bson:"description"`
	Category      string             `bson:"category"`
	Priority      int                `bson:"priority"`
	Ideas         int                `bson:"ideas"`
	Prompt        string             `bson:"prompt"`
	RelatedTopics []string           `bson:"related_topics"`
	Active        bool               `bson:"active"`
	CreatedAt     primitive.DateTime `bson:"created_at"`
	UpdatedAt     primitive.DateTime `bson:"updated_at"`
}

// toDocument converts a Topic entity to a MongoDB document
func (r *topicRepository) toDocument(topic *entities.Topic) (*topicDocument, error) {
	if topic == nil {
		return nil, fmt.Errorf("topic cannot be nil")
	}

	userObjectID, err := primitive.ObjectIDFromHex(topic.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	doc := &topicDocument{
		UserID:        userObjectID,
		Name:          topic.Name,
		Description:   topic.Description,
		Category:      topic.Category,
		Priority:      topic.Priority,
		Ideas:         topic.Ideas,
		Prompt:        topic.Prompt,
		RelatedTopics: topic.RelatedTopics,
		Active:        topic.Active,
		CreatedAt:     primitive.NewDateTimeFromTime(topic.CreatedAt),
		UpdatedAt:     primitive.NewDateTimeFromTime(topic.UpdatedAt),
	}

	// Only set ID if it's valid
	if topic.ID != "" {
		objectID, err := primitive.ObjectIDFromHex(topic.ID)
		if err != nil {
			return nil, fmt.Errorf("invalid topic ID: %w", err)
		}
		doc.ID = objectID
	}

	return doc, nil
}

// toEntity converts a MongoDB document to a Topic entity
func (r *topicRepository) toEntity(doc *topicDocument) *entities.Topic {
	if doc == nil {
		return nil
	}

	return &entities.Topic{
		ID:            doc.ID.Hex(),
		UserID:        doc.UserID.Hex(),
		Name:          doc.Name,
		Description:   doc.Description,
		Category:      doc.Category,
		Priority:      doc.Priority,
		Ideas:         doc.Ideas,
		Prompt:        doc.Prompt,
		RelatedTopics: doc.RelatedTopics,
		Active:        doc.Active,
		CreatedAt:     doc.CreatedAt.Time(),
		UpdatedAt: func() time.Time {
			updatedAt := doc.UpdatedAt.Time()
			if updatedAt.IsZero() {
				return doc.CreatedAt.Time()
			}
			return updatedAt
		}(),
	}
}

// Create creates a new topic in the database
func (r *topicRepository) Create(ctx context.Context, topic *entities.Topic) (string, error) {
	if topic == nil {
		return "", database.ErrInvalidEntity
	}

	if topic.UpdatedAt.IsZero() {
		topic.UpdatedAt = topic.CreatedAt
	}

	// Validate topic before persisting
	if err := topic.Validate(); err != nil {
		return "", fmt.Errorf("topic validation failed: %w", err)
	}

	doc, err := r.toDocument(topic)
	if err != nil {
		return "", err
	}

	result, err := r.collection.InsertOne(ctx, doc)
	if err != nil {
		return "", fmt.Errorf("failed to create topic: %w", err)
	}

	insertedID, ok := result.InsertedID.(primitive.ObjectID)
	if !ok {
		return "", fmt.Errorf("failed to convert inserted ID to ObjectID")
	}

	return insertedID.Hex(), nil
}

// FindByID retrieves a topic by its ID
func (r *topicRepository) FindByID(ctx context.Context, topicID string) (*entities.Topic, error) {
	if topicID == "" {
		return nil, database.ErrInvalidID
	}

	objectID, err := primitive.ObjectIDFromHex(topicID)
	if err != nil {
		return nil, database.ErrInvalidID
	}

	var doc topicDocument
	filter := bson.M{"_id": objectID}

	err = r.collection.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, database.ErrEntityNotFound
		}
		return nil, fmt.Errorf("failed to find topic by ID: %w", err)
	}

	return r.toEntity(&doc), nil
}

// ListByUserID retrieves all topics belonging to a specific user
func (r *topicRepository) ListByUserID(ctx context.Context, userID string) ([]*entities.Topic, error) {
	if userID == "" {
		return nil, database.ErrInvalidID
	}

	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, database.ErrInvalidID
	}

	filter := bson.M{"user_id": userObjectID}
	// Sort by priority (descending) and then by name
	opts := options.Find().SetSort(bson.D{
		{Key: "priority", Value: -1},
		{Key: "name", Value: 1},
	})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list topics by user ID: %w", err)
	}
	defer cursor.Close(ctx)

	var topics []*entities.Topic
	for cursor.Next(ctx) {
		var doc topicDocument
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("failed to decode topic document: %w", err)
		}
		topics = append(topics, r.toEntity(&doc))
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return topics, nil
}

// FindRandomByUserID selects a random topic from user's topics
func (r *topicRepository) FindRandomByUserID(ctx context.Context, userID string) (*entities.Topic, error) {
	if userID == "" {
		return nil, database.ErrInvalidID
	}

	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, database.ErrInvalidID
	}

	// Use MongoDB's $sample aggregation to get a random document
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"user_id": userObjectID}}},
		{{Key: "$sample", Value: bson.M{"size": 1}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to find random topic: %w", err)
	}
	defer cursor.Close(ctx)

	if !cursor.Next(ctx) {
		// No topics found for this user
		return nil, database.ErrEntityNotFound
	}

	var doc topicDocument
	if err := cursor.Decode(&doc); err != nil {
		return nil, fmt.Errorf("failed to decode random topic: %w", err)
	}

	return r.toEntity(&doc), nil
}

// Update updates an existing topic in the database
func (r *topicRepository) Update(ctx context.Context, topic *entities.Topic) error {
	if topic == nil {
		return database.ErrInvalidEntity
	}

	if topic.ID == "" {
		return database.ErrInvalidID
	}

	topic.UpdatedAt = time.Now()

	// Validate topic before persisting
	if err := topic.Validate(); err != nil {
		return fmt.Errorf("topic validation failed: %w", err)
	}

	objectID, err := primitive.ObjectIDFromHex(topic.ID)
	if err != nil {
		return database.ErrInvalidID
	}

	// Prepare update document (excluding ID, UserID, and CreatedAt)
	update := bson.M{
		"$set": bson.M{
			"name":           topic.Name,
			"description":    topic.Description,
			"category":       topic.Category,
			"priority":       topic.Priority,
			"ideas":          topic.Ideas,
			"prompt":         topic.Prompt,
			"related_topics": topic.RelatedTopics,
			"active":         topic.Active,
			"updated_at":     primitive.NewDateTimeFromTime(topic.UpdatedAt),
		},
	}

	filter := bson.M{"_id": objectID}
	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update topic: %w", err)
	}

	if result.MatchedCount == 0 {
		return database.ErrEntityNotFound
	}

	return nil
}

// Delete removes a topic from the database
func (r *topicRepository) Delete(ctx context.Context, topicID string) error {
	if topicID == "" {
		return database.ErrInvalidID
	}

	objectID, err := primitive.ObjectIDFromHex(topicID)
	if err != nil {
		return database.ErrInvalidID
	}

	filter := bson.M{"_id": objectID}
	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete topic: %w", err)
	}

	if result.DeletedCount == 0 {
		return database.ErrEntityNotFound
	}

	return nil
}

// FindByPrompt retrieves topics that reference a specific prompt
func (r *topicRepository) FindByPrompt(ctx context.Context, userID string, promptName string) ([]*entities.Topic, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, database.ErrInvalidID
	}

	filter := bson.M{
		"user_id": userObjectID,
		"prompt":  promptName,
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

// FindByIdeasRange retrieves topics with ideas count in the specified range
func (r *topicRepository) FindByIdeasRange(ctx context.Context, userID string, minIdeas, maxIdeas int) ([]*entities.Topic, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, database.ErrInvalidID
	}

	filter := bson.M{
		"user_id": userObjectID,
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
