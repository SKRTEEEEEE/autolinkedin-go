package services

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/linkgen-ai/backend/src/domain/entities"
	"github.com/linkgen-ai/backend/src/domain/interfaces"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// PromptService handles prompt synchronization with seed files and management
type PromptService struct {
	promptsRepo interfaces.PromptsRepository
	userRepo    interfaces.UserRepository
	loader      *PromptLoader
	logger      interfaces.Logger
}

// NewPromptService creates a new PromptService instance
func NewPromptService(
	promptsRepo interfaces.PromptsRepository,
	userRepo interfaces.UserRepository,
	logger interfaces.Logger,
) *PromptService {
	return &PromptService{
		promptsRepo: promptsRepo,
		userRepo:    userRepo,
		loader:      NewPromptLoader(logger),
		logger:      logger,
	}
}

// SyncSeedPrompts synchronizes seed prompts with the database for all users
func (ps *PromptService) SyncSeedPrompts(ctx context.Context, seedDir string) error {
	if seedDir == "" {
		return fmt.Errorf("seed directory path cannot be empty")
	}

	// Check if seed directory exists
	if _, err := os.Stat(seedDir); os.IsNotExist(err) {
		return fmt.Errorf("seed directory does not exist: %s", seedDir)
	}

	// Load prompts from seed directory
	promptFiles, err := ps.loader.LoadPromptsFromDir(seedDir)
	if err != nil {
		return fmt.Errorf("failed to load prompts from seed directory: %w", err)
	}

	if len(promptFiles) == 0 {
		ps.logger.Info("No prompt files found in seed directory", "directory", seedDir)
		return nil
	}

	// Get all users to sync their prompts
	users, err := ps.getAllUsers(ctx)
	if err != nil {
		return fmt.Errorf("failed to get users: %w", err)
	}

	// Sync prompts for each user
	for _, user := range users {
		if err := ps.syncUserPrompts(ctx, user.ID, promptFiles); err != nil {
			ps.logger.Warn("Failed to sync prompts for user", "user_id", user.ID, "error", err)
			// Continue with other users even if one fails
		}
	}

	ps.logger.Info("Seed prompts synchronized successfully",
		"prompt_count", len(promptFiles),
		"user_count", len(users))

	return nil
}

// SyncSeedPromptsForUser synchronizes seed prompts with the database for a specific user
func (ps *PromptService) SyncSeedPromptsForUser(ctx context.Context, userID string, seedDir string) error {
	if seedDir == "" {
		return fmt.Errorf("seed directory path cannot be empty")
	}

	if userID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	// Check if user exists
	user, err := ps.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to find user: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user not found: %s", userID)
	}

	// Load prompts from seed directory
	promptFiles, err := ps.loader.LoadPromptsFromDir(seedDir)
	if err != nil {
		return fmt.Errorf("failed to load prompts from seed directory: %w", err)
	}

	if len(promptFiles) == 0 {
		ps.logger.Info("No prompt files found in seed directory", "directory", seedDir)
		return nil
	}

	// Sync prompts for the user
	if err := ps.syncUserPrompts(ctx, userID, promptFiles); err != nil {
		return fmt.Errorf("failed to sync prompts for user %s: %w", userID, err)
	}

	ps.logger.Info("Seed prompts synchronized for user",
		"user_id", userID,
		"prompt_count", len(promptFiles))

	return nil
}

// CreateCustomPrompt creates a new custom prompt for a user
func (ps *PromptService) CreateCustomPrompt(
	ctx context.Context,
	userID string,
	name string,
	promptType entities.PromptType,
	template string,
) (*entities.Prompt, error) {
	// Validate input
	if userID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}

	if name == "" {
		return nil, fmt.Errorf("prompt name cannot be empty")
	}

	if template == "" {
		return nil, fmt.Errorf("prompt template cannot be empty")
	}

	// Check if prompt with this name already exists for user
	existing, err := ps.promptsRepo.FindByName(ctx, userID, name)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing prompt: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("prompt with name '%s' already exists for user", name)
	}

	// Create new prompt
	now := time.Now()
	prompt := &entities.Prompt{
		ID:             primitive.NewObjectID().Hex(),
		UserID:         userID,
		Type:           promptType,
		Name:           name,
		StyleName:      name, // For backward compatibility
		PromptTemplate: template,
		Active:         true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	// Validate prompt
	if err := prompt.Validate(); err != nil {
		return nil, fmt.Errorf("invalid prompt: %w", err)
	}

	// Save prompt
	promptID, err := ps.promptsRepo.Create(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to create prompt: %w", err)
	}

	// Get the created prompt with ID
	created, err := ps.promptsRepo.FindByID(ctx, promptID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve created prompt: %w", err)
	}

	ps.logger.Info("Custom prompt created",
		"user_id", userID,
		"prompt_id", promptID,
		"prompt_name", name,
		"prompt_type", promptType)

	return created, nil
}

