package database

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// Collection name constants
const (
	CollectionUsers      = "users"
	CollectionTopics     = "topics"
	CollectionIdeas      = "ideas"
	CollectionDrafts     = "drafts"
	CollectionUserTopics = "userTopics"
	CollectionPrompts    = "prompts"
)

// IndexDefinition represents a MongoDB index
type IndexDefinition struct {
	Collection string
	Keys       bson.D
	Options    *options.IndexOptions
}

// GetAllIndexDefinitions returns all index definitions for the application
func GetAllIndexDefinitions() []IndexDefinition {
	return []IndexDefinition{
		// Users collection indexes
		{
			Collection: CollectionUsers,
			Keys:       bson.D{{Key: "email", Value: 1}},
			Options:    options.Index().SetUnique(true).SetName("email_unique"),
		},
		{
			Collection: CollectionUsers,
			Keys:       bson.D{{Key: "linkedin_id", Value: 1}},
			Options:    options.Index().SetUnique(true).SetName("linkedin_id_unique"),
		},
		// Ideas collection indexes
		{
			Collection: CollectionIdeas,
			Keys:       bson.D{{Key: "user_id", Value: 1}},
			Options:    options.Index().SetName("user_id_idx"),
		},
		{
			Collection: CollectionIdeas,
			Keys:       bson.D{{Key: "created_at", Value: -1}},
			Options:    options.Index().SetName("created_at_desc_idx"),
		},
		{
			Collection: CollectionIdeas,
			Keys:       bson.D{{Key: "user_id", Value: 1}, {Key: "topic", Value: 1}},
			Options:    options.Index().SetName("user_topic_compound_idx"),
		},
		// Drafts collection indexes
		{
			Collection: CollectionDrafts,
			Keys:       bson.D{{Key: "user_id", Value: 1}},
			Options:    options.Index().SetName("user_id_idx"),
		},
		{
			Collection: CollectionDrafts,
			Keys:       bson.D{{Key: "status", Value: 1}},
			Options:    options.Index().SetName("status_idx"),
		},
		// Topics collection indexes
		{
			Collection: CollectionTopics,
			Keys:       bson.D{{Key: "user_id", Value: 1}},
			Options:    options.Index().SetName("user_id_idx"),
		},
		// Prompts collection indexes
		{
			Collection: CollectionPrompts,
			Keys:       bson.D{{Key: "user_id", Value: 1}},
			Options:    options.Index().SetName("user_id_idx"),
		},
		{
			Collection: CollectionPrompts,
			Keys:       bson.D{{Key: "user_id", Value: 1}, {Key: "type", Value: 1}},
			Options:    options.Index().SetName("user_type_compound_idx"),
		},
		{
			Collection: CollectionPrompts,
			Keys:       bson.D{{Key: "user_id", Value: 1}, {Key: "style_name", Value: 1}},
			Options:    options.Index().SetName("user_style_compound_idx"),
		},
	}
}

// ValidationSchema represents MongoDB validation schema
type ValidationSchema struct {
	Collection string
	Validator  bson.M
}

// GetValidationSchemas returns all validation schemas for collections
func GetValidationSchemas() []ValidationSchema {
	return []ValidationSchema{
		{
			Collection: CollectionUsers,
			Validator: bson.M{
				"$jsonSchema": bson.M{
					"bsonType": "object",
					"required": []string{"email", "linkedin_id", "created_at"},
					"properties": bson.M{
						"email": bson.M{
							"bsonType":    "string",
							"description": "must be a string and is required",
						},
						"linkedin_id": bson.M{
							"bsonType":    "string",
							"description": "must be a string and is required",
						},
						"created_at": bson.M{
							"bsonType":    "date",
							"description": "must be a date and is required",
						},
						"updated_at": bson.M{
							"bsonType":    "date",
							"description": "must be a date if present",
						},
					},
				},
			},
		},
		{
			Collection: CollectionTopics,
			Validator: bson.M{
				"$jsonSchema": bson.M{
					"bsonType": "object",
					"required": []string{"user_id", "name", "created_at"},
					"properties": bson.M{
						"user_id": bson.M{
							"bsonType":    "objectId",
							"description": "must be an objectId and is required",
						},
						"name": bson.M{
							"bsonType":    "string",
							"description": "must be a string and is required",
						},
						"created_at": bson.M{
							"bsonType":    "date",
							"description": "must be a date and is required",
						},
					},
				},
			},
		},
		{
			Collection: CollectionIdeas,
			Validator: bson.M{
				"$jsonSchema": bson.M{
					"bsonType": "object",
					"required": []string{"user_id", "topic", "idea", "created_at"},
					"properties": bson.M{
						"user_id": bson.M{
							"bsonType":    "objectId",
							"description": "must be an objectId and is required",
						},
						"topic": bson.M{
							"bsonType":    "string",
							"description": "must be a string and is required",
						},
						"idea": bson.M{
							"bsonType":    "string",
							"description": "must be a string and is required",
						},
						"created_at": bson.M{
							"bsonType":    "date",
							"description": "must be a date and is required",
						},
					},
				},
			},
		},
		{
			Collection: CollectionDrafts,
			Validator: bson.M{
				"$jsonSchema": bson.M{
					"bsonType": "object",
					"required": []string{"user_id", "content", "status", "created_at"},
					"properties": bson.M{
						"user_id": bson.M{
							"bsonType":    "objectId",
							"description": "must be an objectId and is required",
						},
						"content": bson.M{
							"bsonType":    "string",
							"description": "must be a string and is required",
						},
						"status": bson.M{
							"bsonType":    "string",
							"description": "must be a string and is required",
						},
						"created_at": bson.M{
							"bsonType":    "date",
							"description": "must be a date and is required",
						},
					},
				},
			},
		},
	}
}

