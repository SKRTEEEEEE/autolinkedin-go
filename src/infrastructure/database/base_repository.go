package database

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	// ErrEntityNotFound is returned when entity is not found
	ErrEntityNotFound = errors.New("entity not found")
	// ErrEntityAlreadyExists is returned when entity already exists
	ErrEntityAlreadyExists = errors.New("entity already exists")
	// ErrInvalidEntity is returned when entity is invalid
	ErrInvalidEntity = errors.New("invalid entity")
	// ErrEmptyUpdate is returned when update data is empty
	ErrEmptyUpdate = errors.New("update data cannot be empty")
	// ErrInvalidID is returned when ID format is invalid
	ErrInvalidID = errors.New("invalid ID format")
	// ErrInvalidPagination is returned when pagination parameters are invalid
	ErrInvalidPagination = errors.New("invalid pagination parameters")
)

// BaseRepository provides common CRUD operations for MongoDB collections
type BaseRepository struct {
	collection *mongo.Collection
}

// NewBaseRepository creates a new base repository
func NewBaseRepository(collection *mongo.Collection) *BaseRepository {
	return &BaseRepository{
		collection: collection,
	}
}

// Create inserts a new entity into the collection
func (r *BaseRepository) Create(ctx context.Context, entity interface{}) (string, error) {
	if entity == nil {
		return "", ErrInvalidEntity
	}

	result, err := r.collection.InsertOne(ctx, entity)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return "", ErrEntityAlreadyExists
		}
		return "", fmt.Errorf("failed to insert entity: %w", err)
	}

	id, ok := result.InsertedID.(primitive.ObjectID)
	if !ok {
		return "", fmt.Errorf("failed to convert inserted ID to ObjectID")
	}

	return id.Hex(), nil
}

// FindByID retrieves an entity by its ID
func (r *BaseRepository) FindByID(ctx context.Context, id string, result interface{}) error {
	if id == "" {
		return ErrInvalidID
	}

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return ErrInvalidID
	}

	filter := bson.M{"_id": objectID}
	err = r.collection.FindOne(ctx, filter).Decode(result)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return ErrEntityNotFound
		}
		return fmt.Errorf("failed to find entity by ID: %w", err)
	}

	return nil
}

// Update updates an entity by its ID
func (r *BaseRepository) Update(ctx context.Context, id string, updates bson.M) error {
	if id == "" {
		return ErrInvalidID
	}

	if len(updates) == 0 {
		return ErrEmptyUpdate
	}

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return ErrInvalidID
	}

	filter := bson.M{"_id": objectID}
	update := bson.M{"$set": updates}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update entity: %w", err)
	}

	if result.MatchedCount == 0 {
		return ErrEntityNotFound
	}

	return nil
}

// Delete removes an entity by its ID
func (r *BaseRepository) Delete(ctx context.Context, id string) error {
	if id == "" {
		return ErrInvalidID
	}

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return ErrInvalidID
	}

	filter := bson.M{"_id": objectID}
	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete entity: %w", err)
	}

	if result.DeletedCount == 0 {
		return ErrEntityNotFound
	}

	return nil
}

// FindAll retrieves all entities matching the filter
func (r *BaseRepository) FindAll(ctx context.Context, filter bson.M, results interface{}) error {
	if filter == nil {
		filter = bson.M{}
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to find entities: %w", err)
	}
	defer cursor.Close(ctx)

	if err := cursor.All(ctx, results); err != nil {
		return fmt.Errorf("failed to decode entities: %w", err)
	}

	return nil
}

// FindWithPagination retrieves entities with pagination
func (r *BaseRepository) FindWithPagination(ctx context.Context, filter bson.M, page, pageSize int, results interface{}) error {
	if page <= 0 {
		return ErrInvalidPagination
	}

	if pageSize <= 0 {
		return ErrInvalidPagination
	}

	if filter == nil {
		filter = bson.M{}
	}

	skip := int64((page - 1) * pageSize)
	limit := int64(pageSize)

	opts := options.Find().
		SetSkip(skip).
		SetLimit(limit)

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return fmt.Errorf("failed to find entities with pagination: %w", err)
	}
	defer cursor.Close(ctx)

	if err := cursor.All(ctx, results); err != nil {
		return fmt.Errorf("failed to decode entities: %w", err)
	}

	return nil
}