// UpdateCustomPrompt updates an existing custom prompt
func (ps *PromptService) UpdateCustomPrompt(
	ctx context.Context,
	userID string,
	promptID string,
	name string,
	template string,
) (*entities.Prompt, error) {
	// Validate input
	if userID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}

	if promptID == "" {
		return nil, fmt.Errorf("prompt ID cannot be empty")
	}

	// Get existing prompt
	prompt, err := ps.promptsRepo.FindByID(ctx, promptID)
	if err != nil {
		return nil, fmt.Errorf("failed to find prompt: %w", err)
	}
	if prompt == nil {
		return nil, fmt.Errorf("prompt not found: %s", promptID)
	}

	// Check ownership
	if !prompt.IsOwnedBy(userID) {
		return nil, fmt.Errorf("prompt does not belong to user")
	}

	// Check if new name conflicts with another prompt
	if name != "" && name != prompt.Name {
		existing, err := ps.promptsRepo.FindByName(ctx, userID, name)
		if err != nil {
			return nil, fmt.Errorf("failed to check existing prompt: %w", err)
		}
		if existing != nil && existing.ID != promptID {
			return nil, fmt.Errorf("prompt with name '%s' already exists", name)
		}
		prompt.Name = name
		prompt.StyleName = name // For backward compatibility
	}

	// Update template if provided
	if template != "" {
		if err := prompt.UpdateTemplate(template); err != nil {
			return nil, fmt.Errorf("invalid template: %w", err)
		}
	}

	// Update prompt in repository
	if err := ps.promptsRepo.Update(ctx, prompt); err != nil {
		return nil, fmt.Errorf("failed to update prompt: %w", err)
	}

	ps.logger.Info("Custom prompt updated",
		"user_id", userID,
		"prompt_id", promptID,
		"prompt_name", prompt.Name)

	return prompt, nil
}

// DeleteCustomPrompt deletes a custom prompt
func (ps *PromptService) DeleteCustomPrompt(ctx context.Context, userID string, promptID string) error {
	// Validate input
	if userID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	if promptID == "" {
		return fmt.Errorf("prompt ID cannot be empty")
	}

	// Get existing prompt
	prompt, err := ps.promptsRepo.FindByID(ctx, promptID)
	if err != nil {
		return fmt.Errorf("failed to find prompt: %w", err)
	}
	if prompt == nil {
		return fmt.Errorf("prompt not found: %s", promptID)
	}

	// Check ownership
	if !prompt.IsOwnedBy(userID) {
		return fmt.Errorf("prompt does not belong to user")
	}

	// Delete prompt
	if err := ps.promptsRepo.Delete(ctx, promptID); err != nil {
		return fmt.Errorf("failed to delete prompt: %w", err)
	}

	ps.logger.Info("Custom prompt deleted",
		"user_id", userID,
		"prompt_id", promptID,
		"prompt_name", prompt.Name)

	return nil
}

// ResetToSeedPrompts resets user's prompts to seed versions, removing custom prompts
func (ps *PromptService) ResetToSeedPrompts(ctx context.Context, userID string, seedDir string) error {
	if userID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	if seedDir == "" {
		return fmt.Errorf("seed directory path cannot be empty")
	}

	// Get all user prompts
	userPrompts, err := ps.promptsRepo.ListByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user prompts: %w", err)
	}

	// Delete all custom prompts
	for _, prompt := range userPrompts {
		if err := ps.promptsRepo.Delete(ctx, prompt.ID); err != nil {
			ps.logger.Warn("Failed to delete prompt during reset",
				"prompt_id", prompt.ID,
				"error", err)
		}
	}

	// Sync with seed prompts
	if err := ps.SyncSeedPromptsForUser(ctx, userID, seedDir); err != nil {
		return fmt.Errorf("failed to sync seed prompts after reset: %w", err)
	}

	ps.logger.Info("User prompts reset to seed versions", "user_id", userID)

	return nil
}

