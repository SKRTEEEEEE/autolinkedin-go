package usecases

import (
	"context"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/linkgen-ai/backend/src/domain/entities"
	"github.com/linkgen-ai/backend/test"
	"github.com/linkgen-ai/backend/src/test/mocks"
)

// TestPromptSystemPerformance tests the performance impact of the new prompt system
func TestPromptSystemPerformance(tt *testing.T) {
	ctx := context.Background()

	tt.Run("should measure prompt variable replacement performance", func(t *testing.T) {
		// GIVEN different prompt templates and topics
		testDB := test.SetupTestDB(t)
		defer test.CleanupTestDB(t, testDB)

		templates := []string{
			"Generate {ideas} ideas about {name}",
			"Generate {ideas} innovative ideas about {name} with priority {priority} focusing on {[keywords]}",
			"Create professional content about {topic_name} based on: {content}\nUser: {user_context}",
			"Complex template with {name}, {category}, {priority}, {[keywords]}, {ideas_count}, {[related_topics]}",
		}

		topics := []*entities.Topic{
			{
				Name:          "React Hooks",
				Category:      "Frontend",
				Priority:      8,
				IdeasCount:    5,
				Keywords:      []string{"react", "hooks", "state"},
				RelatedTopics: []string{"JavaScript", "State Management", "Functional Programming"},
			},
			{
				Name:          "Go Microservices",
				Category:      "Backend",
				Priority:      9,
				IdeasCount:    3,
				Keywords:      []string{"go", "microservices", "grpc"},
				RelatedTopics: []string{"Docker", "Kubernetes", "API Design"},
			},
		}

		iterations := 100

		// WHEN measuring performance
		start := time.Now()
		
		for i := 0; i < iterations; i++ {
			for _, template := range templates {
				for _, topic := range topics {
					// Process template with variables
					_, err := testDB.PromptEngine.ProcessTemplate(template, topic, nil)
					require.NoError(t, err, "Template processing should succeed")
				}
			}
		}
		
		duration := time.Since(start)
		totalOperations := iterations * len(templates) * len(topics)
		avgDuration := duration / time.Duration(totalOperations)

		// THEN should meet performance requirements
		t.Logf("Processed %d operations in %v", totalOperations, duration)
		t.Logf("Average time per operation: %v", avgDuration)

		// Performance assertions
		assert.Less(t, avgDuration, 10*time.Millisecond, "Each operation should be faster than 10ms")
		assert.Less(t, duration, 30*time.Second, "Total should complete within 30 seconds")
	})

	tt.Run("should measure database operations performance with prompt references", func(t *testing.T) {
		// GIVEN database with prompt-enabled topics
		db := setupTestDB(t)
		defer cleanupTestDB(t, db)

		userID := primitive.NewObjectID().Hex()
		topicCount := 100

		// Create test data
		promptRepo := &mocks.MockPromptRepository{}
		topicRepo := &mocks.MockTopicRepository{}
		ideaRepo := &mocks.MockIdeaRepository{}

		// WHEN measuring bulk operations
		start := time.Now()

		// Bulk create topics with prompt references
		for i := 0; i < topicCount; i++ {
			topic := &entities.Topic{
				ID:         primitive.NewObjectID().Hex(),
				UserID:     userID,
				Name:       fmt.Sprintf("Topic %d", i),
				PromptName: "base1",
				CreatedAt:  time.Now(),
			}

			// This will fail until bulk operations with prompts are implemented
			t.Fatal("implement efficient bulk operations with prompt references - FAILING IN TDD RED PHASE")

			_, err := topicRepo.Create(ctx, topic)
			require.NoError(t, err)
		}

		createDuration := time.Since(start)
		t.Logf("Created %d topics in %v", topicCount, createDuration)

		// Measure fetching performance
		start = time.Now()
		topics, err := topicRepo.ListByUserID(ctx, userID)
		require.NoError(t, err)
		
		fetchDuration := time.Since(start)
		t.Logf("Fetched %d topics in %v", len(topics), fetchDuration)

		// THEN should meet performance requirements
		assert.Equal(t, topicCount, len(topics))
		assert.Less(t, createDuration, 2*time.Second, "Create operations should complete within 2 seconds")
		assert.Less(t, fetchDuration, 500*time.Millisecond, "Fetch operation should complete within 500ms")
	})

	tt.Run("should measure LLM prompt generation performance impact", func(t *testing.T) {
		// GIVEN LLM client and prompt templates
		llmClient := &mocks.MockLLMClient{}
		promptProcessor := NewPromptProcessor()

		// Mock different template complexities
		complexTemplates := []struct {
			name     string
			template string
			expectedVars int
		}{
			{
				name:     "simple",
				template: "Generate 5 ideas about {name}",
				expectedVars: 1,
			},
			{
				name:     "medium",
				template: "Generate {ideas} ideas about {name} with {[keywords]} and priority {priority}",
				expectedVars: 4,
			},
			{
				name:     "complex",
				template: `Generate {ideas} professional ideas about {name} in category {category}
Focus on: {[keywords]}
Priority: {priority}
Related: {[related_topics]}
Target: {ideas_count} ideas`,
				expectedVars: 6,
			},
		}

		iterations := 100

		// WHEN measuring template processing + LLM call preparation
		for _, tc := range complexTemplates {
			t.Run(tc.name, func(t *testing.T) {
				start := time.Now()

				for i := 0; i < iterations; i++ {
					// Process template variables
					processedPrompt, err := promptProcessor.ProcessTemplate(
						tc.template,
						&entities.Topic{
							Name:         "Test Topic",
							Category:     "Technology",
							Keywords:     []string{"test", "performance"},
							Priority:     8,
							IdeasCount:   5,
							RelatedTopics: []string{"Related1", "Related2"},
						},
						nil,
					)

					// This will fail until template processing is implemented
					t.Fatal("implement efficient prompt template processing - FAILING IN TDD RED PHASE")

					require.NoError(t, err)
					require.NotEmpty(t, processedPrompt)

					// Simulate LLM call preparation
					_ = prepareLLMRequest(processedPrompt)
				}

				duration := time.Since(start)
				avgDuration := duration / time.Duration(iterations)

				t.Logf("%s template: Processed %d iterations in %v (avg: %v)", 
					tc.name, iterations, duration, avgDuration)

				// Performance assertions based on complexity
				switch tc.expectedVars {
				case 1:
					assert.Less(t, avgDuration, 100*time.Microsecond, "Simple template should be very fast")
				case 4:
					assert.Less(t, avgDuration, 200*time.Microsecond, "Medium template should be fast")
				case 6:
					assert.Less(t, avgDuration, 300*time.Microsecond, "Complex template should still be fast")
				}
			})
		}
	})
}

