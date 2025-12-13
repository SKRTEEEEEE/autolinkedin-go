package entities

import (
	"testing"
	"time"
)

func TestPromptRefactored(t *testing.T) {
	// Test for the new Prompt entity structure according to entity.md:
	// - name: [unique] identifier (currently using StyleName)
	// - type: for what the prompt is used (ideas | draft)
	// - prompt_template: plain text template with variable placeholders
	// - active: boolean indicating if prompt is active
	// - user_id: ID of the user using the prompt

	t.Run("should create a valid prompt with new structure", func(t *testing.T) {
		// GIVEN a prompt with the new structure requirements
		prompt := &Prompt{
			ID:             "prompt-123",
			UserID:         "user-123",
			Name:           "base1", // NEW field (replacing StyleName as identifier)
			Type:           PromptTypeIdeas,
			PromptTemplate: "Genera {ideas} ideas sobre el tema: {name}",
			Active:         true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		// WHEN validating the entity
		err := prompt.ValidateRefactored()

		// THEN it should be valid
		if err != nil {
			t.Errorf("Expected prompt to be valid, got error: %v", err)
		}

		// AND name field should be used as identifier
		if prompt.Name != "base1" {
			t.Errorf("Expected name to be 'base1', got '%s'", prompt.Name)
		}

		// AND prompt template should support variables
		if prompt.PromptTemplate == "" {
			t.Error("Expected prompt template to be set")
		}
	})

	t.Run("should validate name field is required and unique", func(t *testing.T) {
		// GIVEN prompts with invalid name values
		testCases := []struct {
			name  string
			valid bool
		}{
			{"", false},
			{" ", false},
			{"base1", true},
			{"professional", true},
			{"creative_ideas", true},
			{"prompt-with-dashes", true},
			{"a", true}, // Short names should be valid
		}

		for _, tc := range testCases {
			t.Run("name '"+tc.name+"'", func(t *testing.T) {
				prompt := &Prompt{
					ID:             "prompt-456",
					UserID:         "user-456",
					Name:           tc.name,
					Type:           PromptTypeIdeas,
					PromptTemplate: "Test template for {name}",
					Active:         true,
					CreatedAt:      time.Now(),
					UpdatedAt:      time.Now(),
				}

				err := prompt.ValidateRefactored()
				if tc.valid && err != nil {
					t.Errorf("Expected prompt to be valid with name='%s', got error: %v", tc.name, err)
				}
				if !tc.valid && err == nil {
					t.Errorf("Expected prompt to be invalid with name='%s', but validation passed", tc.name)
				}
			})
		}
	})

	t.Run("should validate type field is required (ideas or draft)", func(t *testing.T) {
		// GIVEN a prompt with invalid type
		prompt := &Prompt{
			ID:             "prompt-789",
			UserID:         "user-789",
			Name:           "base1",
			Type:           PromptType("invalid"), // Invalid type
			PromptTemplate: "Test template",
			Active:         true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		// WHEN validating the entity
		err := prompt.ValidateRefactored()

		// THEN it should be invalid
		if err == nil {
			t.Error("Expected prompt to be invalid with invalid type")
		}
	})

	t.Run("should validate prompt template contains proper placeholders", func(t *testing.T) {
		// GIVEN prompts with different template formats
		testCases := []struct {
			name          string
			template      string
			valid         bool
			expectedError string
		}{
			{
				"valid template with single placeholder",
				"Genera ideas sobre: {name}",
				true,
				"",
			},
			{
				"valid template with multiple placeholders",
				"Genera {ideas} ideas sobre {name} con contexto {related_topics}",
				true,
				"",
			},
			{
				"valid template with no placeholders",
				"Genera ideas genéricas sin parámetros",
				true,
				"",
			},
			{
				"invalid empty template",
				"",
				false,
				"prompt template cannot be empty",
			},
			{
				"invalid template too short",
				"Hi",
				false,
				"prompt template too short",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				prompt := &Prompt{
					ID:             "prompt-999",
					UserID:         "user-999",
					Name:           "base1",
					Type:           PromptTypeIdeas,
					PromptTemplate: tc.template,
					Active:         true,
					CreatedAt:      time.Now(),
					UpdatedAt:      time.Now(),
				}

				err := prompt.ValidateRefactored()
				if tc.valid && err != nil {
					t.Errorf("Expected template to be valid: '%s', got error: %v", tc.template, err)
				}
				if !tc.valid && err == nil {
					t.Errorf("Expected template to be invalid: '%s'", tc.template)
				}
				if !tc.valid && tc.expectedError != "" && err != nil {
					if !contains(err.Error(), tc.expectedError) {
						t.Errorf("Expected error to contain '%s', got '%s'", tc.expectedError, err.Error())
					}
				}
			})
		}
	})

	t.Run("should validate active field defaults to true", func(t *testing.T) {
		// GIVEN a prompt without explicit active value
		prompt := &Prompt{
			ID:             "prompt-000",
			UserID:         "user-000",
			Name:           "base1",
			Type:           PromptTypeIdeas,
			PromptTemplate: "Test template",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		// WHEN checking default values
		// THEN active should default to true
		if !prompt.Active {
			t.Error("Expected active to default to true")
		}
	})

	t.Run("should support both idea and draft type prompts", func(t *testing.T) {
		// GIVEN an ideas prompt
		ideasPrompt := &Prompt{
			ID:             "prompt-ideas",
			UserID:         "user-ideas",
			Name:           "base1",
			Type:           PromptTypeIdeas,
			PromptTemplate: "Genera {ideas} ideas sobre {name}",
			Active:         true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		// AND a draft prompt
		draftPrompt := &Prompt{
			ID:             "prompt-draft",
			UserID:         "user-draft",
			Name:           "professional",
			Type:           PromptTypeDrafts,
			PromptTemplate: "Crea un post profesional sobre: {content}",
			Active:         true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		// WHEN validating both prompts
		ideasErr := ideasPrompt.ValidateRefactored()
		draftErr := draftPrompt.ValidateRefactored()

		// THEN both should be valid
		if ideasErr != nil {
			t.Errorf("Expected ideas prompt to be valid, got error: %v", ideasErr)
		}

		if draftErr != nil {
			t.Errorf("Expected draft prompt to be valid, got error: %v", draftErr)
		}

		// AND type methods should work correctly
		if !ideasPrompt.IsIdeasPrompt() {
			t.Error("Expected IsIdeasPrompt() to return true for ideas prompt")
		}

		if ideasPrompt.IsDraftsPrompt() {
			t.Error("Expected IsDraftsPrompt() to return false for ideas prompt")
		}

		if !draftPrompt.IsDraftsPrompt() {
			t.Error("Expected IsDraftsPrompt() to return true for draft prompt")
		}

		if draftPrompt.IsIdeasPrompt() {
			t.Error("Expected IsIdeasPrompt() to return false for draft prompt")
		}
	})

	t.Run("should validate user_id field is required", func(t *testing.T) {
		// GIVEN a prompt without user ID
		prompt := &Prompt{
			ID:             "prompt-no-user",
			UserID:         "", // Missing user ID
			Name:           "base1",
			Type:           PromptTypeIdeas,
			PromptTemplate: "Test template",
			Active:         true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		// WHEN validating the entity
		err := prompt.ValidateRefactored()

		// THEN it should be invalid
		if err == nil {
			t.Error("Expected prompt to be invalid without user ID")
		}
	})

	t.Run("should process template variables correctly", func(t *testing.T) {
		// GIVEN a prompt with template variables
		prompt := &Prompt{
			ID:             "prompt-vars",
			UserID:         "user-vars",
			Name:           "base1",
			Type:           PromptTypeIdeas,
			PromptTemplate: "Genera {ideas} ideas sobre {name} con temas {related_topics}",
			Active:         true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		// WHEN extracting template variables
		variables := prompt.GetTemplateVariables()

		// THEN it should identify all variables
		expectedVariables := []string{"ideas", "name", "related_topics"}
		if len(variables) != len(expectedVariables) {
			t.Errorf("Expected %d variables, got %d", len(expectedVariables), len(variables))
		}

		for _, expected := range expectedVariables {
			found := false
			for _, actual := range variables {
				if actual == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected variable '%s' not found in: %v", expected, variables)
			}
		}
	})
}

// NEW validation method for the refactored Prompt entity
func (p *Prompt) ValidateRefactored() error {
	// Validate ID
	if p.ID == "" {
		return fmt.Errorf("prompt ID cannot be empty")
	}

	// Validate user ID
	if p.UserID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	// Validate name (new field replacing StyleName as identifier)
	if p.Name == "" {
		return fmt.Errorf("prompt name cannot be empty")
	}

	trimmedName := strings.TrimSpace(p.Name)
	if trimmedName == "" {
		return fmt.Errorf("prompt name cannot be only whitespace")
	}

	// Validate type
	if p.Type != PromptTypeIdeas && p.Type != PromptTypeDrafts {
		return fmt.Errorf("invalid prompt type: must be '%s' or '%s'", PromptTypeIdeas, PromptTypeDrafts)
	}

	// Validate prompt template
	if p.PromptTemplate == "" {
		return fmt.Errorf("prompt template cannot be empty")
	}

	trimmed := strings.TrimSpace(p.PromptTemplate)
	if trimmed == "" {
		return fmt.Errorf("prompt template cannot be only whitespace")
	}

	if len(trimmed) < MinPromptTemplateLength {
		return fmt.Errorf("prompt template too short (minimum %d characters)", MinPromptTemplateLength)
	}

	// Validate timestamps
	if p.CreatedAt.IsZero() {
		return fmt.Errorf("created timestamp cannot be zero")
	}

	if p.UpdatedAt.IsZero() {
		return fmt.Errorf("updated timestamp cannot be zero")
	}

	if p.CreatedAt.After(time.Now()) {
		return fmt.Errorf("created timestamp cannot be in the future")
	}

	if p.UpdatedAt.Before(p.CreatedAt) {
		return fmt.Errorf("updated timestamp cannot be before created timestamp")
	}

	return nil
}

// NEW method to extract template variables from prompt template
func (p *Prompt) GetTemplateVariables() []string {
	// Extract variables in the format {variable_name}
	re := regexp.MustCompile(`\{([^}]+)\}`)
	matches := re.FindAllStringSubmatch(p.PromptTemplate, -1)

	variables := make([]string, 0, len(matches))
	seen := make(map[string]bool)

	for _, match := range matches {
		if len(match) > 1 {
			variable := match[1]
			if !seen[variable] {
				seen[variable] = true
				variables = append(variables, variable)
			}
		}
	}

	return variables
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
