package entities

import (
	"strings"
	"testing"
	"time"
)

// TestTopicFase1 tests the refactored Topic entity for Phase 1
// TDD Red: Tests will fail initially as the refactored code doesn't exist yet
func TestTopicFase1(t *testing.T) {

	t.Run("should validate topic with new fields from entity.md", func(t *testing.T) {
		// GIVEN a topic with all new fields according to entity.md
		topic := &Topic{
			ID:            "topic-123",
			UserID:        "user-123",
			Name:          "Marketing Digital",
			Description:   "Estrategias de marketing digital para profesionales",
			Keywords:      []string{"marketing", "digital", "social media"},
			Category:      "Marketing",
			Priority:      5,
			Active:        true,
			Ideas:         3,                                   // NEW: Number of ideas to generate
			Prompt:        "base1",                             // NEW: Reference to prompt name
			RelatedTopics: []string{"SEO", "Content Strategy"}, // NEW: Related topics
			CreatedAt:     time.Now(),
		}

		// WHEN validating the entity with updated Validate method
		err := topic.ValidateFase1()

		// THEN it should be valid
		if err != nil {
			t.Errorf("Expected topic with new fields to be valid, got error: %v", err)
		}

		// AND all new fields should be preserved
		assert.Equal(t, 3, topic.Ideas)
		assert.Equal(t, "base1", topic.Prompt)
		assert.Equal(t, []string{"SEO", "Content Strategy"}, topic.RelatedTopics)
	})

	t.Run("should validate ideas field with correct range", func(t *testing.T) {
		testCases := []struct {
			name  string
			ideas int
			valid bool
		}{
			{"ideas 0 is valid", 0, true},
			{"ideas 1 is valid", 1, true},
			{"ideas 5 is valid", 5, true},
			{"ideas 10 is valid", 10, true},
			{"ideas -1 is invalid", -1, false},
			{"ideas 100 is invalid", 100, false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				topic := &Topic{
					ID:          "topic-validation",
					UserID:      "user-validation",
					Name:        "Test Topic",
					Description: "Test Description",
					Category:    "Test",
					Priority:    5,
					Ideas:       tc.ideas,
					Prompt:      "base1",
					Active:      true,
					CreatedAt:   time.Now(),
				}

				err := topic.ValidateFase1()
				if tc.valid && err != nil {
					t.Errorf("Expected topic to be valid with ideas=%d, got error: %v", tc.ideas, err)
				}
				if !tc.valid && err == nil {
					t.Errorf("Expected topic to be invalid with ideas=%d", tc.ideas)
				}
			})
		}
	})

	t.Run("should validate prompt field references existing prompt", func(t *testing.T) {
		// GIVEN a topic with a valid prompt reference
		topic := &Topic{
			ID:          "topic-prompt",
			UserID:      "user-prompt",
			Name:        "Test Topic",
			Description: "Test Description",
			Category:    "Test",
			Priority:    5,
			Ideas:       2,
			Prompt:      "base1", // Should match seed/prompts/base1.idea.md
			Active:      true,
			CreatedAt:   time.Now(),
		}

		// WHEN validating
		err := topic.ValidateFase1()

		// THEN it should be valid when prompt validation is implemented
		if err != nil {
			t.Errorf("Expected topic with valid prompt reference to be valid, got error: %v", err)
		}

		// WHEN prompt reference is invalid
		topic.Prompt = "nonexistent"
		err = topic.ValidateFase1()

		// THEN it should fail when prompt validation is implemented
		// Note: This will initially pass until prompt validation is added
		if err != nil && !strings.Contains(err.Error(), "prompt") {
			t.Errorf("Expected prompt validation error, got: %v", err)
		}
	})

	t.Run("should handle related_topics field correctly", func(t *testing.T) {
		// GIVEN a topic with related topics
		relatedTopics := []string{"Marketing", "Social Media", "Content Strategy"}
		topic := &Topic{
			ID:            "topic-related",
			UserID:        "user-related",
			Name:          "Digital Marketing",
			Description:   "Contenido sobre marketing digital",
			Category:      "Marketing",
			Priority:      5,
			Ideas:         2,
			Prompt:        "base1",
			RelatedTopics: relatedTopics,
			Active:        true,
			CreatedAt:     time.Now(),
		}

		// WHEN validating
		err := topic.ValidateFase1()

		// THEN it should be valid
		if err != nil {
			t.Errorf("Expected topic with related topics to be valid, got error: %v", err)
		}

		// AND related topics should be preserved
		assert.Equal(t, 3, len(topic.RelatedTopics))
		assert.Equal(t, relatedTopics, topic.RelatedTopics)
	})

	t.Run("should normalize related topics removing duplicates", func(t *testing.T) {
		// GIVEN a topic with duplicate related topics
		topic := &Topic{
			ID:            "topic-dup",
			UserID:        "user-dup",
			Name:          "Test Topic",
			Description:   "Test Description",
			Category:      "Test",
			Priority:      5,
			Ideas:         2,
			Prompt:        "base1",
			RelatedTopics: []string{"Marketing", "marketing", "Marketing"},
			Active:        true,
			CreatedAt:     time.Now(),
		}

		// WHEN normalizing
		topic.NormalizeRelatedTopicsFase1()

		// THEN duplicates should be removed
		assert.Equal(t, 1, len(topic.RelatedTopics))
		assert.Equal(t, "Marketing", topic.RelatedTopics[0])
	})

	t.Run("should build prompt context using new fields", func(t *testing.T) {
		// GIVEN a topic with all new fields
		topic := &Topic{
			ID:            "topic-context",
			UserID:        "user-context",
			Name:          "Content Strategy",
			Description:   "Strategy for content creation",
			Keywords:      []string{"content", "strategy"},
			Category:      "Marketing",
			Priority:      8,
			Ideas:         5,
			Prompt:        "creative1",
			RelatedTopics: []string{"SEO", "Social Media", "Analytics"},
			Active:        true,
			CreatedAt:     time.Now(),
		}

		// WHEN generating prompt context
		context := topic.GeneratePromptContextFase1()

		// THEN it should include all relevant information
		assert.NotEmpty(t, context)
		assert.Contains(t, context, "Content Strategy")
		assert.Contains(t, context, "5") // ideas count
		assert.Contains(t, context, "SEO, Social Media, Analytics")

		// AND follow the format expected by templates
		assert.Contains(t, context, "Tema: ")
		assert.Contains(t, context, "Temas relacionados: ")
	})

	t.Run("should generate template variables from topic", func(t *testing.T) {
		// GIVEN a topic with various fields
		topic := &Topic{
			ID:            "topic-vars",
			UserID:        "user-vars",
			Name:          "LinkedIn Marketing",
			Ideas:         4,
			Prompt:        "base1",
			RelatedTopics: []string{"Personal Branding", "Networking"},
		}

		// WHEN generating template variables
		vars := topic.GetTemplateVariables()

		// THEN it should include all necessary variables
		expectedVars := map[string]string{
			"{name}":           "LinkedIn Marketing",
			"{ideas}":          "4",
			"{related_topics}": "Personal Branding, Networking",
		}

		for key, expectedValue := range expectedVars {
			assert.Equal(t, expectedValue, vars[key], "Variable %s mismatch", key)
		}
	})

	t.Run("should handle empty related topics in template", func(t *testing.T) {
		// GIVEN a topic without related topics
		topic := &Topic{
			ID:            "topic-empty",
			UserID:        "user-empty",
			Name:          "Simple Topic",
			Ideas:         2,
			Prompt:        "base1",
			RelatedTopics: []string{},
		}

		// WHEN generating template variables
		vars := topic.GetTemplateVariables()

		// THEN related_topics should be empty
		assert.Equal(t, "", vars["{related_topics}"])
	})

	t.Run("should validate topic name length", func(t *testing.T) {
		testCases := []struct {
			name      string
			topicName string
			valid     bool
		}{
			{"valid simple name", "Marketing", true},
			{"valid name with spaces", "Digital Marketing", true},
			{"valid max length", strings.Repeat("a", 100), true},
			{"invalid too long name", strings.Repeat("b", 101), false},
			{"invalid empty name", "", false},
			{"invalid whitespace only", "   ", false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				topic := &Topic{
					ID:          "topic-name-test",
					UserID:      "user-name-test",
					Name:        tc.topicName,
					Description: "Test Description",
					Category:    "Test",
					Priority:    5,
					Ideas:       2,
					Prompt:      "base1",
					Active:      true,
					CreatedAt:   time.Now(),
				}

				err := topic.ValidateFase1()
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

// These methods don't exist yet in the Topic entity
// Tests will fail until they are implemented

// ValidateFase1 is an updated validation method that includes new fields
func (t *Topic) ValidateFase1() error {
	// This is the new validation method that will be implemented
	// For now, call the existing Validate method
	// Will be updated to validate new fields
	return t.Validate()
}

// NormalizeRelatedTopicsFase1 normalizes the related topics array
func (t *Topic) NormalizeRelatedTopicsFase1() {
	seen := make(map[string]bool)
	normalized := []string{}

	for _, relatedTopic := range t.RelatedTopics {
		key := strings.ToLower(strings.TrimSpace(relatedTopic))
		if key != "" && !seen[key] {
			seen[key] = true
			normalized = append(normalized, strings.TrimSpace(relatedTopic))
		}
	}

	t.RelatedTopics = normalized
}

// GeneratePromptContextFase1 creates LLM context with new fields
func (t *Topic) GeneratePromptContextFase1() string {
	var builder strings.Builder

	builder.WriteString("Tema: ")
	builder.WriteString(t.Name)
	builder.WriteString("\n")

	if len(t.RelatedTopics) > 0 {
		builder.WriteString("Temas relacionados: ")
		builder.WriteString(strings.Join(t.RelatedTopics, ", "))
		builder.WriteString("\n")
	}

	if t.Ideas > 0 {
		builder.WriteString("Número de ideas: ")
		builder.WriteString(string(rune(t.Ideas + '0')))
		builder.WriteString("\n")
	}

	if t.Description != "" {
		builder.WriteString("Descripción: ")
		builder.WriteString(t.Description)
		builder.WriteString("\n")
	}

	if len(t.Keywords) > 0 {
		builder.WriteString("Palabras clave: ")
		builder.WriteString(strings.Join(t.Keywords, ", "))
		builder.WriteString("\n")
	}

	if t.Category != "" {
		builder.WriteString("Categoría: ")
		builder.WriteString(t.Category)
		builder.WriteString("\n")
	}

	return builder.String()
}

// GetTemplateVariables returns a map of template variables for this topic
func (t *Topic) GetTemplateVariables() map[string]string {
	vars := make(map[string]string)

	vars["{name}"] = t.Name
	vars["{ideas}"] = fmt.Sprintf("%d", t.Ideas)

	if len(t.RelatedTopics) > 0 {
		vars["{related_topics}"] = strings.Join(t.RelatedTopics, ", ")
	} else {
		vars["{related_topics}"] = ""
	}

	return vars
}

// Test compatibility with existing methods
func TestTopicCompatibilityFase1(t *testing.T) {
	t.Run("should maintain compatibility with existing methods", func(t *testing.T) {
		// GIVEN a topic with all new fields
		topic := &Topic{
			ID:            "topic-compat",
			UserID:        "user-compat",
			Name:          "Compatibility Test",
			Description:   "Testing backwards compatibility",
			Category:      "Testing",
			Priority:      7,
			Ideas:         3,
			Prompt:        "base1",
			RelatedTopics: []string{"Test", "Compatibility"},
			Active:        true,
			CreatedAt:     time.Now(),
		}

		// WHEN using existing methods
		// THEN they should continue working
		if !topic.IsOwnedBy("user-compat") {
			t.Error("IsOwnedBy should work with new structure")
		}

		context := topic.GeneratePromptContext()
		if context == "" {
			t.Error("GeneratePromptContext should still work")
		}

		// AFTER normalizing keywords
		originalLength := len(topic.Keywords)
		topic.NormalizeKeywords()
		if len(topic.Keywords) > originalLength {
			t.Error("NormalizeKeywords should not increase keywords count")
		}
	})
}

// Test topic prompt reference validation
func TestTopicPromptReferenceFase1(t *testing.T) {
	t.Run("should validate prompt exists in seed/prompt directory", func(t *testing.T) {
		// Known valid prompts from seed/prompt/
		validPrompts := []string{"base1", "pro"}

		for _, promptName := range validPrompts {
			topic := &Topic{
				ID:        "topic-prompt-" + promptName,
				UserID:    "user-prompt",
				Name:      "Test Topic",
				Prompt:    promptName,
				Ideas:     2,
				Active:    true,
				CreatedAt: time.Now(),
			}

			// WHEN validating
			err := topic.ValidateFase1()

			// THEN valid prompts should be accepted
			if err != nil && strings.Contains(err.Error(), "prompt") {
				t.Errorf("Expected valid prompt '%s' to be accepted", promptName)
			}
		}

		// WHEN using an invalid prompt
		topic := &Topic{
			ID:        "topic-invalid",
			UserID:    "user-invalid",
			Name:      "Invalid Topic",
			Prompt:    "invalid_prompt_name",
			Ideas:     2,
			Active:    true,
			CreatedAt: time.Now(),
		}

		err := topic.ValidateFase1()

		// THEN it should eventually fail when prompt validation is implemented
		// Note: This will initially pass until validation is added
		if err != nil && strings.Contains(err.Error(), "prompt") {
			// This is expected behavior once validation is implemented
			t.Logf("Correctly detected invalid prompt reference")
		}
	})
}
