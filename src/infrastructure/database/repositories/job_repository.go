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

// jobRepository implements the JobRepository interface for MongoDB
type jobRepository struct {
	*database.BaseRepository
	collection *mongo.Collection
}

// NewJobRepository creates a new MongoDB job repository
func NewJobRepository(collection *mongo.Collection) interfaces.JobRepository {
	return &jobRepository{
		BaseRepository: database.NewBaseRepository(collection),
		collection:     collection,
	}
}

// jobDocument represents the MongoDB document structure for Job
type jobDocument struct {
	ID          primitive.ObjectID   `bson:"_id,omitempty"`
	UserID      primitive.ObjectID   `bson:"user_id"`
	Type        string               `bson:"type"`
	Status      string               `bson:"status"`
	IdeaID      *primitive.ObjectID  `bson:"idea_id,omitempty"`
	DraftIDs    []primitive.ObjectID `bson:"draft_ids"`
	Error       string               `bson:"error"`
	CreatedAt   primitive.DateTime   `bson:"created_at"`
	UpdatedAt   primitive.DateTime   `bson:"updated_at"`
	StartedAt   *primitive.DateTime  `bson:"started_at,omitempty"`
	CompletedAt *primitive.DateTime  `bson:"completed_at,omitempty"`
}

// toDocument converts a Job entity to a MongoDB document
func (r *jobRepository) toDocument(job *entities.Job) (*jobDocument, error) {
	if job == nil {
		return nil, fmt.Errorf("job cannot be nil")
	}

	userObjectID, err := primitive.ObjectIDFromHex(job.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	doc := &jobDocument{
		UserID:    userObjectID,
		Type:      string(job.Type),
		Status:    string(job.Status),
		Error:     job.Error,
		CreatedAt: primitive.NewDateTimeFromTime(job.CreatedAt),
		UpdatedAt: primitive.NewDateTimeFromTime(job.UpdatedAt),
	}

	// Only set ID if it's valid
	if job.ID != "" {
		objectID, err := primitive.ObjectIDFromHex(job.ID)
		if err != nil {
			// If job.ID is a UUID, create a new ObjectID
			doc.ID = primitive.NewObjectID()
		} else {
			doc.ID = objectID
		}
	}

	// Set idea ID if present
	if job.IdeaID != nil && *job.IdeaID != "" {
		ideaObjectID, err := primitive.ObjectIDFromHex(*job.IdeaID)
		if err != nil {
			return nil, fmt.Errorf("invalid idea ID: %w", err)
		}
		doc.IdeaID = &ideaObjectID
	}

	// Convert draft IDs
	if len(job.DraftIDs) > 0 {
		doc.DraftIDs = make([]primitive.ObjectID, 0, len(job.DraftIDs))
		for _, draftID := range job.DraftIDs {
			draftObjectID, err := primitive.ObjectIDFromHex(draftID)
			if err != nil {
				return nil, fmt.Errorf("invalid draft ID %s: %w", draftID, err)
			}
			doc.DraftIDs = append(doc.DraftIDs, draftObjectID)
		}
	}

	// Set started timestamp if present
	if job.StartedAt != nil {
		startedAt := primitive.NewDateTimeFromTime(*job.StartedAt)
		doc.StartedAt = &startedAt
	}

	// Set completed timestamp if present
	if job.CompletedAt != nil {
		completedAt := primitive.NewDateTimeFromTime(*job.CompletedAt)
		doc.CompletedAt = &completedAt
	}

	return doc, nil
}

// toEntity converts a MongoDB document to a Job entity
func (r *jobRepository) toEntity(doc *jobDocument) *entities.Job {
	if doc == nil {
		return nil
	}

	job := &entities.Job{
		ID:        doc.ID.Hex(),
		UserID:    doc.UserID.Hex(),
		Type:      entities.JobType(doc.Type),
		Status:    entities.JobStatus(doc.Status),
		Error:     doc.Error,
		CreatedAt: doc.CreatedAt.Time(),
		UpdatedAt: doc.UpdatedAt.Time(),
	}

	// Set idea ID if present
	if doc.IdeaID != nil {
		ideaID := doc.IdeaID.Hex()
		job.IdeaID = &ideaID
	}

	// Convert draft IDs
	if len(doc.DraftIDs) > 0 {
		job.DraftIDs = make([]string, len(doc.DraftIDs))
		for i, draftID := range doc.DraftIDs {
			job.DraftIDs[i] = draftID.Hex()
		}
	}

	// Set started timestamp if present
	if doc.StartedAt != nil {
		startedAt := doc.StartedAt.Time()
		job.StartedAt = &startedAt
	}

	// Set completed timestamp if present
	if doc.CompletedAt != nil {
		completedAt := doc.CompletedAt.Time()
		job.CompletedAt = &completedAt
	}

	return job
}

// Create creates a new job
func (r *jobRepository) Create(ctx context.Context, job *entities.Job) (string, error) {
	if err := job.Validate(); err != nil {
		return "", fmt.Errorf("validation failed: %w", err)
	}

	doc, err := r.toDocument(job)
	if err != nil {
		return "", fmt.Errorf("failed to convert to document: %w", err)
	}

	result, err := r.collection.InsertOne(ctx, doc)
	if err != nil {
		return "", fmt.Errorf("failed to insert job: %w", err)
	}

	insertedID := result.InsertedID.(primitive.ObjectID)
	return insertedID.Hex(), nil
}

// FindByID retrieves a job by its unique ID
func (r *jobRepository) FindByID(ctx context.Context, jobID string) (*entities.Job, error) {
	objectID, err := primitive.ObjectIDFromHex(jobID)
	if err != nil {
		return nil, fmt.Errorf("invalid job ID: %w", err)
	}

	var doc jobDocument
	err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find job: %w", err)
	}

	return r.toEntity(&doc), nil
}

// Update updates an existing job
func (r *jobRepository) Update(ctx context.Context, job *entities.Job) error {
	if err := job.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	doc, err := r.toDocument(job)
	if err != nil {
		return fmt.Errorf("failed to convert to document: %w", err)
	}

	// Update timestamp
	doc.UpdatedAt = primitive.NewDateTimeFromTime(time.Now())

	filter := bson.M{"_id": doc.ID}
	update := bson.M{
		"$set": bson.M{
			"status":       doc.Status,
			"draft_ids":    doc.DraftIDs,
			"error":        doc.Error,
			"updated_at":   doc.UpdatedAt,
			"started_at":   doc.StartedAt,
			"completed_at": doc.CompletedAt,
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("job not found: %s", job.ID)
	}

	return nil
}

// ListByUserID retrieves all jobs for a specific user
func (r *jobRepository) ListByUserID(ctx context.Context, userID string, limit int) ([]*entities.Job, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	filter := bson.M{"user_id": userObjectID}

	// Set options
	opts := options.Find()
	opts.SetSort(bson.D{{Key: "created_at", Value: -1}}) // Most recent first

	if limit > 0 {
		opts.SetLimit(int64(limit))
	}

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to query jobs: %w", err)
	}
	defer cursor.Close(ctx)

	var jobs []*entities.Job
	for cursor.Next(ctx) {
		var doc jobDocument
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("failed to decode job: %w", err)
		}
		jobs = append(jobs, r.toEntity(&doc))
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return jobs, nil
}
