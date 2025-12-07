package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	llmclient "github.com/linkgen-ai/backend/src/infrastructure/http/llm"
)

// TestLLMHTTPClientCreation validates LLM HTTP client initialization
func TestLLMHTTPClientCreation(t *testing.T) {
	tests := []struct {
		name        string
		baseURL     string
		timeout     time.Duration
		maxRetries  int
		expectError bool
	}{
		{
			name:        "create client with valid configuration",
			baseURL:     "http://100.105.212.98:8317",
			timeout:     30 * time.Second,
			maxRetries:  3,
			expectError: false,
		},
		{
			name:        "create client with custom timeout",
			baseURL:     "http://100.105.212.98:8317",
			timeout:     60 * time.Second,
			maxRetries:  5,
			expectError: false,
		},
		{
			name:        "create client with empty URL",
			baseURL:     "",
			timeout:     30 * time.Second,
			maxRetries:  3,
			expectError: true,
		},
		{
			name:        "create client with invalid URL",
			baseURL:     "not-a-url",
			timeout:     30 * time.Second,
			maxRetries:  3,
			expectError: true,
		},
		{
			name:        "create client with zero timeout",
			baseURL:     "http://100.105.212.98:8317",
			timeout:     0,
			maxRetries:  3,
			expectError: true,
		},
		{
			name:        "create client with negative max retries",
			baseURL:     "http://100.105.212.98:8317",
			timeout:     30 * time.Second,
			maxRetries:  -1,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := llmclient.Config{
				BaseURL:    tt.baseURL,
				Timeout:    tt.timeout,
				MaxRetries: tt.maxRetries,
			}

			client, err := llmclient.NewLLMHTTPClient(config)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
				if client != nil {
					t.Errorf("expected nil client but got %v", client)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if client == nil {
					t.Errorf("expected client but got nil")
				}
			}
		})
	}
}

// TestGenerateIdeas validates idea generation from LLM
func TestGenerateIdeas(t *testing.T) {
	tests := []struct {
		name        string
		topic       string
		count       int
		mockStatus  int
		mockBody    string
		expectError bool
		expectIdeas int
	}{
		{
			name:        "generate ideas successfully",
			topic:       "Go concurrency patterns",
			count:       5,
			mockStatus:  http.StatusOK,
			mockBody:    `{"choices": [{"message": {"role": "assistant", "content": "{\"ideas\": [\"idea1\", \"idea2\", \"idea3\", \"idea4\", \"idea5\"]}"}}]}`,
			expectError: false,
			expectIdeas: 5,
		},
		{
			name:        "generate ideas with different topic",
			topic:       "Machine Learning in Go",
			count:       3,
			mockStatus:  http.StatusOK,
			mockBody:    `{"choices": [{"message": {"role": "assistant", "content": "{\"ideas\": [\"idea1\", \"idea2\", \"idea3\"]}"}}]}`,
			expectError: false,
			expectIdeas: 3,
		},
		{
			name:        "handle empty topic",
			topic:       "",
			count:       5,
			mockStatus:  0,
			mockBody:    "",
			expectError: true,
			expectIdeas: 0,
		},
		{
			name:        "handle zero count",
			topic:       "Testing in Go",
			count:       0,
			mockStatus:  0,
			mockBody:    "",
			expectError: true,
			expectIdeas: 0,
		},
		{
			name:        "handle negative count",
			topic:       "Testing in Go",
			count:       -1,
			mockStatus:  0,
			mockBody:    "",
			expectError: true,
			expectIdeas: 0,
		},
		{
			name:        "handle LLM server error 500",
			topic:       "Go performance",
			count:       5,
			mockStatus:  http.StatusInternalServerError,
			mockBody:    `{"error": {"message": "internal server error"}}`,
			expectError: true,
			expectIdeas: 0,
		},
		{
			name:        "handle LLM bad request 400",
			topic:       "Invalid topic",
			count:       5,
			mockStatus:  http.StatusBadRequest,
			mockBody:    `{"error": {"message": "bad request"}}`,
			expectError: true,
			expectIdeas: 0,
		},
		{
			name:        "handle malformed JSON response",
			topic:       "Go testing",
			count:       5,
			mockStatus:  http.StatusOK,
			mockBody:    `{invalid json`,
			expectError: true,
			expectIdeas: 0,
		},
		{
			name:        "handle empty response body",
			topic:       "Go modules",
			count:       5,
			mockStatus:  http.StatusOK,
			mockBody:    "",
			expectError: true,
			expectIdeas: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip validation errors that don't need a server
			if tt.topic == "" || tt.count <= 0 {
				config := llmclient.Config{
					BaseURL:    "http://localhost:9999",
					Timeout:    1 * time.Second,
					MaxRetries: 0,
				}
				client, _ := llmclient.NewLLMHTTPClient(config)
				
				ctx := context.Background()
				ideas, err := client.GenerateIdeas(ctx, tt.topic, tt.count)

				if !tt.expectError && err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if tt.expectError && err == nil {
					t.Errorf("expected error but got nil")
				}
				if len(ideas) != tt.expectIdeas {
					t.Errorf("expected %d ideas, got %d", tt.expectIdeas, len(ideas))
				}
				return
			}

			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.mockStatus)
				w.Write([]byte(tt.mockBody))
			}))
			defer server.Close()

			config := llmclient.Config{
				BaseURL:    server.URL,
				Timeout:    5 * time.Second,
				MaxRetries: 0,
			}
			client, err := llmclient.NewLLMHTTPClient(config)
			if err != nil {
				t.Fatalf("failed to create client: %v", err)
			}

			ctx := context.Background()
			ideas, err := client.GenerateIdeas(ctx, tt.topic, tt.count)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if len(ideas) != tt.expectIdeas {
					t.Errorf("expected %d ideas, got %d", tt.expectIdeas, len(ideas))
				}
			}
		})
	}
}