// TestConcurrentPromptProcessing tests concurrent access to prompt system
func TestConcurrentPromptProcessing(tt *testing.T) {
	ctx := context.Background()

	tt.Run("should handle concurrent prompt processing safely", func(t *testing.T) {
		// GIVEN a prompt processor and shared resources
		promptProcessor := NewPromptProcessor()

		numGoroutines := 50
		operationsPerGoroutine := 100

		var wg sync.WaitGroup
		errors := make(chan error, numGoroutines)

		// WHEN processing prompts concurrently
		start := time.Now()

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				for j := 0; j < operationsPerGoroutine; j++ {
					template := "Generate {ideas} ideas about {name}" + fmt.Sprintf("-%d-%d", id, j)
					topic := &entities.Topic{
						Name:       fmt.Sprintf("Topic-%d", id),
						IdeasCount: j % 10 + 1,
					}

					// This will fail until thread-safe concurrent processing is implemented
					t.Fatal("implement thread-safe concurrent prompt processing - FAILING IN TDD RED PHASE")

					processed, err := promptProcessor.ProcessTemplate(template, topic, nil)
					if err != nil {
						errors <- err
						return
					}

					// Basic validation
					if processed == "" {
						errors <- assert.AnError
						return
					}
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		duration := time.Since(start)
		totalOperations := numGoroutines * operationsPerGoroutine

		// Check for errors
		errorCount := 0
		for err := range errors {
			t.Logf("Concurrent processing error: %v", err)
			errorCount++
		}

		// THEN should handle concurrency without errors
		assert.Equal(t, 0, errorCount, "Should have no concurrent processing errors")
		assert.Less(t, duration, 10*time.Second, "Concurrent processing should be efficient")
		
		t.Logf("Processed %d operations concurrently in %v", totalOperations, duration)
		t.Logf("Throughput: %.2f ops/sec", float64(totalOperations)/duration.Seconds())
	})

	tt.Run("should handle concurrent database access with prompts", func(t *testing.T) {
		// GIVEN shared database and repositories
		db := setupTestDB(t)
		defer cleanupTestDB(t, db)

		// Mock repositories that would share database connections
		topicRepo := &mocks.MockTopicRepository{}
		promptRepo := &mocks.MockPromptRepository{}

		numGoroutines := 20
		operationsPerGoroutine := 50

		var wg sync.WaitGroup
		errors := make(chan error, numGoroutines)

		// WHEN accessing database concurrently
		start := time.Now()

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				userID := primitive.NewObjectID().Hex()

				for j := 0; j < operationsPerGoroutine; j++ {
					// Create topic with prompt reference
					topic := &entities.Topic{
						ID:         primitive.NewObjectID().Hex(),
						UserID:     userID,
						Name:       fmt.Sprintf("Concurrent Topic %d-%d", id, j),
						PromptName: "base1",
					}

					// This will fail until concurrent database access is properly implemented
					t.Fatal("implement thread-safe database operations with prompts - FAILING IN TDD RED PHASE")

					created, err := topicRepo.Create(ctx, topic)
					if err != nil {
						errors <- err
						return
					}

					// Fetch topic back
					fetched, err := topicRepo.GetByID(ctx, created.ID)
					if err != nil {
						errors <- err
						return
					}

					if fetched.Name != topic.Name {
						errors <- assert.AnError
						return
					}
				}
			}(i)
		}

		wg.Wait()
		close(errors)
		duration := time.Since(start)

		// Check for errors
		errorCount := 0
		for err := range errors {
			t.Logf("Concurrent database error: %v", err)
			errorCount++
		}

		// THEN should handle concurrent database access safely
		assert.Equal(t, 0, errorCount, "Should have no concurrent database errors")
		assert.Less(t, duration, 15*time.Second, "Concurrent database operations should be reasonable")
		
		t.Logf("Concurrent database access completed in %v", duration)
	})
}

