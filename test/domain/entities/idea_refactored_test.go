package entities

import (
	"testing"
	"time"
)

func TestIdeaRefactored(t *testing.T) {
	// Test for the new Idea entity structure according to entity.md:
	// - content: Text of the idea (10-200 characters, reduced from 5000)
	// - quality_score: Optional score (0.0-1.0, default 0.0)
	// - used: Boolean indicating if used for drafts (default false)
	// - expires_at: Expiration date (30 days default)
	// - user_id: ID of the owner user
	// - topic_id: ID of the related topic
	// - topic_name: (NEW) unique name of the related topic

	t.Run("should create a valid idea with all required fields including topic_name", func(t *testing.T) {
		// GIVEN an idea with all required fields
		idea := &Idea{
			ID:          "idea-123",
			UserID:      "user-123",
			TopicID:     "topic-123",
			TopicName:   "Marketing Digital", // NEW field
			Content:     "Crea contenido sobre estrategias SEO para LinkedIn",
			QualityScore: &[]float64{0.8}[0],
			Used:        false,
			CreatedAt:   time.Now(),
		}

		// Set expiration
		idea.CalculateExpiration(30)

		// WHEN validating the entity
		err := idea.ValidateRefactored()

		// THEN it should be valid
		if err != nil {
			t.Errorf("Expected idea to be valid, got error: %v", err)
		}

		// AND topic_name should be set
		if idea.TopicName != "Marketing Digital" {
			t.Errorf("Expected topic_name to be 'Marketing Digital', got '%s'", idea.TopicName)
		}

		// AND content length should be within new limits
		if len(idea.Content) > MaxIdeaContentLengthRefactored {
			t.Errorf("Expected content length to be at most %d, got %d", MaxIdeaContentLengthRefactored, len(idea.Content))
		}
	})

	t.Run("should validate topic_name field is required", func(t *testing.T) {
		// GIVEN an idea without topic name
		idea := &Idea{
			ID:        "idea-456",
			UserID:    "user-456",
			TopicID:   "topic-456",
			TopicName: "", // Missing topic name
			Content:   "Valid content for idea",
			CreatedAt: time.Now(),
		}

		// WHEN validating the entity
		err := idea.ValidateRefactored()

		// THEN it should be invalid
		if err == nil {
			t.Error("Expected idea to be invalid without topic name")
		}

		if !contains(err.Error(), "topic name cannot be empty") {
			t.Errorf("Expected error message to mention topic name, got: %v", err)
		}
	})

	t.Run("should validate content length with new limits (10-200 characters)", func(t *testing.T) {
		// GIVEN ideas with different content lengths
		testCases := []struct {
			name    string
			content string
			valid   bool
		}{
			{
				"valid content exactly 10 characters",
				"1234567890",
				true,
			},
			{
				"valid content exactly 200 characters",
				strings.Repeat("a", 200),
				true,
			},
			{
				"invalid content 9 characters (too short)",
				"123456789",
				false,
			},
			{
				"invalid content 201 characters (too long)",
				strings.Repeat("b", 201),
				false,
			},
			{
				"invalid empty content",
				"",
				false,
			},
			{
				"invalid whitespace-only content",
				"   ",
				false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				idea := &Idea{
					ID:        "idea-length-test",
					UserID:    "user-length-test",
					TopicID:   "topic-length-test",
					TopicName: "Test Topic",
					Content:   tc.content,
					CreatedAt: time.Now(),
				}

				err := idea.ValidateRefactored()
				if tc.valid && err != nil {
					t.Errorf("Expected idea to be valid with content length %d, got error: %v", len(tc.content), err)
				}
				if !tc.valid && err == nil {
					t.Errorf("Expected idea to be invalid with content length %d", len(tc.content))
				}
			})
		}
	})

	t.Run("should default quality_score to 0.0 when not specified", func(t *testing.T) {
		// GIVEN an idea without quality score
		idea := &Idea{
			ID:        "idea-no-score",
			UserID:    "user-no-score",
			TopicID:   "topic-no-score",
			TopicName: "Test Topic",
			Content:   "Valid testing idea content",
			CreatedAt: time.Now(),
		}

		// WHEN validating and checking defaults
		err := idea.ValidateRefactored()

		// THEN it should be valid
		if err != nil {
			t.Errorf("Expected idea to be valid without quality score, got error: %v", err)
		}

		// AND quality score should default to 0.0
		if idea.QualityScore != nil && *idea.QualityScore != 0.0 {
			t.Errorf("Expected quality score to default to 0.0, got %f", *idea.QualityScore)
		}
	})

	t.Run("should default used to false when not specified", func(t *testing.T) {
		// GIVEN an idea without used flag
		idea := &Idea{
			ID:        "idea-no-used",
			UserID:    "user-no-used",
			TopicID:   "topic-no-used",
			TopicName: "Test Topic",
			Content:   "Valid testing idea content",
			CreatedAt: time.Now(),
		}

		// WHEN validateRefactored and checking defaults
		err := idea.ValidateRefactored()

		// THEN it should be valid
		if err != nil {
			t.Errorf("Expected idea to be valid without used flag, got error: %v", err)
		}

		// AND used should default to false
		if idea.Used != false {
			t.Errorf("Expected used to default to false, got %v", idea.Used)
		}
	})

	t.Run("should calculate expiration based on created date", func(t *testing.T) {
		// GIVEN an idea created at a specific time
		baseTime := time.Date(2023, 12, 1, 10, 0, 0, 0, time.UTC)
		idea := &Idea{
			ID:        "idea-expiration",
			UserID:    "user-expiration",
			TopicID:   "topic-expiration",
			TopicName: "Test Topic",
			Content:   "Valid testing idea content",
			CreatedAt: baseTime,
		}

		// WHEN calculating expiration
		idea.CalculateExpiration(30)

		// THEN expiration should be set correctly
		if idea.ExpiresAt == nil {
			t.Error("Expected expiration date to be set")
		}

		expectedExpiration := baseTime.AddDate(0, 0, 30)
		if !idea.ExpiresAt.Equal(expectedExpiration) {
			t.Errorf("Expected expiration to be %v, got %v", expectedExpiration, *idea.ExpiresAt)
		}
	})

	t.Run("should check if idea belongs to topic by name", func(t *testing.T) {
		// GIVEN an idea with a specific topic name
		idea := &Idea{
			ID:        "idea-topic-check",
			UserID:    "user-topic-check",
			TopicID:   "topic-123",
			TopicName: "Marketing Digital",
			Content:   "Valid testing idea content",
			CreatedAt: time.Now(),
		}

		// WHEN checking topic membership
		// THEN it should correctly identify topic relationships
		if !idea.BelongsToTopicByName("Marketing Digital") {
			t.Error("Expected idea to belong to 'Marketing Digital' topic")
		}

		if idea.BelongsToTopicByName("Other Topic") {
			t.Error("Expected idea to NOT belong to 'Other Topic' topic")
		}
	})

	t.Run("should maintain compatibility with existing methods", func(t *testing.T) {
		// GIVEN an idea with all fields set
		idea := &Idea{
			ID:        "idea-compat",
			UserID:    "user-compat",
			TopicID:   "topic-compat",
			TopicName: "Compatible Topic",
			Content:   "Valid content for compatibility test",
			CreatedAt: time.Now(),
		}
		idea.CalculateExpiration(30)

		// WHEN using existing methods
		// THEN they should work with the new structure
		if !idea.BelongsToUser("user-compat") {
			t.Error("Expected BelongsToUser to work with new structure")
		}

		if idea.Used {
			// Mark as used to test the method
			err := idea.MarkAsUsed()
			if err != nil {
				t.Errorf("Expected MarkAsUsed to work with new structure, got error: %v", err)
			}

			if !idea.Used {
				t.Error("Expected idea to be marked as used")
			}
		}

		if idea.IsExpired() {
			t.Error("Expected idea not to be expired")
		}

		if !idea.CanBeUsed() {
			t.Error("Expected idea to be usable when not used and not expired")
		}
	})

	t.Run("should handle topic_name field in all operations", func(t *testing.T) {
		// GIVEN an idea with topic name
		idea := &Idea{
			ID:        "idea-topic-ops",
			UserID:    "user-topic-ops",
			TopicID:   "topic-123",
			TopicName: "Leadership Strategy",
			Content:   "Create content about leadership development",
			CreatedAt: time.Now(),
		}

		// WHEN performing various operations
		// THEN topic name should be preserved
		err := idea.ValidateRefactored()
		if err != nil {
			t.Errorf("Expected validation to pass with topic name, got error: %v", err)
		}

		// Check topic name is preserved after marking as used
		originalTopicName := idea.TopicName
		err = idea.MarkAsUsed()
		if err != nil {
			t.Errorf("Expected MarkAsUsed to work, got error: %v", err)
		}

		if idea.TopicName != originalTopicName {
			t.Errorf("Expected topic name to be preserved after MarkAsUsed, got '%s'", idea.TopicName)
		}

		// Check topic name is preserved after calculating expiration
		idea.CalculateExpiration(60)
		if idea.TopicName != originalTopicName {
			t.Errorf("Expected topic name to be preserved after CalculateExpiration, got '%s'", idea.TopicName)
		}
	})

	t.Run("should validate topic_name format and length", func(t *testing.T) {
		// GIVEN ideas with different topic name formats
		testCases := []struct {
			name      string
			topicName string
			valid     bool
		}{
			{"valid simple name", "Marketing", true},
			{"valid name with spaces", "Digital Marketing", true},
			{"valid name with special chars", "SEO & Analytics", true},
			{"valid name with numbers", "2024 Trends", true},
			{"invalid empty name", "", false},
			{"invalid whitespace only", "   ", false},
			{"valid long name", strings.Repeat("a", 50), true},
			{"invalid too long name", strings.Repeat("b", 101), false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				idea := &Idea{
					ID:        "idea-topic-format",
					UserID:    "user-topic-format",
					TopicID:   "topic-format",
					TopicName: tc.topicName,
					Content:   "Valid testing idea content",
					CreatedAt: time.Now(),
				}

				err := idea.ValidateRefactored()
				if tc.valid && err != nil {
					t.Errorf("Expected topic name '%s' to be valid, got error: %v", tc.topicName, err)
				}
				if !tc.valid && err == nil {
					t.Errorf("Expected topic name '%s' to be invalid", tc.topicName)
				}
			})
		}
	})
}

