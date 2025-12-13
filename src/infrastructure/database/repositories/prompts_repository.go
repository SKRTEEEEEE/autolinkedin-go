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
	Name           string             `bson:"name"`
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
		Name:           prompt.Name,
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
		Name:           doc.Name,
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

// FindByName implements PromptsRepository.FindByName
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

// CreateBatch implements PromptsRepository.CreateBatch
func (r *PromptsRepository) CreateBatch(ctx context.Context, prompts []*entities.Prompt) ([]string, error) {
	if len(prompts) == 0 {
		return []string{}, nil
	}

	var documents []interface{}
	for _, prompt := range prompts {
		doc, err := r.toDocument(prompt)
		if err != nil {
			return nil, fmt.Errorf("failed to convert prompt to document: %w", err)
		}
		documents = append(documents, doc)
	}

	// InsertMany for batch creation
	result, err := r.collection.InsertMany(ctx, documents)
	if err != nil {
		return nil, fmt.Errorf("failed to insert prompts batch: %w", err)
	}

	// Convert ObjectIDs to strings
	var ids []string
	for _, id := range result.InsertedIDs {
		if oid, ok := id.(primitive.ObjectID); ok {
			ids = append(ids, oid.Hex())
		}
	}

	return ids, nil
}

// FindOrCreateByName implements PromptsRepository.FindOrCreateByName
func (r *PromptsRepository) FindOrCreateByName(
	ctx context.Context,
	userID string,
	name string,
	promptType entities.PromptType,
	template string,
) (*entities.Prompt, error) {
	// Try to find existing prompt
	prompt, err := r.FindByName(ctx, userID, name)
	if err != nil {
		return nil, fmt.Errorf("failed to find prompt by name: %w", err)
	}

	// If found, return it
	if prompt != nil {
		return prompt, nil
	}

	// Create new prompt if not found
	now := time.Now()
	newPrompt := &entities.Prompt{
		ID:             primitive.NewObjectID().Hex(),
		UserID:         userID,
		Type:           promptType,
		Name:           name,
		StyleName:      name,
		PromptTemplate: template,
		Active:         true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	// Validate the prompt
	if err := newPrompt.Validate(); err != nil {
		return nil, fmt.Errorf("invalid prompt: %w", err)
	}

	// Create the prompt
	promptID, err := r.Create(ctx, newPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to create prompt: %w", err)
	}

	// Get the created prompt with ID
	created, err := r.FindByID(ctx, promptID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve created prompt: %w", err)
	}

	return created, nil
}

// ListActiveByUserIDAndType implements PromptsRepository.ListActiveByUserIDAndType
// This is an alias for FindActiveByUserIDAndType for consistency
func (r *PromptsRepository) ListActiveByUserIDAndType(
	ctx context.Context,
	userID string,
	promptType entities.PromptType,
) ([]*entities.Prompt, error) {
	return r.FindActiveByUserIDAndType(ctx, userID, promptType)
}

// DeactivateByUserIDAndName implements PromptsRepository.DeactivateByUserIDAndName
func (r *PromptsRepository) DeactivateByUserIDAndName(ctx context.Context, userID string, name string) error {
	if userID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}
	if name == "" {
		return fmt.Errorf("prompt name cannot be empty")
	}

	filter := bson.M{
		"user_id": userID,
		"name":    name,
	}

	update := bson.M{
		"$set": bson.M{
			"active":     false,
			"updated_at": time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to deactivate prompt: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("prompt not found for user")
	}

	return nil
}

// Upsert implements PromptsRepository.Upsert
func (r *PromptsRepository) Upsert(ctx context.Context, prompt *entities.Prompt) (string, error) {
	if prompt == nil {
		return "", fmt.Errorf("prompt cannot be nil")
	}

	if prompt.ID != "" {
		// Try to update first
		var oid primitive.ObjectID
		var err error

		oid, err = primitive.ObjectIDFromHex(prompt.ID)
		if err != nil {
			return "", fmt.Errorf("invalid prompt ID: %w", err)
		}

		// Check if prompt exists
		filter := bson.M{"_id": oid}
		count, err := r.collection.CountDocuments(ctx, filter)
		if err != nil {
			return "", fmt.Errorf("failed to check prompt existence: %w", err)
		}

		if count > 0 {
			// Update existing prompt
			update := bson.M{
				"$set": bson.M{
					"user_id":         prompt.UserID,
					"type":            string(prompt.Type),
					"name":            prompt.Name,
					"style_name":      prompt.StyleName,
					"prompt_template": prompt.PromptTemplate,
					"active":          prompt.Active,
					"updated_at":      prompt.UpdatedAt,
				},
			}

			result, err := r.collection.UpdateOne(ctx, filter, update)
			if err != nil {
				return "", fmt.Errorf("failed to update prompt: %w", err)
			}

			if result.MatchedCount == 0 {
				return "", fmt.Errorf("prompt not found for update")
			}

			return prompt.ID, nil
		}
	}

	// Create new prompt (either ID is empty or prompt doesn't exist)
	if prompt.ID == "" {
		prompt.ID = primitive.NewObjectID().Hex()
	}

	// Ensure timestamps are set
	if prompt.CreatedAt.IsZero() {
		prompt.CreatedAt = time.Now()
	}
	if prompt.UpdatedAt.IsZero() {
		prompt.UpdatedAt = time.Now()
	}

	return r.Create(ctx, prompt)
}
