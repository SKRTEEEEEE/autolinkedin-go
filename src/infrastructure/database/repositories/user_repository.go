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
)

// userRepository implements the UserRepository interface for MongoDB
type userRepository struct {
	*database.BaseRepository
	collection *mongo.Collection
}

// NewUserRepository creates a new MongoDB user repository
func NewUserRepository(collection *mongo.Collection) interfaces.UserRepository {
	return &userRepository{
		BaseRepository: database.NewBaseRepository(collection),
		collection:     collection,
	}
}

// userDocument represents the MongoDB document structure for User
type userDocument struct {
	ID            primitive.ObjectID         `bson:"_id,omitempty"`
	Email         string                     `bson:"email"`
	LinkedInToken string                     `bson:"linkedin_token"`
	APIKeys       map[string]string          `bson:"api_keys"`
	Configuration map[string]interface{}     `bson:"configuration"`
	CreatedAt     primitive.DateTime         `bson:"created_at"`
	UpdatedAt     primitive.DateTime         `bson:"updated_at"`
	Active        bool                       `bson:"active"`
}

// toDocument converts a User entity to a MongoDB document
func (r *userRepository) toDocument(user *entities.User) (*userDocument, error) {
	if user == nil {
		return nil, fmt.Errorf("user cannot be nil")
	}

	doc := &userDocument{
		Email:         user.Email,
		LinkedInToken: user.LinkedInToken,
		APIKeys:       user.APIKeys,
		Configuration: user.Configuration,
		CreatedAt:     primitive.NewDateTimeFromTime(user.CreatedAt),
		UpdatedAt:     primitive.NewDateTimeFromTime(user.UpdatedAt),
		Active:        user.Active,
	}

	// Only set ID if it's valid
	if user.ID != "" {
		objectID, err := primitive.ObjectIDFromHex(user.ID)
		if err != nil {
			return nil, fmt.Errorf("invalid user ID: %w", err)
		}
		doc.ID = objectID
	}

	return doc, nil
}

// toEntity converts a MongoDB document to a User entity
func (r *userRepository) toEntity(doc *userDocument) *entities.User {
	if doc == nil {
		return nil
	}

	return &entities.User{
		ID:            doc.ID.Hex(),
		Email:         doc.Email,
		LinkedInToken: doc.LinkedInToken,
		APIKeys:       doc.APIKeys,
		Configuration: doc.Configuration,
		CreatedAt:     doc.CreatedAt.Time(),
		UpdatedAt:     doc.UpdatedAt.Time(),
		Active:        doc.Active,
	}
}

// Create creates a new user in the database
func (r *userRepository) Create(ctx context.Context, user *entities.User) (string, error) {
	if user == nil {
		return "", database.ErrInvalidEntity
	}

	// Validate user before persisting
	if err := user.Validate(); err != nil {
		return "", fmt.Errorf("user validation failed: %w", err)
	}

	doc, err := r.toDocument(user)
	if err != nil {
		return "", err
	}

	// Ensure email uniqueness with index
	result, err := r.collection.InsertOne(ctx, doc)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return "", database.ErrEntityAlreadyExists
		}
		return "", fmt.Errorf("failed to create user: %w", err)
	}

	insertedID, ok := result.InsertedID.(primitive.ObjectID)
	if !ok {
		return "", fmt.Errorf("failed to convert inserted ID to ObjectID")
	}

	return insertedID.Hex(), nil
}

// FindByID retrieves a user by their ID
func (r *userRepository) FindByID(ctx context.Context, userID string) (*entities.User, error) {
	if userID == "" {
		return nil, database.ErrInvalidID
	}

	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, database.ErrInvalidID
	}

	var doc userDocument
	filter := bson.M{"_id": objectID}

	err = r.collection.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, database.ErrEntityNotFound
		}
		return nil, fmt.Errorf("failed to find user by ID: %w", err)
	}

	return r.toEntity(&doc), nil
}

// FindByEmail retrieves a user by their email address
func (r *userRepository) FindByEmail(ctx context.Context, email string) (*entities.User, error) {
	if email == "" {
		return nil, fmt.Errorf("email cannot be empty")
	}

	var doc userDocument
	filter := bson.M{"email": email}

	err := r.collection.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, database.ErrEntityNotFound
		}
		return nil, fmt.Errorf("failed to find user by email: %w", err)
	}

	return r.toEntity(&doc), nil
}

// Update updates user information
func (r *userRepository) Update(ctx context.Context, userID string, updates map[string]interface{}) error {
	if userID == "" {
		return database.ErrInvalidID
	}

	if len(updates) == 0 {
		return database.ErrEmptyUpdate
	}

	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return database.ErrInvalidID
	}

	// Add updated timestamp
	updates["updated_at"] = primitive.NewDateTimeFromTime(time.Now())

	filter := bson.M{"_id": objectID}
	update := bson.M{"$set": updates}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	if result.MatchedCount == 0 {
		return database.ErrEntityNotFound
	}

	return nil
}

// UpdateLinkedInToken updates the LinkedIn access token for a user
func (r *userRepository) UpdateLinkedInToken(ctx context.Context, userID string, token string) error {
	if userID == "" {
		return database.ErrInvalidID
	}

	if token == "" {
		return fmt.Errorf("LinkedIn token cannot be empty")
	}

	updates := map[string]interface{}{
		"linkedin_token": token,
	}

	return r.Update(ctx, userID, updates)
}

// Delete removes a user from the database
func (r *userRepository) Delete(ctx context.Context, userID string) error {
	if userID == "" {
		return database.ErrInvalidID
	}

	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return database.ErrInvalidID
	}

	filter := bson.M{"_id": objectID}
	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	if result.DeletedCount == 0 {
		return database.ErrEntityNotFound
	}

	return nil
}