// TestMemoryUsageOptimization tests memory efficiency of the prompt system
func TestMemoryUsageOptimization(tt *testing.T) {
	tt.Run("should optimize memory usage for prompt variable caching", func(t *testing.T) {
		// GIVEN many similar templates
		promptProcessor := NewPromptProcessor()

		// Get initial memory
		var m runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m)
		initialMemory := m.Alloc

		templates := make([]string, 1000)
		for i := range templates {
			templates[i] = fmt.Sprintf("Template %d with {name} and {[keywords]}", i)
		}

		// WHEN processing many templates
		processedResults := make([]string, len(templates))

		for i, template := range templates {
			topic := &entities.Topic{
				Name:     fmt.Sprintf("Topic %d", i),
				Keywords: []string{"keyword1", "keyword2", "keyword3"},
			}

			// This will fail until memory-efficient processing is implemented
			t.Fatal("implement memory-efficient prompt processing - FAILING IN TDD RED PHASE")

			result, err := promptProcessor.ProcessTemplate(template, topic, nil)
			require.NoError(tt, err)
			processedResults[i] = result
		}

		// Check memory after processing
		runtime.GC()
		runtime.ReadMemStats(&m)
		finalMemory := m.Alloc
		memoryUsed := finalMemory - initialMemory

		tt.Logf("Memory used for processing %d templates: %d bytes", len(templates), memoryUsed)
		tt.Logf("Average memory per template: %.2f bytes", float64(memoryUsed)/float64(len(templates)))

		// THEN should use memory efficiently
		// Should be under 1KB per template on average
		avgMemoryPerTemplate := float64(memoryUsed) / float64(len(templates))
		assert.Less(tt, avgMemoryPerTemplate, 1024.0, "Should use less than 1KB per template")
	})

	tt.Run("should prevent memory leaks in prompt processing", func(t *testing.T) {
		// GIVEN prompt processor
		promptProcessor := NewPromptProcessor()

		iterations := 10000

		// Get initial memory
		var m runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m)
		initialMemory := m.Alloc

		// WHEN processing many templates in a loop
		for i := 0; i < iterations; i++ {
			template := fmt.Sprintf("Iterative template %d with {name}", i)
			topic := &entities.Topic{Name: fmt.Sprintf("Topic %d", i)}

			// This will fail until leak-free processing is implemented
			t.Fatal("implement memory leak-free prompt processing - FAILING IN TDD RED PHASE")

			result, err := promptProcessor.ProcessTemplate(template, topic, nil)
			require.NoError(tt, err)
			_ = result // Use result to prevent optimization

			// Periodically check memory growth
			if i%1000 == 0 && i > 0 {
				runtime.GC()
				runtime.ReadMemStats(&m)
				currentMemory := m.Alloc
				memoryGrowth := currentMemory - initialMemory
				
				tt.Logf("Iteration %d: Memory growth: %d bytes", i, memoryGrowth)
				
				// Should not grow unbounded
				assert.Less(tt, memoryGrowth, uint64(10*1024*1024), "Memory growth should be bounded")
			}
		}

		// Final memory check
		runtime.GC()
		runtime.ReadMemStats(&m)
		finalMemory := m.Alloc
		totalGrowth := finalMemory - initialMemory

		tt.Logf("Total memory growth after %d iterations: %d bytes", iterations, totalGrowth)
		assert.Less(tt, totalGrowth, uint64(5*1024*1024), "Total memory growth should be reasonable")
	})
}

// Helper functions that will be implemented
func NewPromptProcessor() interface{} {
	// Implementation will go here
	t.Fatal("not implemented - FAILING IN TDD RED PHASE")
	return nil
}

func prepareLLMRequest(prompt string) interface{} {
	// Implementation will go here
	return prompt
}

func setupTestDB(t *testing.T) interface{} {
	// Implementation will go here
	t.Fatal("not implemented - FAILING IN TDD RED PHASE")
	return nil
}

func cleanupTestDB(t *testing.T, db interface{}) {
	// Implementation will go here
}
