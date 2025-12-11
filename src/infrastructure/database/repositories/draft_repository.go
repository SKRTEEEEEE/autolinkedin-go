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

// draftRepository implements the DraftRepository interface for MongoDB
type draftRepository struct {
	*database.BaseRepository
	collection *mongo.Collection
}

// NewDraftRepository creates a new MongoDB draft repository
func NewDraftRepository(collection *mongo.Collection) interfaces.DraftRepository {
	return &draftRepository{
		BaseRepository: database.NewBaseRepository(collection),
		collection:     collection,
	}
}

// refinementEntryDocument represents the MongoDB document structure for RefinementEntry
type refinementEntryDocument struct {
	Timestamp primitive.DateTime `bson:"timestamp"`
	Prompt    string             `bson:"prompt"`
	Content   string             `bson:"content"`
	Version   int                `bson:"version"`
}

// draftDocument represents the MongoDB document structure for Draft
type draftDocument struct {
	ID                primitive.ObjectID        `bson:"_id,omitempty"`
	UserID            primitive.ObjectID        `bson:"user_id"`
	IdeaID            *primitive.ObjectID       `bson:"idea_id,omitempty"`
	Type              string                    `bson:"type"`
	Title             string                    `bson:"title"`
	Content           string                    `bson:"content"`
	Status            string                    `bson:"status"`
	RefinementHistory []refinementEntryDocument `bson:"refinement_history"`
	PublishedAt       *primitive.DateTime       `bson:"published_at,omitempty"`
	LinkedInPostID    string                    `bson:"linkedin_post_id"`
	Metadata          map[string]interface{}    `bson:"metadata"`
	CreatedAt         primitive.DateTime        `bson:"created_at"`
	UpdatedAt         primitive.DateTime        `bson:"updated_at"`
}

// toDocument converts a Draft entity to a MongoDB document
func (r *draftRepository) toDocument(draft *entities.Draft) (*draftDocument, error) {
	if draft == nil {
		return nil, fmt.Errorf("draft cannot be nil")
	}

	userObjectID, err := primitive.ObjectIDFromHex(draft.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	doc := &draftDocument{
		UserID:         userObjectID,
		Type:           string(draft.Type),
		Title:          draft.Title,
		Content:        draft.Content,
		Status:         string(draft.Status),
		LinkedInPostID: draft.LinkedInPostID,
		Metadata:       draft.Metadata,
		CreatedAt:      primitive.NewDateTimeFromTime(draft.CreatedAt),
		UpdatedAt:      primitive.NewDateTimeFromTime(draft.UpdatedAt),
	}

	// Only set ID if it's valid
	if draft.ID != "" {
		objectID, err := primitive.ObjectIDFromHex(draft.ID)
		if err != nil {
			return nil, fmt.Errorf("invalid draft ID: %w", err)
		}
		doc.ID = objectID
	}

	// Set idea ID if present
	if draft.IdeaID != nil && *draft.IdeaID != "" {
		ideaObjectID, err := primitive.ObjectIDFromHex(*draft.IdeaID)
		if err != nil {
			return nil, fmt.Errorf("invalid idea ID: %w", err)
		}
		doc.IdeaID = &ideaObjectID
	}

	// Convert refinement history
	if len(draft.RefinementHistory) > 0 {
		doc.RefinementHistory = make([]refinementEntryDocument, len(draft.RefinementHistory))
		for i, entry := range draft.RefinementHistory {
			doc.RefinementHistory[i] = refinementEntryDocument{
				Timestamp: primitive.NewDateTimeFromTime(entry.Timestamp),
				Prompt:    entry.Prompt,
				Content:   entry.Content,
				Version:   entry.Version,
			}
		}
	}

	// Set published timestamp if present
	if draft.PublishedAt != nil {
		publishedAt := primitive.NewDateTimeFromTime(*draft.PublishedAt)
		doc.PublishedAt = &publishedAt
	}

	return doc, nil
}

// toEntity converts a MongoDB document to a Draft entity
func (r *draftRepository) toEntity(doc *draftDocument) *entities.Draft {
	if doc == nil {
		return nil
	}

	draft := &entities.Draft{
		ID:             doc.ID.Hex(),
		UserID:         doc.UserID.Hex(),
		Type:           entities.DraftType(doc.Type),
		Title:          doc.Title,
		Content:        doc.Content,
		Status:         entities.DraftStatus(doc.Status),
		LinkedInPostID: doc.LinkedInPostID,
		Metadata:       doc.Metadata,
		CreatedAt:      doc.CreatedAt.Time(),
		UpdatedAt:      doc.UpdatedAt.Time(),
	}

	// Set idea ID if present
	if doc.IdeaID != nil {
		ideaID := doc.IdeaID.Hex()
		draft.IdeaID = &ideaID
	}

	// Convert refinement history
	if len(doc.RefinementHistory) > 0 {
		draft.RefinementHistory = make([]entities.RefinementEntry, len(doc.RefinementHistory))
		for i, entry := range doc.RefinementHistory {
			draft.RefinementHistory[i] = entities.RefinementEntry{
				Timestamp: entry.Timestamp.Time(),
				Prompt:    entry.Prompt,
				Content:   entry.Content,
				Version:   entry.Version,
			}
		}
	}

	// Set published timestamp if present
	if doc.PublishedAt != nil {
		publishedAt := doc.PublishedAt.Time()
		draft.PublishedAt = &publishedAt
	}

	return draft
}

// Create creates a new draft in the database
func (r *draftRepository) Create(ctx context.Context, draft *entities.Draft) (string, error) {
	if draft == nil {
		return "", database.ErrInvalidEntity
	}

	// Validate draft before persisting
	if err := draft.Validate(); err != nil {
		return "", fmt.Errorf("draft validation failed: %w", err)
	}

	doc, err := r.toDocument(draft)
	if err != nil {
		return "", err
	}

	result, err := r.collection.InsertOne(ctx, doc)
	if err != nil {
		return "", fmt.Errorf("failed to create draft: %w", err)
	}

	insertedID, ok := result.InsertedID.(primitive.ObjectID)
	if !ok {
		return "", fmt.Errorf("failed to convert inserted ID to ObjectID")
	}

	return insertedID.Hex(), nil
}

// FindByID retrieves a draft by its ID
func (r *draftRepository) FindByID(ctx context.Context, draftID string) (*entities.Draft, error) {
	if draftID == "" {
		return nil, database.ErrInvalidID
	}

	objectID, err := primitive.ObjectIDFromHex(draftID)
	if err != nil {
		return nil, database.ErrInvalidID
	}

	var doc draftDocument
	filter := bson.M{"_id": objectID}

	err = r.collection.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, database.ErrEntityNotFound
		}
		return nil, fmt.Errorf("failed to find draft by ID: %w", err)
	}

	return r.toEntity(&doc), nil
}

