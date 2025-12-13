package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/linkgen-ai/backend/src/domain/entities"
)

// MigrationReport represents the results of a migration operation
type MigrationReport struct {
	MigratedCount  int       `json:"migrated_count"`
	FailedCount    int       `json:"failed_count"`
	MigrationDate  time.Time `json:"migration_date"`
	Errors         []string  `json:"errors,omitempty"`
}

// RollbackReport represents the results of a rollback operation
type RollbackReport struct {
	RolledBackCount int       `json:"rolled_back_count"`
	FailedCount     int       `json:"failed_count"`
	RollbackDate    time.Time `json:"rollback_date"`
	Errors          []string  `json:"errors,omitempty"`
}

// ValidationReport represents migration validation results
type ValidationReport struct {
	ItemsNeedingMigration  int       `json:"items_needing_migration"`
	ItemsAlreadyMigrated   int       `json:"items_already_migrated"`
	ConflictCount          int       `json:"conflict_count"`
	ValidationDate         time.Time `json:"validation_date"`
	Warnings               []string  `json:"warnings,omitempty"`
}

// DataMigrator handles data migration between schemas
type DataMigrator struct {
	db *mongo.Database
}

// NewDataMigrator creates a new data migrator
func NewDataMigrator(db *mongo.Database) *DataMigrator {
	return &DataMigrator{db: db}
}

// MigrateTopics migrates topics from old schema to new schema
func (m *DataMigrator) MigrateTopics(ctx context.Context) (*MigrationReport, error) {
	log.Println("Starting topic migration from old to new schema")

	report := &MigrationReport{
		MigrationDate: time.Now(),
		Errors:        []string{},
	}

	topicCollection := m.db.Collection("topics")

	// Find all topics that need migration (old schema)
	cursor, err := topicCollection.Find(ctx, bson.M{
		"prompt_name": bson.M{"$exists": false},
	})
	if err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("Error finding topics to migrate: %v", err))
		return report, err
	}
	defer cursor.Close(ctx)

	var topicsToMigrate []bson.M
	if err = cursor.All(ctx, &topicsToMigrate); err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("Error decoding topics: %v", err))
		return report, err
	}

	log.Printf("Found %d topics to migrate", len(topicsToMigrate))

	for _, topic := range topicsToMigrate {
		topicID := topic["_id"].(primitive.ObjectID).Hex()
		
		// Build update with new fields
		update := bson.M{
			"$set": bson.M{
				"prompt_name":  "base1",             // Default prompt
				"ideas_count":  getOldIdeasCount(topic),
				"updated_at":   time.Now(),
			},
		}

		// Apply migration
		_, err := topicCollection.UpdateOne(
			ctx,
			bson.M{"_id": topic["_id"]},
			update,
		)

		if err != nil {
			report.FailedCount++
			report.Errors = append(report.Errors, fmt.Sprintf("Failed to migrate topic %s: %v", topicID, err))
			log.Printf("Failed to migrate topic %s: %v", topicID, err)
		} else {
			report.MigratedCount++
			log.Printf("Successfully migrated topic %s", topicID)
		}
	}

	log.Printf("Topic migration completed. Migrated: %d, Failed: %d", report.MigratedCount, report.FailedCount)
	return report, nil
}