// ListUserPrompts returns all prompts for a user
func (ps *PromptService) ListUserPrompts(ctx context.Context, userID string) ([]*entities.Prompt, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}

	prompts, err := ps.promptsRepo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list user prompts: %w", err)
	}

	return prompts, nil
}

// ListUserPromptsByType returns prompts for a user filtered by type
func (ps *PromptService) ListUserPromptsByType(
	ctx context.Context,
	userID string,
	promptType entities.PromptType,
) ([]*entities.Prompt, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}

	prompts, err := ps.promptsRepo.ListByUserIDAndType(ctx, userID, promptType)
	if err != nil {
		return nil, fmt.Errorf("failed to list user prompts by type: %w", err)
	}

	return prompts, nil
}

// GetPromptByName returns a specific prompt by name for a user
func (ps *PromptService) GetPromptByName(ctx context.Context, userID string, name string) (*entities.Prompt, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}

	if name == "" {
		return nil, fmt.Errorf("prompt name cannot be empty")
	}

	prompt, err := ps.promptsRepo.FindByName(ctx, userID, name)
	if err != nil {
		return nil, fmt.Errorf("failed to find prompt by name: %w", err)
	}

	return prompt, nil
}

// syncUserPrompts synchronizes prompts for a specific user
func (ps *PromptService) syncUserPrompts(ctx context.Context, userID string, promptFiles []*PromptFile) error {
	now := time.Now()

	// Get existing prompts for user
	existingPrompts, err := ps.promptsRepo.ListByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get existing prompts: %w", err)
	}

	// Create map of existing prompts by name
	existingMap := make(map[string]*entities.Prompt)
	for _, prompt := range existingPrompts {
		existingMap[prompt.Name] = prompt
	}

	// Process each seed prompt
	for _, promptFile := range promptFiles {
		// Create prompt entity
		prompt, err := ps.loader.CreatePromptsFromFile(userID, []*PromptFile{promptFile})
		if err != nil {
			ps.logger.Warn("Failed to create prompt from file",
				"file_name", promptFile.Name,
				"error", err)
			continue
		}

		if len(prompt) == 0 {
			continue
		}

		promptEntity := prompt[0]

		// Check if prompt already exists
		if existing, exists := existingMap[promptEntity.Name]; exists {
			// Update existing prompt if template changed
			if existing.PromptTemplate != promptEntity.PromptTemplate {
				existing.PromptTemplate = promptEntity.PromptTemplate
				existing.UpdatedAt = now

				if err := ps.promptsRepo.Update(ctx, existing); err != nil {
					ps.logger.Warn("Failed to update existing prompt",
						"prompt_id", existing.ID,
						"error", err)
				}
			}
			// Remove from map to track which prompts are still active
			delete(existingMap, promptEntity.Name)
		} else {
			// Create new prompt
			promptEntity.CreatedAt = now
			promptEntity.UpdatedAt = now

			if _, err := ps.promptsRepo.Create(ctx, promptEntity); err != nil {
				ps.logger.Warn("Failed to create new prompt",
					"prompt_name", promptEntity.Name,
					"error", err)
			}
		}
	}

	// Deactivate prompts that are no longer in seed files
	for _, orphanPrompt := range existingMap {
		// Only deactivate if it's a seed-type prompt (not custom)
		if ps.isSeedPrompt(orphanPrompt) {
			orphanPrompt.Active = false
			orphanPrompt.UpdatedAt = now

			if err := ps.promptsRepo.Update(ctx, orphanPrompt); err != nil {
				ps.logger.Warn("Failed to deactivate orphan prompt",
					"prompt_id", orphanPrompt.ID,
					"error", err)
			}
		}
	}

	return nil
}

// getAllUsers retrieves all users from the repository
func (ps *PromptService) getAllUsers(ctx context.Context) ([]*entities.User, error) {
	// Note: This is a simplified implementation
	// In a real scenario, you'd have a ListAll method in UserRepository
	// For now, we'll use a hardcoded dev user ID as that's what the project uses
	devUser, err := ps.userRepo.FindByEmail(ctx, "dev@example.com")
	if err != nil {
		return nil, fmt.Errorf("failed to find dev user: %w", err)
	}

	if devUser == nil {
		// Create dev user if it doesn't exist
		return []*entities.User{}, nil
	}

	return []*entities.User{devUser}, nil
}

