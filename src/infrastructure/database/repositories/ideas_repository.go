package repositories

import (
	"context"
	"fmt"

	"github.com/linkgen-ai/backend/src/domain/entities"
	"github.com/linkgen-ai/backend/src/domain/interfaces"
	"github.com/linkgen-ai/backend/src/infrastructure/database"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ideasRepository implements the IdeasRepository interface for MongoDB
type ideasRepository struct {
	*database.BaseRepository
	collection *mongo.Collection
}

// NewIdeasRepository creates a new MongoDB ideas repository
func NewIdeasRepository(collection *mongo.Collection) interfaces.IdeasRepository {
	return &ideasRepository{
		BaseRepository: database.NewBaseRepository(collection),
		collection:     collection,
	}
}

// ideaDocument represents the MongoDB document structure for Idea
type ideaDocument struct {
	ID           primitive.ObjectID  `bson:"_id,omitempty"`
	UserID       primitive.ObjectID  `bson:"user_id"`
	TopicID      primitive.ObjectID  `bson:"topic_id"`
	Content      string              `bson:"content"`
	QualityScore *float64            `bson:"quality_score,omitempty"`
	Used         bool                `bson:"used"`
	CreatedAt    primitive.DateTime  `bson:"created_at"`
	ExpiresAt    *primitive.DateTime `bson:"expires_at,omitempty"`
}

// toDocument converts an Idea entity to a MongoDB document
func (r *ideasRepository) toDocument(idea *entities.Idea) (*ideaDocument, error) {
	if idea == nil {
		return nil, fmt.Errorf("idea cannot be nil")
	}

	userObjectID, err := primitive.ObjectIDFromHex(idea.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	topicObjectID, err := primitive.ObjectIDFromHex(idea.TopicID)
	if err != nil {
		return nil, fmt.Errorf("invalid topic ID: %w", err)
	}

	doc := &ideaDocument{
		UserID:       userObjectID,
		TopicID:      topicObjectID,
		Content:      idea.Content,
		QualityScore: idea.QualityScore,
		Used:         idea.Used,
		CreatedAt:    primitive.NewDateTimeFromTime(idea.CreatedAt),
	}

	// Only set ID if it's valid
	if idea.ID != "" {
		objectID, err := primitive.ObjectIDFromHex(idea.ID)
		if err != nil {
			return nil, fmt.Errorf("invalid idea ID: %w", err)
		}
		doc.ID = objectID
	}

	// Set expiration if present
	if idea.ExpiresAt != nil {
		expiresAt := primitive.NewDateTimeFromTime(*idea.ExpiresAt)
		doc.ExpiresAt = &expiresAt
	}

	return doc, nil
}

// toEntity converts a MongoDB document to an Idea entity
func (r *ideasRepository) toEntity(doc *ideaDocument) *entities.Idea {
	if doc == nil {
		return nil
	}

	idea := &entities.Idea{
		ID:           doc.ID.Hex(),
		UserID:       doc.UserID.Hex(),
		TopicID:      doc.TopicID.Hex(),
		Content:      doc.Content,
		QualityScore: doc.QualityScore,
		Used:         doc.Used,
		CreatedAt:    doc.CreatedAt.Time(),
	}

	if doc.ExpiresAt != nil {
		expiresAt := doc.ExpiresAt.Time()
		idea.ExpiresAt = &expiresAt
	}

	return idea
}

// CreateBatch creates multiple ideas at once
func (r *ideasRepository) CreateBatch(ctx context.Context, ideas []*entities.Idea) error {
	if len(ideas) == 0 {
		return fmt.Errorf("ideas list cannot be empty")
	}

	// Validate all ideas first
	for i, idea := range ideas {
		if idea == nil {
			return fmt.Errorf("idea at index %d is nil", i)
		}
		if err := idea.Validate(); err != nil {
			return fmt.Errorf("idea at index %d validation failed: %w", i, err)
		}
	}

	// Convert to documents
	documents := make([]interface{}, len(ideas))
	for i, idea := range ideas {
		doc, err := r.toDocument(idea)
		if err != nil {
			return fmt.Errorf("failed to convert idea at index %d: %w", i, err)
		}
		documents[i] = doc
	}

	// Insert all at once
	_, err := r.collection.InsertMany(ctx, documents)
	if err != nil {
		return fmt.Errorf("failed to create batch of ideas: %w", err)
	}

	return nil
}

// ListByUserID retrieves ideas for a user with optional filtering
func (r *ideasRepository) ListByUserID(ctx context.Context, userID string, topicID string, limit int) ([]*entities.Idea, error) {
	if userID == "" {
		return nil, database.ErrInvalidID
	}

	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, database.ErrInvalidID
	}

	// Build filter
	filter := bson.M{"user_id": userObjectID}

	// Add topic filter if provided
	if topicID != "" {
		topicObjectID, err := primitive.ObjectIDFromHex(topicID)
		if err != nil {
			return nil, fmt.Errorf("invalid topic ID: %w", err)
		}
		filter["topic_id"] = topicObjectID
	}

	// Set options
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list ideas: %w", err)
	}
	defer cursor.Close(ctx)

	var ideas []*entities.Idea
	for cursor.Next(ctx) {
		var doc ideaDocument
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("failed to decode idea document: %w", err)
		}
		ideas = append(ideas, r.toEntity(&doc))
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return ideas, nil
}

// CountByUserID returns the total number of ideas for a user
func (r *ideasRepository) CountByUserID(ctx context.Context, userID string) (int64, error) {
	if userID == "" {
		return 0, database.ErrInvalidID
	}

	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return 0, database.ErrInvalidID
	}

	filter := bson.M{"user_id": userObjectID}
	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to count ideas: %w", err)
	}

	return count, nil
}

// ClearByUserID removes all ideas for a specific user
func (r *ideasRepository) ClearByUserID(ctx context.Context, userID string) error {
	if userID == "" {
		return database.ErrInvalidID
	}

	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return database.ErrInvalidID
	}

	filter := bson.M{"user_id": userObjectID}
	_, err = r.collection.DeleteMany(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to clear ideas: %w", err)
	}

	// Note: We don't return an error if no documents were deleted
	// because clearing an empty collection is still a successful operation
	return nil
}
