package usecases

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/linkgen-ai/backend/src/domain/entities"
)

// TDD Red tests for Issue 3 (Fase 1) focused on GenerateIdeasForUser path.

func TestGenerateIdeasForUser_UsesTopicPromptReference(t *testing.T) {
	ctx := context.Background()

	user := &entities.User{
		ID:        "user-t1",
		Email:     "user-t1@example.com",
		Language:  "es",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Active:    true,
	}

	topic := &entities.Topic{
		ID:            "topic-t1",
		UserID:        user.ID,
		Name:          "DevOps Pipelines",
		Description:   "Automatización CI/CD",
		Ideas:         2,
		Prompt:        "custom-topic",
		RelatedTopics: []string{"CI/CD", "Kubernetes"},
		Active:        true,
		CreatedAt:     time.Now(),
	}

	defaultPrompt := &entities.Prompt{
		ID:             "prompt-default",
		UserID:         user.ID,
		Name:           "base1",
		Type:           entities.PromptTypeIdeas,
		PromptTemplate: "DEFAULT TEMPLATE for {name} with {ideas} ideas",
		Active:         true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	customPrompt := &entities.Prompt{
		ID:             "prompt-custom",
		UserID:         user.ID,
		Name:           "custom-topic",
		Type:           entities.PromptTypeIdeas,
		PromptTemplate: "CUSTOM TEMPLATE for {name} with {ideas} ideas in {language}",
		Active:         true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	llmResponse := `{"ideas": ["Idea 1", "Idea 2"]}`

	userRepo := &stubUserRepo{user: user}
	topicRepo := &stubTopicRepo{topics: map[string]*entities.Topic{topic.ID: topic}}
	promptRepo := &stubPromptsRepo{
		promptsByName: map[string]*entities.Prompt{
			customPrompt.Name: customPrompt,
		},
		activeByType: map[entities.PromptType][]*entities.Prompt{
			entities.PromptTypeIdeas: {defaultPrompt, customPrompt},
		},
	}
	ideaRepo := &stubIdeasRepo{}
	llm := &stubLLM{response: llmResponse}

	useCase := NewGenerateIdeasUseCase(userRepo, topicRepo, ideaRepo, promptRepo, llm)

	ideas, err := useCase.GenerateIdeasForUser(ctx, user.ID, 5)

	if err != nil {
		t.Fatalf("expected ideas generation without error, got: %v", err)
	}

	if len(ideas) != topic.Ideas {
		t.Fatalf("expected %d ideas but got %d", topic.Ideas, len(ideas))
	}

	if len(llm.prompts) == 0 {
		t.Fatalf("expected LLM to receive at least one prompt")
	}

	if !strings.Contains(llm.prompts[0], "CUSTOM TEMPLATE") {
		t.Fatalf("expected GenerateIdeasForUser to use topic prompt '%s', got prompt: %s", topic.Prompt, llm.prompts[0])
	}
}

func TestGenerateIdeasForUser_RespectsTopicIdeasCount(t *testing.T) {
	ctx := context.Background()

	user := &entities.User{
		ID:        "user-t2",
		Email:     "user-t2@example.com",
		Language:  "es",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Active:    true,
	}

	topic := &entities.Topic{
		ID:        "topic-t2",
		UserID:    user.ID,
		Name:      "AI Safety",
		Ideas:     2,
		Prompt:    "base1",
		Active:    true,
		CreatedAt: time.Now(),
	}

	prompt := &entities.Prompt{
		ID:             "prompt-base1",
		UserID:         user.ID,
		Name:           "base1",
		Type:           entities.PromptTypeIdeas,
		PromptTemplate: "Generate {ideas} ideas about {name} in {language}",
		Active:         true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	llmResponse := `{"ideas": ["Idea 1", "Idea 2", "Idea 3", "Idea 4", "Idea 5"]}`

	userRepo := &stubUserRepo{user: user}
	topicRepo := &stubTopicRepo{topics: map[string]*entities.Topic{topic.ID: topic}}
	promptRepo := &stubPromptsRepo{
		promptsByName: map[string]*entities.Prompt{
			prompt.Name: prompt,
		},
		activeByType: map[entities.PromptType][]*entities.Prompt{
			entities.PromptTypeIdeas: {prompt},
		},
	}
	ideaRepo := &stubIdeasRepo{}
	llm := &stubLLM{response: llmResponse}

	useCase := NewGenerateIdeasUseCase(userRepo, topicRepo, ideaRepo, promptRepo, llm)

	ideas, err := useCase.GenerateIdeasForUser(ctx, user.ID, 5)

	if err != nil {
		t.Fatalf("expected ideas generation without error, got: %v", err)
	}

	if len(ideas) != topic.Ideas {
		t.Fatalf("expected exactly %d ideas saved from LLM response, got %d", topic.Ideas, len(ideas))
	}

	if len(ideaRepo.saved) != topic.Ideas {
		t.Fatalf("expected repository to store %d ideas, got %d", topic.Ideas, len(ideaRepo.saved))
	}

	if len(llm.prompts) == 0 || strings.Contains(llm.prompts[0], "5 ideas") {
		t.Fatalf("expected prompt to request %d ideas from topic, got prompt: %s", topic.Ideas, llm.prompts[0])
	}
}