// TestGenerateDrafts validates draft generation from LLM
func TestGenerateDrafts(t *testing.T) {
	tests := []struct {
		name           string
		idea           string
		userContext    string
		mockStatus     int
		mockBody       string
		expectError    bool
		expectPosts    int
		expectArticles int
	}{
		{
			name:           "generate drafts successfully",
			idea:           "Write about Go concurrency",
			userContext:    "Software engineer, 5 years experience",
			mockStatus:     http.StatusOK,
			mockBody:       `{"choices": [{"message": {"role": "assistant", "content": "{\"posts\": [\"post1\", \"post2\", \"post3\", \"post4\", \"post5\"], \"articles\": [\"article1\"]}"}}]}`,
			expectError:    false,
			expectPosts:    5,
			expectArticles: 1,
		},
		{
			name:           "handle empty idea",
			idea:           "",
			userContext:    "Developer",
			mockStatus:     0,
			mockBody:       "",
			expectError:    true,
			expectPosts:    0,
			expectArticles: 0,
		},
		{
			name:           "handle empty user context",
			idea:           "Go performance",
			userContext:    "",
			mockStatus:     0,
			mockBody:       "",
			expectError:    true,
			expectPosts:    0,
			expectArticles: 0,
		},
		{
			name:           "handle LLM server error 500",
			idea:           "Testing patterns",
			userContext:    "Developer",
			mockStatus:     http.StatusInternalServerError,
			mockBody:       `{"error": {"message": "server error"}}`,
			expectError:    true,
			expectPosts:    0,
			expectArticles: 0,
		},
		{
			name:           "handle malformed response",
			idea:           "Go modules",
			userContext:    "Developer",
			mockStatus:     http.StatusOK,
			mockBody:       `{invalid json`,
			expectError:    true,
			expectPosts:    0,
			expectArticles: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip validation errors that don't need a server
			if tt.idea == "" || tt.userContext == "" {
				config := llmclient.Config{
					BaseURL:    "http://localhost:9999",
					Timeout:    1 * time.Second,
					MaxRetries: 0,
				}
				client, _ := llmclient.NewLLMHTTPClient(config)

				ctx := context.Background()
				draftSet, err := client.GenerateDrafts(ctx, tt.idea, tt.userContext)

				if tt.expectError && err == nil {
					t.Errorf("expected error but got nil")
				}
				if !tt.expectError && err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if len(draftSet.Posts) != tt.expectPosts {
					t.Errorf("expected %d posts, got %d", tt.expectPosts, len(draftSet.Posts))
				}
				return
			}

			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.mockStatus)
				w.Write([]byte(tt.mockBody))
			}))
			defer server.Close()

			config := llmclient.Config{
				BaseURL:    server.URL,
				Timeout:    5 * time.Second,
				MaxRetries: 0,
			}
			client, err := llmclient.NewLLMHTTPClient(config)
			if err != nil {
				t.Fatalf("failed to create client: %v", err)
			}

			ctx := context.Background()
			draftSet, err := client.GenerateDrafts(ctx, tt.idea, tt.userContext)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if len(draftSet.Posts) != tt.expectPosts {
					t.Errorf("expected %d posts, got %d", tt.expectPosts, len(draftSet.Posts))
				}
				if len(draftSet.Articles) != tt.expectArticles {
					t.Errorf("expected %d articles, got %d", tt.expectArticles, len(draftSet.Articles))
				}
			}
		})
	}
}