// MigrateIdeas migrates ideas to include topic_name field
func (m *DataMigrator) MigrateIdeas(ctx context.Context) (*MigrationReport, error) {
	log.Println("Starting idea migration to populate topic_name")

	report := &MigrationReport{
		MigrationDate: time.Now(),
		Errors:        []string{},
	}

	ideaCollection := m.db.Collection("ideas")
	topicCollection := m.db.Collection("topics")

	// Find all ideas without topic_name
	cursor, err := ideaCollection.Find(ctx, bson.M{
		"topic_name": bson.M{"$exists": false},
	})
	if err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("Error finding ideas to migrate: %v", err))
		return report, err
	}
	defer cursor.Close(ctx)

	var ideasToMigrate []bson.M
	if err = cursor.All(ctx, &ideasToMigrate); err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("Error decoding ideas: %v", err))
		return report, err
	}

	log.Printf("Found %d ideas to migrate", len(ideasToMigrate))

	for _, idea := range ideasToMigrate {
		ideaID := idea["_id"].(primitive.ObjectID).Hex()
		
		// Get the related topic to fetch its name
		topicID := idea["topic_id"].(primitive.ObjectID)
		var topic bson.M
		err := topicCollection.FindOne(ctx, bson.M{"_id": topicID}).Decode(&topic)
		if err != nil {
			report.FailedCount++
			report.Errors = append(report.Errors, fmt.Sprintf("Failed to find topic for idea %s: %v", ideaID, err))
			log.Printf("Failed to find topic for idea %s: %v", ideaID, err)
			continue
		}

		// Get topic name
		topicName, ok := topic["name"].(string)
		if !ok {
			report.FailedCount++
			report.Errors = append(report.Errors, fmt.Sprintf("Topic for idea %s has no name field", ideaID))
			continue
		}

		// Update idea with topic name
		_, err = ideaCollection.UpdateOne(
			ctx,
			bson.M{"_id": idea["_id"]},
			bson.M{
				"$set": bson.M{
					"topic_name": topicName,
					"updated_at": time.Now(),
				},
			},
		)

		if err != nil {
			report.FailedCount++
			report.Errors = append(report.Errors, fmt.Sprintf("Failed to migrate idea %s: %v", ideaID, err))
			log.Printf("Failed to migrate idea %s: %v", ideaID, err)
		} else {
			report.MigratedCount++
			log.Printf("Successfully migrated idea %s", ideaID)
		}
	}

	log.Printf("Idea migration completed. Migrated: %d, Failed: %d", report.MigratedCount, report.FailedCount)
	return report, nil
}

// MigratePrompts migrates prompts from style_name to name field
func (m *DataMigrator) MigratePrompts(ctx context.Context) (*MigrationReport, error) {
	log.Println("Starting prompt migration from style_name to name")

	report := &MigrationReport{
		MigrationDate: time.Now(),
		Errors:        []string{},
	}

	promptCollection := m.db.Collection("prompts")

	// Find all prompts with style_name but without name
	cursor, err := promptCollection.Find(ctx, bson.M{
		"style_name": bson.M{"$exists": true},
		"name":       bson.M{"$exists": false},
	})
	if err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("Error finding prompts to migrate: %v", err))
		return report, err
	}
	defer cursor.Close(ctx)

	var promptsToMigrate []bson.M
	if err = cursor.All(ctx, &promptsToMigrate); err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("Error decoding prompts: %v", err))
		return report, err
	}

	log.Printf("Found %d prompts to migrate", len(promptsToMigrate))

	for _, prompt := range promptsToMigrate {
		promptID := prompt["_id"].(primitive.ObjectID).Hex()
		styleName := prompt["style_name"].(string)

		// Update prompt with name field
		_, err := promptCollection.UpdateOne(
			ctx,
			bson.M{"_id": prompt["_id"]},
			bson.M{
				"$set": bson.M{
					"name":       styleName,
					"updated_at": time.Now(),
				},
				"$unset": bson.M{
					"style_name": "",
				},
			},
		)

		if err != nil {
			report.FailedCount++
			report.Errors = append(report.Errors, fmt.Sprintf("Failed to migrate prompt %s: %v", promptID, err))
			log.Printf("Failed to migrate prompt %s: %v", promptID, err)
		} else {
			report.MigratedCount++
			log.Printf("Successfully migrated prompt %s (style: %s)", promptID, styleName)
		}
	}

	log.Printf("Prompt migration completed. Migrated: %d, Failed: %d", report.MigratedCount, report.FailedCount)
	return report, nil
}

