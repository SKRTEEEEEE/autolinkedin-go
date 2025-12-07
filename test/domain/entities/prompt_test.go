package entities_test

import (
	"testing"
	"time"

	"github.com/linkgen-ai/backend/src/domain/entities"
)

func TestPrompt_Validate(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		prompt  *entities.Prompt
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid ideas prompt",
			prompt: &entities.Prompt{
				ID:             "507f1f77bcf86cd799439011",
				UserID:         "000000000000000000000001",
				Type:           entities.PromptTypeIdeas,
				PromptTemplate: "Generate ideas for: {topic}",
				Active:         true,
				CreatedAt:      now,
				UpdatedAt:      now,
			},
			wantErr: false,
		},
		{
			name: "valid drafts prompt with style",
			prompt: &entities.Prompt{
				ID:             "507f1f77bcf86cd799439011",
				UserID:         "000000000000000000000001",
				Type:           entities.PromptTypeDrafts,
				StyleName:      "professional",
				PromptTemplate: "Create professional content based on: {idea}",
				Active:         true,
				CreatedAt:      now,
				UpdatedAt:      now,
			},
			wantErr: false,
		},
		{
			name: "missing prompt ID",
			prompt: &entities.Prompt{
				ID:             "",
				UserID:         "000000000000000000000001",
				Type:           entities.PromptTypeIdeas,
				PromptTemplate: "Generate ideas for: {topic}",
				Active:         true,
				CreatedAt:      now,
				UpdatedAt:      now,
			},
			wantErr: true,
			errMsg:  "prompt ID cannot be empty",
		},
		{
			name: "missing user ID",
			prompt: &entities.Prompt{
				ID:             "507f1f77bcf86cd799439011",
				UserID:         "",
				Type:           entities.PromptTypeIdeas,
				PromptTemplate: "Generate ideas for: {topic}",
				Active:         true,
				CreatedAt:      now,
				UpdatedAt:      now,
			},
			wantErr: true,
			errMsg:  "user ID cannot be empty",
		},
		{
			name: "invalid prompt type",
			prompt: &entities.Prompt{
				ID:             "507f1f77bcf86cd799439011",
				UserID:         "000000000000000000000001",
				Type:           "invalid",
				PromptTemplate: "Generate ideas for: {topic}",
				Active:         true,
				CreatedAt:      now,
				UpdatedAt:      now,
			},
			wantErr: true,
			errMsg:  "invalid prompt type",
		},
		{
			name: "drafts prompt missing style name",
			prompt: &entities.Prompt{
				ID:             "507f1f77bcf86cd799439011",
				UserID:         "000000000000000000000001",
				Type:           entities.PromptTypeDrafts,
				StyleName:      "",
				PromptTemplate: "Create professional content based on: {idea}",
				Active:         true,
				CreatedAt:      now,
				UpdatedAt:      now,
			},
			wantErr: true,
			errMsg:  "style name is required for draft prompts",
		},
		{
			name: "empty prompt template",
			prompt: &entities.Prompt{
				ID:             "507f1f77bcf86cd799439011",
				UserID:         "000000000000000000000001",
				Type:           entities.PromptTypeIdeas,
				PromptTemplate: "",
				Active:         true,
				CreatedAt:      now,
				UpdatedAt:      now,
			},
			wantErr: true,
			errMsg:  "prompt template cannot be empty",
		},
		{
			name: "prompt template too short",
			prompt: &entities.Prompt{
				ID:             "507f1f77bcf86cd799439011",
				UserID:         "000000000000000000000001",
				Type:           entities.PromptTypeIdeas,
				PromptTemplate: "short",
				Active:         true,
				CreatedAt:      now,
				UpdatedAt:      now,
			},
			wantErr: true,
			errMsg:  "prompt template too short",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.prompt.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Prompt.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if err.Error() != tt.errMsg && !contains(err.Error(), tt.errMsg) {
					t.Errorf("Prompt.Validate() error message = %v, want to contain %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestPrompt_UpdateTemplate(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		prompt      *entities.Prompt
		newTemplate string
		wantErr     bool
	}{
		{
			name: "valid template update",
			prompt: &entities.Prompt{
				ID:             "507f1f77bcf86cd799439011",
				UserID:         "000000000000000000000001",
				Type:           entities.PromptTypeIdeas,
				PromptTemplate: "Old template for ideas",
				Active:         true,
				CreatedAt:      now,
				UpdatedAt:      now,
			},
			newTemplate: "New template for generating ideas about {topic}",
			wantErr:     false,
		},
		{
			name: "invalid template update - too short",
			prompt: &entities.Prompt{
				ID:             "507f1f77bcf86cd799439011",
				UserID:         "000000000000000000000001",
				Type:           entities.PromptTypeIdeas,
				PromptTemplate: "Old template for ideas",
				Active:         true,
				CreatedAt:      now,
				UpdatedAt:      now,
			},
			newTemplate: "short",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldUpdatedAt := tt.prompt.UpdatedAt
			err := tt.prompt.UpdateTemplate(tt.newTemplate)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("Prompt.UpdateTemplate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr {
				if tt.prompt.PromptTemplate != tt.newTemplate {
					t.Errorf("Prompt.UpdateTemplate() template = %v, want %v", tt.prompt.PromptTemplate, tt.newTemplate)
				}
				if !tt.prompt.UpdatedAt.After(oldUpdatedAt) {
					t.Errorf("Prompt.UpdateTemplate() should update UpdatedAt timestamp")
				}
			}
		})
	}
}

