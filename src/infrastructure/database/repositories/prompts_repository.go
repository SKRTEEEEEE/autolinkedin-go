package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/linkgen-ai/backend/src/domain/entities"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// PromptsRepository implements interfaces.PromptsRepository for MongoDB
type PromptsRepository struct {
	collection *mongo.Collection
}

// NewPromptsRepository creates a new PromptsRepository
func NewPromptsRepository(collection *mongo.Collection) *PromptsRepository {
	return &PromptsRepository{
		collection: collection,
	}
}

// promptDocument represents the MongoDB document structure for Prompt
type promptDocument struct {
	ID             primitive.ObjectID `bson:"_id,omitempty"`
	UserID         string             `bson:"user_id"`
	Type           string             `bson:"type"`
	StyleName      string             `bson:"style_name,omitempty"`
	PromptTemplate string             `bson:"prompt_template"`
	Active         bool               `bson:"active"`
	CreatedAt      time.Time          `bson:"created_at"`
	UpdatedAt      time.Time          `bson:"updated_at"`
}

// toDocument converts a Prompt entity to a MongoDB document
func (r *PromptsRepository) toDocument(prompt *entities.Prompt) (*promptDocument, error) {
	var oid primitive.ObjectID
	var err error

	if prompt.ID == "" {
		oid = primitive.NewObjectID()
	} else {
		oid, err = primitive.ObjectIDFromHex(prompt.ID)
		if err != nil {
			return nil, fmt.Errorf("invalid prompt ID: %w", err)
		}
	}

	return &promptDocument{
		ID:             oid,
		UserID:         prompt.UserID,
		Type:           string(prompt.Type),
		StyleName:      prompt.StyleName,
		PromptTemplate: prompt.PromptTemplate,
		Active:         prompt.Active,
		CreatedAt:      prompt.CreatedAt,
		UpdatedAt:      prompt.UpdatedAt,
	}, nil
}

// toEntity converts a MongoDB document to a Prompt entity
func (r *PromptsRepository) toEntity(doc *promptDocument) *entities.Prompt {
	return &entities.Prompt{
		ID:             doc.ID.Hex(),
		UserID:         doc.UserID,
		Type:           entities.PromptType(doc.Type),
		StyleName:      doc.StyleName,
		PromptTemplate: doc.PromptTemplate,
		Active:         doc.Active,
		CreatedAt:      doc.CreatedAt,
		UpdatedAt:      doc.UpdatedAt,
	}
}

// Create implements PromptsRepository.Create
func (r *PromptsRepository) Create(ctx context.Context, prompt *entities.Prompt) (string, error) {
	doc, err := r.toDocument(prompt)
	if err != nil {
		return "", err
	}

	result, err := r.collection.InsertOne(ctx, doc)
	if err != nil {
		return "", fmt.Errorf("failed to insert prompt: %w", err)
	}

	oid, ok := result.InsertedID.(primitive.ObjectID)
	if !ok {
		return "", fmt.Errorf("failed to convert inserted ID to ObjectID")
	}

	return oid.Hex(), nil
}

// FindByID implements PromptsRepository.FindByID
func (r *PromptsRepository) FindByID(ctx context.Context, id string) (*entities.Prompt, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid prompt ID: %w", err)
	}

	var doc promptDocument
	err = r.collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find prompt: %w", err)
	}

	return r.toEntity(&doc), nil
}

// ListByUserID implements PromptsRepository.ListByUserID
func (r *PromptsRepository) ListByUserID(ctx context.Context, userID string) ([]*entities.Prompt, error) {
	filter := bson.M{"user_id": userID}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list prompts: %w", err)
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

// ListByUserIDAndType implements PromptsRepository.ListByUserIDAndType
func (r *PromptsRepository) ListByUserIDAndType(ctx context.Context, userID string, promptType entities.PromptType) ([]*entities.Prompt, error) {
	filter := bson.M{
		"user_id": userID,
		"type":    string(promptType),
	}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list prompts by type: %w", err)
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

// FindActiveByUserIDAndType implements PromptsRepository.FindActiveByUserIDAndType
func (r *PromptsRepository) FindActiveByUserIDAndType(ctx context.Context, userID string, promptType entities.PromptType) ([]*entities.Prompt, error) {
	filter := bson.M{
		"user_id": userID,
		"type":    string(promptType),
		"active":  true,
	}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find active prompts: %w", err)
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

// FindByUserIDAndStyle implements PromptsRepository.FindByUserIDAndStyle
func (r *PromptsRepository) FindByUserIDAndStyle(ctx context.Context, userID string, styleName string) (*entities.Prompt, error) {
	filter := bson.M{
		"user_id":    userID,
		"type":       string(entities.PromptTypeDrafts),
		"style_name": styleName,
		"active":     true,
	}

	var doc promptDocument
	err := r.collection.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find prompt by style: %w", err)
	}

	return r.toEntity(&doc), nil
}

// Update implements PromptsRepository.Update
func (r *PromptsRepository) Update(ctx context.Context, prompt *entities.Prompt) error {
	oid, err := primitive.ObjectIDFromHex(prompt.ID)
	if err != nil {
		return fmt.Errorf("invalid prompt ID: %w", err)
	}

	update := bson.M{
		"$set": bson.M{
			"type":            string(prompt.Type),
			"style_name":      prompt.StyleName,
			"prompt_template": prompt.PromptTemplate,
			"active":          prompt.Active,
			"updated_at":      prompt.UpdatedAt,
		},
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": oid}, update)
	if err != nil {
		return fmt.Errorf("failed to update prompt: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("prompt not found")
	}

	return nil
}

// Delete implements PromptsRepository.Delete
func (r *PromptsRepository) Delete(ctx context.Context, id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid prompt ID: %w", err)
	}

	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": oid})
	if err != nil {
		return fmt.Errorf("failed to delete prompt: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("prompt not found")
	}

	return nil
}

// CountByUserID implements PromptsRepository.CountByUserID
func (r *PromptsRepository) CountByUserID(ctx context.Context, userID string) (int64, error) {
	count, err := r.collection.CountDocuments(ctx, bson.M{"user_id": userID})
	if err != nil {
		return 0, fmt.Errorf("failed to count prompts: %w", err)
	}

	return count, nil
}