// RollbackTopics rolls back topic migration
func (m *DataMigrator) RollbackTopics(ctx context.Context, userID string) (*RollbackReport, error) {
	log.Printf("Starting topic rollback for user %s", userID)

	report := &RollbackReport{
		RollbackDate: time.Now(),
		Errors:       []string{},
	}

	topicCollection := m.db.Collection("topics")

	// Find all topics for the user that have been migrated
	filter := bson.M{
		"user_id":     userID,
		"prompt_name": bson.M{"$exists": true},
	}
	
	cursor, err := topicCollection.Find(ctx, filter)
	if err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("Error finding topics to rollback: %v", err))
		return report, err
	}
	defer cursor.Close(ctx)

	var topicsToRollback []bson.M
	if err = cursor.All(ctx, &topicsToRollback); err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("Error decoding topics: %v", err))
		return report, err
	}

	log.Printf("Found %d topics to rollback", len(topicsToRollback))

	for _, topic := range topicsToRollback {
		topicID := topic["_id"].(primitive.ObjectID).Hex()
		
		// Build update to restore old schema
		update := bson.M{
			"$set": bson.M{
				"ideas":     getIdeasCount(topic), // Convert ideas_count back to ideas
				"updated_at": time.Now(),
			},
			"$unset": bson.M{
				"prompt_name":  "",
				"ideas_count":  "",
			},
		}

		// Apply rollback
		_, err := topicCollection.UpdateOne(
			ctx,
			bson.M{"_id": topic["_id"]},
			update,
		)

		if err != nil {
			report.FailedCount++
			report.Errors = append(report.Errors, fmt.Sprintf("Failed to rollback topic %s: %v", topicID, err))
			log.Printf("Failed to rollback topic %s: %v", topicID, err)
		} else {
			report.RolledBackCount++
			log.Printf("Successfully rolled back topic %s", topicID)
		}
	}

	log.Printf("Topic rollback completed. Rolled back: %d, Failed: %d", report.RolledBackCount, report.FailedCount)
	return report, nil
}

// ValidateMigration validates migration before execution
func (m *DataMigrator) ValidateMigration(ctx context.Context, userID string) (*ValidationReport, error) {
	log.Printf("Starting migration validation for user %s", userID)

	report := &ValidationReport{
		ValidationDate: time.Now(),
		Warnings:       []string{},
	}

	topicCollection := m.db.Collection("topics")

	// Check for topics that need migration (old schema)
	oldTopicsCount, err := topicCollection.CountDocuments(ctx, bson.M{
		"user_id":     userID,
		"prompt_name": bson.M{"$exists": false},
	})
	if err != nil {
		return nil, fmt.Errorf("Error counting old topics: %v", err)
	}

	// Check for topics already migrated (new schema)
	newTopicsCount, err := topicCollection.CountDocuments(ctx, bson.M{
		"user_id":     userID,
		"prompt_name": bson.M{"$exists": true},
	})
	if err != nil {
		return nil, fmt.Errorf("Error counting new topics: %v", err)
	}

	report.ItemsNeedingMigration = int(oldTopicsCount)
	report.ItemsAlreadyMigrated = int(newTopicsCount)

	// Check for potential conflicts
	cursor, err := topicCollection.Find(ctx, bson.M{
		"user_id":       userID,
		"prompt_name":   bson.M{"$exists": true},
		"ideas_count":   bson.M{"$exists": false},
	})
	if err != nil {
		return nil, fmt.Errorf("Error checking conflicts: %v", err)
	}
	defer cursor.Close(ctx)

	var conflicts []bson.M
	if err = cursor.All(ctx, &conflicts); err != nil {
		return nil, fmt.Errorf("Error decoding conflicts: %v", err)
	}
	report.ConflictCount = len(conflicts)

	// Add warnings if needed
	if report.ConflictCount > 0 {
		report.Warnings = append(report.Warnings, fmt.Sprintf("Found %d topics with partial migration", report.ConflictCount))
	}

	log.Printf("Migration validation completed. Need migration: %d, Already migrated: %d, Conflicts: %d", 
		report.ItemsNeedingMigration, report.ItemsAlreadyMigrated, report.ConflictCount)

	return report, nil
}

// Helper functions
func getOldIdeasCount(topic bson.M) int {
	if ideas, ok := topic["ideas"]; ok {
		switch v := ideas.(type) {
		case int32:
			return int(v)
		case int64:
			return int(v)
		case int:
			return v
		}
	}
	return 3 // Default value
}

func getIdeasCount(topic bson.M) int {
	if ideasCount, ok := topic["ideas_count"]; ok {
		switch v := ideasCount.(type) {
		case int32:
			return int(v)
		case int64:
			return int(v)
		case int:
			return v
		}
	}
	return 3 // Default value
}