func TestPrompt_ActivateDeactivate(t *testing.T) {
	now := time.Now()

	prompt := &entities.Prompt{
		ID:             "507f1f77bcf86cd799439011",
		UserID:         "000000000000000000000001",
		Type:           entities.PromptTypeIdeas,
		PromptTemplate: "Generate ideas for: {topic}",
		Active:         true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	// Test deactivate
	oldUpdatedAt := prompt.UpdatedAt
	time.Sleep(10 * time.Millisecond)
	prompt.Deactivate()

	if prompt.Active {
		t.Error("Prompt.Deactivate() should set Active to false")
	}
	if !prompt.UpdatedAt.After(oldUpdatedAt) {
		t.Error("Prompt.Deactivate() should update UpdatedAt timestamp")
	}

	// Test activate
	oldUpdatedAt = prompt.UpdatedAt
	time.Sleep(10 * time.Millisecond)
	prompt.Activate()

	if !prompt.Active {
		t.Error("Prompt.Activate() should set Active to true")
	}
	if !prompt.UpdatedAt.After(oldUpdatedAt) {
		t.Error("Prompt.Activate() should update UpdatedAt timestamp")
	}
}

func TestPrompt_TypeChecks(t *testing.T) {
	now := time.Now()

	ideasPrompt := &entities.Prompt{
		ID:             "507f1f77bcf86cd799439011",
		UserID:         "000000000000000000000001",
		Type:           entities.PromptTypeIdeas,
		PromptTemplate: "Generate ideas for: {topic}",
		Active:         true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	draftsPrompt := &entities.Prompt{
		ID:             "507f1f77bcf86cd799439012",
		UserID:         "000000000000000000000001",
		Type:           entities.PromptTypeDrafts,
		StyleName:      "professional",
		PromptTemplate: "Create content based on: {idea}",
		Active:         true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if !ideasPrompt.IsIdeasPrompt() {
		t.Error("IsIdeasPrompt() should return true for ideas prompt")
	}
	if ideasPrompt.IsDraftsPrompt() {
		t.Error("IsDraftsPrompt() should return false for ideas prompt")
	}

	if draftsPrompt.IsIdeasPrompt() {
		t.Error("IsIdeasPrompt() should return false for drafts prompt")
	}
	if !draftsPrompt.IsDraftsPrompt() {
		t.Error("IsDraftsPrompt() should return true for drafts prompt")
	}
}

func TestPrompt_IsOwnedBy(t *testing.T) {
	now := time.Now()

	prompt := &entities.Prompt{
		ID:             "507f1f77bcf86cd799439011",
		UserID:         "000000000000000000000001",
		Type:           entities.PromptTypeIdeas,
		PromptTemplate: "Generate ideas for: {topic}",
		Active:         true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if !prompt.IsOwnedBy("000000000000000000000001") {
		t.Error("IsOwnedBy() should return true for correct user")
	}

	if prompt.IsOwnedBy("000000000000000000000002") {
		t.Error("IsOwnedBy() should return false for different user")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr))
}