// isSeedPrompt checks if a prompt is a seed-type prompt (not custom)
func (ps *PromptService) isSeedPrompt(prompt *entities.Prompt) bool {
	// Simple heuristic: if prompt ID doesn't contain "custom" or if it matches seed patterns
	// This is a basic implementation - could be enhanced with more sophisticated tracking
	return !strings.Contains(prompt.ID, "custom")
}

// ValidatePromptTemplate validates the syntax of a prompt template
func (ps *PromptService) ValidatePromptTemplate(template string, promptType entities.PromptType) error {
	if template == "" {
		return fmt.Errorf("template cannot be empty")
	}

	// Check for expected variables based on prompt type
	if promptType == entities.PromptTypeIdeas {
		if !strings.Contains(template, "{name}") {
			ps.logger.Warn("Ideas prompt missing recommended variable", "variable", "{name}")
		}
		if !strings.Contains(template, "{ideas}") {
			ps.logger.Warn("Ideas prompt missing recommended variable", "variable", "{ideas}")
		}
	}

	if promptType == entities.PromptTypeDrafts {
		if !strings.Contains(template, "{content}") {
			ps.logger.Warn("Drafts prompt missing recommended variable", "variable", "{content}")
		}
		if !strings.Contains(template, "{user_context}") {
			ps.logger.Warn("Drafts prompt missing recommended variable", "variable", "{user_context}")
		}
	}

	// Check for malformed variable patterns
	variables := ps.ExtractVariables(template)
	for _, variable := range variables {
		if !strings.HasPrefix(variable, "{") || !strings.HasSuffix(variable, "}") {
			return fmt.Errorf("malformed variable format: %s", variable)
		}
	}

	return nil
}

// ExtractVariables extracts all variable placeholders from a template (public method)
func (ps *PromptService) ExtractVariables(template string) []string {
	var variables []string
	start := 0

	for {
		openIdx := strings.Index(template[start:], "{")
		if openIdx == -1 {
			break
		}

		openIdx += start
		closeIdx := strings.Index(template[openIdx:], "}")
		if closeIdx == -1 {
			break
		}

		closeIdx += openIdx + 1
		variable := template[openIdx:closeIdx]
		variables = append(variables, variable)
		start = closeIdx
	}

	return variables
}

// GetSeedPrompts loads and returns all seed prompts without saving to database
func (ps *PromptService) GetSeedPrompts(seedDir string) ([]*PromptFile, error) {
	if seedDir == "" {
		return nil, fmt.Errorf("seed directory path cannot be empty")
	}

	// Check if seed directory exists
	if _, err := os.Stat(seedDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("seed directory does not exist: %s", seedDir)
	}

	// Load prompts from seed directory
	promptFiles, err := ps.loader.LoadPromptsFromDir(seedDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load prompts from seed directory: %w", err)
	}

	return promptFiles, nil
}

// PromptStatistics provides statistics about prompts for a user
type PromptStatistics struct {
	TotalCount   int
	IdeasCount   int
	DraftsCount  int
	ActiveCount  int
	CustomCount  int
	LastSyncedAt *time.Time
}

// GetPromptStatistics returns statistics about prompts for a user
func (ps *PromptService) GetPromptStatistics(ctx context.Context, userID string) (*PromptStatistics, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}

	prompts, err := ps.promptsRepo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list user prompts: %w", err)
	}

	stats := &PromptStatistics{
		TotalCount: len(prompts),
	}

	for _, prompt := range prompts {
		switch prompt.Type {
		case entities.PromptTypeIdeas:
			stats.IdeasCount++
		case entities.PromptTypeDrafts:
			stats.DraftsCount++
		}

		if prompt.Active {
			stats.ActiveCount++
		}

		if !ps.isSeedPrompt(prompt) {
			stats.CustomCount++
		}

		// Track latest update time
		if stats.LastSyncedAt == nil || prompt.UpdatedAt.After(*stats.LastSyncedAt) {
			stats.LastSyncedAt = &prompt.UpdatedAt
		}
	}

	return stats, nil
}
