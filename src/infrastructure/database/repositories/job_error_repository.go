package repositories

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/linkgen-ai/backend/src/domain/entities"
	"github.com/linkgen-ai/backend/src/domain/interfaces"
	"github.com/linkgen-ai/backend/src/infrastructure/database"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

const maxJobErrorBlobLength = 20000

// jobErrorRepository implements JobErrorRepository for MongoDB
type jobErrorRepository struct {
	*database.BaseRepository
	collection *mongo.Collection
}

// NewJobErrorRepository creates a new MongoDB job error repository
func NewJobErrorRepository(collection *mongo.Collection) interfaces.JobErrorRepository {
	return &jobErrorRepository{
		BaseRepository: database.NewBaseRepository(collection),
		collection:     collection,
	}
}

// jobErrorDocument represents the Mongo document
type jobErrorDocument struct {
	ID          primitive.ObjectID     `bson:"_id,omitempty"`
	JobID       string                 `bson:"job_id"`
	UserID      primitive.ObjectID     `bson:"user_id"`
	IdeaID      *primitive.ObjectID    `bson:"idea_id,omitempty"`
	Stage       string                 `bson:"stage"`
	Error       string                 `bson:"error"`
	RawResponse string                 `bson:"raw_response,omitempty"`
	Prompt      string                 `bson:"prompt,omitempty"`
	Attempt     int                    `bson:"attempt"`
	Metadata    map[string]interface{} `bson:"metadata,omitempty"`
	CreatedAt   primitive.DateTime     `bson:"created_at"`
}

// Create persists a job error document
func (r *jobErrorRepository) Create(ctx context.Context, jobError *entities.JobError) (string, error) {
	if jobError == nil {
		return "", fmt.Errorf("job error cannot be nil")
	}

	if err := jobError.Validate(); err != nil {
		return "", err
	}

	doc, err := r.toDocument(jobError)
	if err != nil {
		return "", err
	}

	result, err := r.collection.InsertOne(ctx, doc)
	if err != nil {
		return "", fmt.Errorf("failed to insert job error: %w", err)
	}

	insertedID := result.InsertedID.(primitive.ObjectID)
	return insertedID.Hex(), nil
}

func (r *jobErrorRepository) toDocument(jobError *entities.JobError) (*jobErrorDocument, error) {
	userObjectID, err := primitive.ObjectIDFromHex(jobError.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	var ideaObjectID *primitive.ObjectID
	if jobError.IdeaID != nil && strings.TrimSpace(*jobError.IdeaID) != "" {
		objID, err := primitive.ObjectIDFromHex(*jobError.IdeaID)
		if err != nil {
			return nil, fmt.Errorf("invalid idea ID: %w", err)
		}
		ideaObjectID = &objID
	}

	return &jobErrorDocument{
		JobID:       jobError.JobID,
		UserID:      userObjectID,
		IdeaID:      ideaObjectID,
		Stage:       string(jobError.Stage),
		Error:       truncateJobErrorBlob(jobError.Error),
		RawResponse: truncateJobErrorBlob(jobError.RawResponse),
		Prompt:      truncateJobErrorBlob(jobError.Prompt),
		Attempt:     jobError.Attempt,
		Metadata:    jobError.Metadata,
		CreatedAt:   primitive.NewDateTimeFromTime(jobError.CreatedAt),
	}, nil
}

func truncateJobErrorBlob(content string) string {
	if content == "" {
		return content
	}

	if utf8.RuneCountInString(content) <= maxJobErrorBlobLength {
		return content
	}

	runes := []rune(content)
	return string(runes[:maxJobErrorBlobLength])
}
