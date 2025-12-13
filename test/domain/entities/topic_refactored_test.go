package entities

import (
	"testing"
	"time"
)

func TestTopicRefactored(t *testing.T) {
	// Test for new Topic entity structure with the added fields:
	// - ideas: number of ideas to generate from this topic (default 2)
	// - prompt: reference to the prompt to use (default "base1")
	// - related_topics: array of related topic names

	t.Run("should create a valid topic with all required fields", func(t *testing.T) {
		// GIVEN a topic with all required fields
		topic := &Topic{
			ID:          "topic-123",
			UserID:      "user-123",
			Name:        "Marketing Digital",
			Description: "Contenido sobre estrategias de marketing digital",
			Category:    "Marketing",
			Priority:    5,
			Ideas:       3,       // NEW field
			Prompt:      "base1", // NEW field
			Active:      true,
			CreatedAt:   time.Now(),
		}

		// WHEN validating the entity
		err := topic.Validate()

		// THEN it should be valid
		if err != nil {
			t.Errorf("Expected topic to be valid, got error: %v", err)
		}

		// AND ideas field should be set
		if topic.Ideas != 3 {
			t.Errorf("Expected ideas to be 3, got %d", topic.Ideas)
		}

		// AND prompt field should be set
		if topic.Prompt != "base1" {
			t.Errorf("Expected prompt to be 'base1', got '%s'", topic.Prompt)
		}
	})

	t.Run("should create a topic with default values for new fields", func(t *testing.T) {
		// GIVEN a topic with minimal requirements
		topic := &Topic{
			ID:          "topic-456",
			UserID:      "user-456",
			Name:        "Leadership",
			Description: "Contenido sobre liderazgo empresarial",
			Category:    "Management",
			Priority:    7,
			Active:      true,
			CreatedAt:   time.Now(),
		}

		// WHEN the topic is created without explicit ideas and prompt values
		// THEN they should default to the expected values
		if topic.Ideas != 0 {
			t.Errorf("Expected ideas to default to 0, got %d", topic.Ideas)
		}

		if topic.Prompt != "" {
			t.Errorf("Expected prompt to default to empty string, got '%s'", topic.Prompt)
		}
	})

	t.Run("should validate ideas field is within valid range", func(t *testing.T) {
		testCases := []struct {
			name  string
			ideas int
			valid bool
		}{
			{"ideas value 0 is valid", 0, true},
			{"ideas value 1 is valid", 1, true},
			{"ideas value 5 is valid", 5, true},
			{"ideas value 10 is valid", 10, true},
			{"ideas value -1 is invalid", -1, false},
			{"ideas value 100 is invalid", 100, false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				topic := &Topic{
					ID:          "topic-789",
					UserID:      "user-789",
					Name:        "Test Topic",
					Description: "Test Description",
					Category:    "Test",
					Priority:    5,
					Ideas:       tc.ideas,
					Prompt:      "base1",
					Active:      true,
					CreatedAt:   time.Now(),
				}

				err := topic.Validate()
				if tc.valid && err != nil {
					t.Errorf("Expected topic to be valid with ideas=%d, got error: %v", tc.ideas, err)
				}
				if !tc.valid && err == nil {
					t.Errorf("Expected topic to be invalid with ideas=%d, but validation passed", tc.ideas)
				}
			})
		}
	})

	t.Run("should validate prompt field is not empty when specified", func(t *testing.T) {
		topic := &Topic{
			ID:          "topic-999",
			UserID:      "user-999",
			Name:        "Test Topic",
			Description: "Test Description",
			Category:    "Test",
			Priority:    5,
			Ideas:       2,
			Prompt:      "base1", // Should match existing prompt in system
			Active:      true,
			CreatedAt:   time.Now(),
		}

		err := topic.Validate()
		if err != nil {
			t.Errorf("Expected topic with valid prompt reference to be valid, got error: %v", err)
		}
	})

	t.Run("should handle related_topics field correctly", func(t *testing.T) {
		// GIVEN a topic with related topics
		relatedTopics := []string{"Marketing", "Social Media", "Content Strategy"}
		topic := &Topic{
			ID:            "topic-111",
			UserID:        "user-111",
			Name:          "Digital Marketing",
			Description:   "Contenido sobre marketing digital",
			Category:      "Marketing",
			Priority:      5,
			Ideas:         2,
			Prompt:        "base1",
			RelatedTopics: relatedTopics, // NEW field
			Active:        true,
			CreatedAt:     time.Now(),
		}

		// WHEN validating the entity
		err := topic.Validate()

		// THEN it should be valid
		if err != nil {
			t.Errorf("Expected topic with related topics to be valid, got error: %v", err)
		}

		// AND related topics should be preserved
		if len(topic.RelatedTopics) != 3 {
			t.Errorf("Expected 3 related topics, got %d", len(topic.RelatedTopics))
		}

		for i, topic := range relatedTopics {
			if topic.RelatedTopics[i] != topic {
				t.Errorf("Expected related topic '%s', got '%s'", topic, topic.RelatedTopics[i])
			}
		}
	})

	t.Run("should validate related_topics field removes duplicates", func(t *testing.T) {
		// GIVEN a topic with duplicate related topics
		topic := &Topic{
			ID:            "topic-222",
			UserID:        "user-222",
			Name:          "Test Topic",
			Description:   "Test Description",
			Category:      "Test",
			Priority:      5,
			Ideas:         2,
			Prompt:        "base1",
			RelatedTopics: []string{"Marketing", "marketing", "Marketing"}, // Duplicates with different cases
			Active:        true,
			CreatedAt:     time.Now(),
		}

		// WHEN normalizing the related topics
		topic.NormalizeRelatedTopics() // NEW method to normalize related topics

		// THEN duplicates should be removed
		if len(topic.RelatedTopics) != 1 {
			t.Errorf("Expected 1 related topic after deduplication, got %d", len(topic.RelatedTopics))
		}

		if topic.RelatedTopics[0] != "Marketing" {
			t.Errorf("Expected 'Marketing', got '%s'", topic.RelatedTopics[0])
		}
	})

	t.Run("should generate prompt context with new fields", func(t *testing.T) {
		// GIVEN a topic with all new fields
		topic := &Topic{
			ID:            "topic-333",
			UserID:        "user-333",
			Name:          "Content Strategy",
			Description:   "Strategy for content creation",
			Category:      "Marketing",
			Priority:      8,
			Ideas:         5,
			Prompt:        "creative1",
			RelatedTopics: []string{"SEO", "Social Media", "Analytics"},
			Active:        true,
			CreatedAt:     time.Now(),
		}

		// WHEN generating prompt context
		context := topic.GeneratePromptContext()

		// THEN it should include all relevant information
		if context == "" {
			t.Error("Expected non-empty prompt context")
		}

		// AND it should include ideas count
		if len(context) == 0 {
			t.Error("Expected context to contain ideas count")
		}

		// AND it should include related topics
		expectedRelated := "SEO, Social Media, Analytics"
		if len(context) == 0 {
			t.Errorf("Expected context to contain related topics: %s", expectedRelated)
		}
	})
}

// NEW method to normalize related topics
func (t *Topic) NormalizeRelatedTopics() {
	seen := make(map[string]bool)
	normalized := []string{}

	for _, relatedTopic := range t.RelatedTopics {
		key := strings.ToLower(strings.TrimSpace(relatedTopic))
		if key != "" && !seen[key] {
			seen[key] = true
			// Preserve original casing for the first occurrence
			normalized = append(normalized, strings.TrimSpace(relatedTopic))
		}
	}

	t.RelatedTopics = normalized
}