// Count returns the number of documents matching the filter
func (r *BaseRepository) Count(ctx context.Context, filter bson.M) (int64, error) {
	if filter == nil {
		filter = bson.M{}
	}

	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to count documents: %w", err)
	}

	return count, nil
}

// BulkInsert inserts multiple entities at once
func (r *BaseRepository) BulkInsert(ctx context.Context, entities []interface{}) ([]string, error) {
	if len(entities) == 0 {
		return nil, ErrInvalidEntity
	}

	result, err := r.collection.InsertMany(ctx, entities)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, ErrEntityAlreadyExists
		}
		return nil, fmt.Errorf("failed to bulk insert entities: %w", err)
	}

	ids := make([]string, len(result.InsertedIDs))
	for i, id := range result.InsertedIDs {
		objID, ok := id.(primitive.ObjectID)
		if !ok {
			return nil, fmt.Errorf("failed to convert inserted ID to ObjectID at index %d", i)
		}
		ids[i] = objID.Hex()
	}

	return ids, nil
}

// BulkUpdate updates multiple entities at once
func (r *BaseRepository) BulkUpdate(ctx context.Context, filter bson.M, updates bson.M) (int64, error) {
	if len(updates) == 0 {
		return 0, ErrEmptyUpdate
	}

	if filter == nil {
		filter = bson.M{}
	}

	update := bson.M{"$set": updates}
	result, err := r.collection.UpdateMany(ctx, filter, update)
	if err != nil {
		return 0, fmt.Errorf("failed to bulk update entities: %w", err)
	}

	return result.ModifiedCount, nil
}

// BulkDelete deletes multiple entities matching the filter
func (r *BaseRepository) BulkDelete(ctx context.Context, filter bson.M) (int64, error) {
	if filter == nil {
		return 0, fmt.Errorf("filter cannot be empty for bulk delete")
	}

	result, err := r.collection.DeleteMany(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to bulk delete entities: %w", err)
	}

	return result.DeletedCount, nil
}

// Exists checks if an entity with the given filter exists
func (r *BaseRepository) Exists(ctx context.Context, filter bson.M) (bool, error) {
	if filter == nil {
		filter = bson.M{}
	}

	count, err := r.collection.CountDocuments(ctx, filter, options.Count().SetLimit(1))
	if err != nil {
		return false, fmt.Errorf("failed to check existence: %w", err)
	}

	return count > 0, nil
}

// FindOne retrieves a single entity matching the filter
func (r *BaseRepository) FindOne(ctx context.Context, filter bson.M, result interface{}) error {
	if filter == nil {
		filter = bson.M{}
	}

	err := r.collection.FindOne(ctx, filter).Decode(result)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return ErrEntityNotFound
		}
		return fmt.Errorf("failed to find entity: %w", err)
	}

	return nil
}

// Aggregate executes an aggregation pipeline
func (r *BaseRepository) Aggregate(ctx context.Context, pipeline interface{}, results interface{}) error {
	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return fmt.Errorf("failed to execute aggregation: %w", err)
	}
	defer cursor.Close(ctx)

	if err := cursor.All(ctx, results); err != nil {
		return fmt.Errorf("failed to decode aggregation results: %w", err)
	}

	return nil
}

// UpdateByFilter updates entities matching the filter
func (r *BaseRepository) UpdateByFilter(ctx context.Context, filter bson.M, updates bson.M) (int64, error) {
	if len(updates) == 0 {
		return 0, ErrEmptyUpdate
	}

	if filter == nil {
		filter = bson.M{}
	}

	update := bson.M{"$set": updates}
	result, err := r.collection.UpdateMany(ctx, filter, update)
	if err != nil {
		return 0, fmt.Errorf("failed to update by filter: %w", err)
	}

	return result.ModifiedCount, nil
}

// DeleteByFilter deletes entities matching the filter
func (r *BaseRepository) DeleteByFilter(ctx context.Context, filter bson.M) (int64, error) {
	if filter == nil {
		return 0, fmt.Errorf("filter cannot be empty for delete by filter")
	}

	result, err := r.collection.DeleteMany(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to delete by filter: %w", err)
	}

	return result.DeletedCount, nil
}