// Update updates draft information
func (r *draftRepository) Update(ctx context.Context, draftID string, updates map[string]interface{}) error {
	if draftID == "" {
		return database.ErrInvalidID
	}

	if len(updates) == 0 {
		return database.ErrEmptyUpdate
	}

	objectID, err := primitive.ObjectIDFromHex(draftID)
	if err != nil {
		return database.ErrInvalidID
	}

	// Add updated timestamp
	updates["updated_at"] = primitive.NewDateTimeFromTime(time.Now())

	filter := bson.M{"_id": objectID}
	update := bson.M{"$set": updates}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update draft: %w", err)
	}

	if result.MatchedCount == 0 {
		return database.ErrEntityNotFound
	}

	return nil
}

// Delete removes a draft from the database
func (r *draftRepository) Delete(ctx context.Context, draftID string) error {
	if draftID == "" {
		return database.ErrInvalidID
	}

	objectID, err := primitive.ObjectIDFromHex(draftID)
	if err != nil {
		return database.ErrInvalidID
	}

	filter := bson.M{"_id": objectID}
	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete draft: %w", err)
	}

	if result.DeletedCount == 0 {
		return database.ErrEntityNotFound
	}

	return nil
}

// ListByUserID retrieves drafts for a user with optional filtering
func (r *draftRepository) ListByUserID(ctx context.Context, userID string, status entities.DraftStatus, draftType entities.DraftType) ([]*entities.Draft, error) {
	if userID == "" {
		return nil, database.ErrInvalidID
	}

	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, database.ErrInvalidID
	}

	// Build filter
	filter := bson.M{"user_id": userObjectID}

	// Add status filter if provided
	if status != "" {
		filter["status"] = string(status)
	}

	// Add type filter if provided
	if draftType != "" {
		filter["type"] = string(draftType)
	}

	// Sort by creation date (newest first)
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list drafts: %w", err)
	}
	defer cursor.Close(ctx)

	var drafts []*entities.Draft
	for cursor.Next(ctx) {
		var doc draftDocument
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("failed to decode draft document: %w", err)
		}
		drafts = append(drafts, r.toEntity(&doc))
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return drafts, nil
}

// UpdateStatus updates the status of a draft
func (r *draftRepository) UpdateStatus(ctx context.Context, draftID string, status entities.DraftStatus) error {
	if draftID == "" {
		return database.ErrInvalidID
	}

	if status == "" {
		return fmt.Errorf("status cannot be empty")
	}

	updates := map[string]interface{}{
		"status": string(status),
	}

	return r.Update(ctx, draftID, updates)
}

// AppendRefinement adds a refinement entry to a draft
func (r *draftRepository) AppendRefinement(ctx context.Context, draftID string, entry entities.RefinementEntry) error {
	if draftID == "" {
		return database.ErrInvalidID
	}

	objectID, err := primitive.ObjectIDFromHex(draftID)
	if err != nil {
		return database.ErrInvalidID
	}

	// Convert entry to document
	entryDoc := refinementEntryDocument{
		Timestamp: primitive.NewDateTimeFromTime(entry.Timestamp),
		Prompt:    entry.Prompt,
		Content:   entry.Content,
		Version:   entry.Version,
	}

	// Use $push to append to refinement history array
	filter := bson.M{"_id": objectID}
	update := bson.M{
		"$push": bson.M{"refinement_history": entryDoc},
		"$set": bson.M{
			"content":    entry.Content,
			"status":     string(entities.DraftStatusRefined),
			"updated_at": primitive.NewDateTimeFromTime(time.Now()),
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to append refinement: %w", err)
	}

	if result.MatchedCount == 0 {
		return database.ErrEntityNotFound
	}

	return nil
}

// FindReadyForPublishing retrieves drafts ready to be published
func (r *draftRepository) FindReadyForPublishing(ctx context.Context, userID string) ([]*entities.Draft, error) {
	if userID == "" {
		return nil, database.ErrInvalidID
	}

	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, database.ErrInvalidID
	}

	// Drafts are ready for publishing if they are in DRAFT or REFINED status
	filter := bson.M{
		"user_id": userObjectID,
		"status": bson.M{
			"$in": []string{
				string(entities.DraftStatusDraft),
				string(entities.DraftStatusRefined),
			},
		},
	}

	// Sort by creation date (oldest first, FIFO)
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find drafts ready for publishing: %w", err)
	}
	defer cursor.Close(ctx)

	var drafts []*entities.Draft
	for cursor.Next(ctx) {
		var doc draftDocument
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("failed to decode draft document: %w", err)
		}
		drafts = append(drafts, r.toEntity(&doc))
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return drafts, nil
}