// Constants for the refactored Idea entity
const (
	MaxIdeaContentLengthRefactored = 200  // Reduced from 5000
	MaxTopicNameLength             = 100  // Maximum topic name length
)

// NEW validation method for the refactored Idea entity
func (i *Idea) ValidateRefactored() error {
	// Validate ID
	if i.ID == "" {
		return fmt.Errorf("idea ID cannot be empty")
	}

	// Validate user ID
	if i.UserID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	// Validate topic ID
	if i.TopicID == "" {
		return fmt.Errorf("topic ID cannot be empty")
	}

	// Validate topic name (NEW field)
	if i.TopicName == "" {
		return fmt.Errorf("topic name cannot be empty")
	}

	trimmedTopicName := strings.TrimSpace(i.TopicName)
	if trimmedTopicName == "" {
		return fmt.Errorf("topic name cannot be only whitespace")
	}

	if len(trimmedTopicName) > MaxTopicNameLength {
		return fmt.Errorf("topic name too long (maximum %d characters)", MaxTopicNameLength)
	}

	// Validate content with new length limits
	if i.Content == "" {
		return fmt.Errorf("idea content cannot be empty")
	}

	trimmedContent := strings.TrimSpace(i.Content)
	if trimmedContent == "" {
		return fmt.Errorf("idea content cannot be only whitespace")
	}

	if len(trimmedContent) < MinIdeaContentLength {
		return fmt.Errorf("idea content too short (minimum %d characters)", MinIdeaContentLength)
	}

	if len(i.Content) > MaxIdeaContentLengthRefactored {
		return fmt.Errorf("idea content too long (maximum %d characters)", MaxIdeaContentLengthRefactored)
	}

	// Validate quality score
	if err := i.validateQualityScore(); err != nil {
		return err
	}

	// Validate timestamps
	if i.CreatedAt.IsZero() {
		return fmt.Errorf("created timestamp cannot be zero")
	}

	if i.CreatedAt.After(time.Now()) {
		return fmt.Errorf("created timestamp cannot be in the future")
	}

	return nil
}

// NEW method to check if idea belongs to a topic by name
func (i *Idea) BelongsToTopicByName(topicName string) bool {
	return i.TopicName != "" && i.TopicName == topicName
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