// TestRefineDraft validates draft refinement with conversational context
func TestRefineDraft(t *testing.T) {
	tests := []struct {
		name        string
		draft       string
		userPrompt  string
		history     []string
		mockStatus  int
		mockBody    string
		expectError bool
		expectEmpty bool
	}{
		{
			name:        "refine draft successfully",
			draft:       "Original draft content",
			userPrompt:  "Make it more engaging",
			history:     []string{},
			mockStatus:  http.StatusOK,
			mockBody:    `{"choices": [{"message": {"role": "assistant", "content": "{\"refined\": \"Refined draft content\"}"}}]}`,
			expectError: false,
			expectEmpty: false,
		},
		{
			name:        "handle empty draft",
			draft:       "",
			userPrompt:  "Make it better",
			history:     []string{},
			mockStatus:  0,
			mockBody:    "",
			expectError: true,
			expectEmpty: true,
		},
		{
			name:        "handle empty user prompt",
			draft:       "Some draft",
			userPrompt:  "",
			history:     []string{},
			mockStatus:  0,
			mockBody:    "",
			expectError: true,
			expectEmpty: true,
		},
		{
			name:        "handle LLM server error 500",
			draft:       "Draft content",
			userPrompt:  "Refine please",
			history:     []string{},
			mockStatus:  http.StatusInternalServerError,
			mockBody:    `{"error": {"message": "server error"}}`,
			expectError: true,
			expectEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip validation errors that don't need a server
			if tt.draft == "" || tt.userPrompt == "" {
				config := llmclient.Config{
					BaseURL:    "http://localhost:9999",
					Timeout:    1 * time.Second,
					MaxRetries: 0,
				}
				client, _ := llmclient.NewLLMHTTPClient(config)

				ctx := context.Background()
				refined, err := client.RefineDraft(ctx, tt.draft, tt.userPrompt, tt.history)

				if tt.expectError && err == nil {
					t.Errorf("expected error but got nil")
				}
				if !tt.expectError && err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if tt.expectEmpty && refined != "" {
					t.Errorf("expected empty refined content but got: %s", refined)
				}
				return
			}

			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.mockStatus)
				w.Write([]byte(tt.mockBody))
			}))
			defer server.Close()

			config := llmclient.Config{
				BaseURL:    server.URL,
				Timeout:    5 * time.Second,
				MaxRetries: 0,
			}
			client, err := llmclient.NewLLMHTTPClient(config)
			if err != nil {
				t.Fatalf("failed to create client: %v", err)
			}

			ctx := context.Background()
			refined, err := client.RefineDraft(ctx, tt.draft, tt.userPrompt, tt.history)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if tt.expectEmpty && refined != "" {
					t.Errorf("expected empty refined content but got: %s", refined)
				}
				if !tt.expectEmpty && refined == "" {
					t.Errorf("expected refined content but got empty string")
				}
			}
		})
	}
}

// TestLLMClientContextCancellation validates context cancellation handling
func TestLLMClientContextCancellation(t *testing.T) {
	// Create a slow server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"role": "assistant", "content": `{"ideas": ["test"]}`}},
			},
		})
	}))
	defer server.Close()

	config := llmclient.Config{
		BaseURL:    server.URL,
		Timeout:    10 * time.Second,
		MaxRetries: 0,
	}
	client, err := llmclient.NewLLMHTTPClient(config)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	t.Run("cancel GenerateIdeas context", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		_, err := client.GenerateIdeas(ctx, "test topic", 5)
		if err == nil {
			t.Error("expected context error but got nil")
		}
	})
}

// TestLLMClientTimeout validates timeout handling
func TestLLMClientTimeout(t *testing.T) {
	t.Run("request exceeds timeout", func(t *testing.T) {
		// Create a slow server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(2 * time.Second)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		config := llmclient.Config{
			BaseURL:    server.URL,
			Timeout:    100 * time.Millisecond,
			MaxRetries: 0,
		}
		client, err := llmclient.NewLLMHTTPClient(config)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		ctx := context.Background()
		_, err = client.GenerateIdeas(ctx, "test topic", 5)
		if err == nil {
			t.Error("expected timeout error but got nil")
		}
	})
}