// CollectionManager handles collection initialization and management
type CollectionManager struct {
	client *Client
	logger *zap.Logger
}

// NewCollectionManager creates a new collection manager
func NewCollectionManager(client *Client, logger *zap.Logger) *CollectionManager {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &CollectionManager{
		client: client,
		logger: logger,
	}
}

// InitializeCollections creates indexes and applies validation schemas
func (cm *CollectionManager) InitializeCollections(ctx context.Context, createIndexes, applyValidation bool) error {
	db, err := cm.client.GetDefaultDatabase()
	if err != nil {
		return fmt.Errorf("failed to get database: %w", err)
	}

	// Create indexes if requested
	if createIndexes {
		if err := cm.createIndexes(ctx, db); err != nil {
			return fmt.Errorf("failed to create indexes: %w", err)
		}
	}

	// Apply validation schemas if requested
	if applyValidation {
		if err := cm.applyValidationSchemas(ctx, db); err != nil {
			return fmt.Errorf("failed to apply validation schemas: %w", err)
		}
	}

	return nil
}

// createIndexes creates all defined indexes
func (cm *CollectionManager) createIndexes(ctx context.Context, db *mongo.Database) error {
	cm.logger.Info("Creating database indexes")

	indexDefs := GetAllIndexDefinitions()
	for _, indexDef := range indexDefs {
		collection := db.Collection(indexDef.Collection)
		
		indexModel := mongo.IndexModel{
			Keys:    indexDef.Keys,
			Options: indexDef.Options,
		}

		indexName, err := collection.Indexes().CreateOne(ctx, indexModel)
		if err != nil {
			cm.logger.Error("Failed to create index",
				zap.String("collection", indexDef.Collection),
				zap.Error(err),
			)
			return fmt.Errorf("failed to create index on %s: %w", indexDef.Collection, err)
		}

		cm.logger.Info("Created index",
			zap.String("collection", indexDef.Collection),
			zap.String("index", indexName),
		)
	}

	cm.logger.Info("Successfully created all indexes")
	return nil
}

// applyValidationSchemas applies validation schemas to collections
func (cm *CollectionManager) applyValidationSchemas(ctx context.Context, db *mongo.Database) error {
	cm.logger.Info("Applying validation schemas")

	schemas := GetValidationSchemas()
	for _, schema := range schemas {
		// Create collection with validation if it doesn't exist
		cmd := bson.D{
			{Key: "collMod", Value: schema.Collection},
			{Key: "validator", Value: schema.Validator},
			{Key: "validationLevel", Value: "moderate"},
		}

		var result bson.M
		err := db.RunCommand(ctx, cmd).Decode(&result)
		if err != nil {
			// If collection doesn't exist, create it with validation
			if mongo.IsDuplicateKeyError(err) || err.Error() == "ns not found" {
				createCmd := bson.D{
					{Key: "create", Value: schema.Collection},
					{Key: "validator", Value: schema.Validator},
					{Key: "validationLevel", Value: "moderate"},
				}
				err = db.RunCommand(ctx, createCmd).Decode(&result)
			}
		}

		if err != nil {
			cm.logger.Error("Failed to apply validation schema",
				zap.String("collection", schema.Collection),
				zap.Error(err),
			)
			return fmt.Errorf("failed to apply validation on %s: %w", schema.Collection, err)
		}

		cm.logger.Info("Applied validation schema",
			zap.String("collection", schema.Collection),
		)
	}

	cm.logger.Info("Successfully applied all validation schemas")
	return nil
}

// GetUsersCollection returns the users collection
func (cm *CollectionManager) GetUsersCollection() (*mongo.Collection, error) {
	return cm.client.GetCollection(CollectionUsers)
}

// GetTopicsCollection returns the topics collection
func (cm *CollectionManager) GetTopicsCollection() (*mongo.Collection, error) {
	return cm.client.GetCollection(CollectionTopics)
}

// GetIdeasCollection returns the ideas collection
func (cm *CollectionManager) GetIdeasCollection() (*mongo.Collection, error) {
	return cm.client.GetCollection(CollectionIdeas)
}

// GetDraftsCollection returns the drafts collection
func (cm *CollectionManager) GetDraftsCollection() (*mongo.Collection, error) {
	return cm.client.GetCollection(CollectionDrafts)
}

// CountDocuments returns the count of documents in a collection with optional filter
func (cm *CollectionManager) CountDocuments(ctx context.Context, collectionName string, filter bson.M) (int64, error) {
	collection, err := cm.client.GetCollection(collectionName)
	if err != nil {
		return 0, err
	}

	count, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to count documents in %s: %w", collectionName, err)
	}

	return count, nil
}
