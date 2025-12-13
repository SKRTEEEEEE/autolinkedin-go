package usecases

import (
	"context"
	"fmt"
	"runtime"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/linkgen-ai/backend/src/domain/entities"
	"github.com/linkgen-ai/backend/src/test/mocks"
)

// TestVariableSystemMemoryUsage tests memory consumption of the prompt variable system
func TestVariableSystemMemoryUsage(tt *testing.T) {
	tt.Run("should efficiently cache processed templates", func(t *testing.T) {
		// GIVEN a variable processor
		processor := NewVariableProcessor()

		// Create many similar templates that could benefit from caching
		templates := make([]string, 1000)
		for i := range templates {
			// Similar structure with different values
			templates[i] = fmt.Sprintf("Generate {ideas} ideas about {name} with priority %d", i%10)
		}

		// Get baseline memory
		runtime.GC()
		var m1 runtime.MemStats
		runtime.ReadMemStats(&m1)
		initialMemory := m1.Alloc

		// Process templates multiple times to test caching
		iterations := 3
		for round := 0; round < iterations; round++ {
			for i, template := range templates {
				topic := &entities.Topic{
					Name:       fmt.Sprintf("Topic %d", i%50),  // Reuse 50 topics
					IdeasCount: i%5 + 1,
				}

				// This will fail until template caching is implemented
				t.Fatal("implement efficient template variable caching system - FAILING IN TDD RED PHASE")

				_, err := processor.ProcessTemplate(template, topic, nil)
				require.NoError(tt, err)
			}
		}

		// Check memory after processing
		runtime.GC()
		var m2 runtime.MemStats
		runtime.ReadMemStats(&m2)
		finalMemory := m2.Alloc

		memoryUsed := finalMemory - initialMemory
		totalOperations := len(templates) * iterations
		avgMemoryPerOp := float64(memoryUsed) / float64(totalOperations)

		tt.Logf("Memory usage after %d operations: %d bytes", totalOperations, memoryUsed)
		tt.Logf("Average memory per operation: %.2f bytes", avgMemoryPerOp)

		// THEN should use memory efficiently with caching
		assert.Less(tt, avgMemoryPerOp, 100.0, "Should use less than 100 bytes per operation with caching")
		assert.Less(tt, memoryUsed, uint64(5*1024*1024), "Total should be under 5MB with caching")
	})

	tt.Run("should prevent memory leaks with large topic arrays", func(t *testing.T) {
		// GIVEN a variable processor
		processor := NewVariableProcessor()

		// Create templates with array variables that could leak
		template := "Analyze {[keywords]} for priority {priority} topics: {[related_topics]}"

		iterations := 1000

		// Get baseline memory
		runtime.GC()
		var m1 runtime.MemStats
		runtime.ReadMemStats(&m1)
		initialMemory := m1.Alloc

		// Process with large arrays
		for i := 0; i < iterations; i++ {
			// Create progressively larger arrays
			keywordSize := (i % 10) + 5 // 5-15 keywords
			relatedSize := (i % 5) + 3   // 3-8 related topics

			keywords := make([]string, keywordSize)
			relatedTopics := make([]string, relatedSize)

			for j := 0; j < keywordSize; j++ {
				keywords[j] = fmt.Sprintf("keyword%d-%d", i, j)
			}
			for j := 0; j < relatedSize; j++ {
				relatedTopics[j] = fmt.Sprintf("topic%d-%d", i, j)
			}

			topic := &entities.Topic{
				Name:          fmt.Sprintf("Topic %d", i),
				Keywords:      keywords,
				RelatedTopics: relatedTopics,
				Priority:      i % 10 + 1,
			}

			// This will fail until array processing is implemented
			t.Fatal("implement memory-efficient array variable processing - FAILING IN TDD RED PHASE")

			_, err := processor.ProcessTemplate(template, topic, nil)
			require.NoError(tt, err)

			// Check memory growth periodically
			if i%100 == 0 && i > 0 {
				runtime.GC()
				var m runtime.MemStats
				runtime.ReadMemStats(&m)
				memoryGrowth := m.Alloc - initialMemory
				
				tt.Logf("Iteration %d: Memory growth: %d bytes", i, memoryGrowth)
				
				// Should not grow unbounded
				assert.Less(tt, memoryGrowth, uint64(10*1024*1024), 
					"Memory growth should be bounded with large arrays")
			}
		}

		// Final memory check
		runtime.GC()
		var m2 runtime.MemStats
		runtime.ReadMemStats(&m2)
		finalMemory := m2.Alloc

		totalGrowth := finalMemory - initialMemory
		tt.Logf("Final memory growth after %d iterations: %d bytes", iterations, totalGrowth)

		// THEN should handle large arrays without leaks
		assert.Less(tt, totalGrowth, uint64(20*1024*1024), "Total growth should be reasonable with large arrays")
	})

	tt.Run("should optimize memory for frequent variable substitutions", func(t *testing.T) {
		// GIVEN processor with optimization for common patterns
		processor := NewVariableProcessor()

		// Common variables that appear frequently
		commonTemplates := []string{
			"Generate {ideas} ideas about {name}",
			"Content for {name} with {[keywords]}",
			"Priority {priority}: {name} analysis",
		}

		commonTopics := []*entities.Topic{
			{Name: "React", Keywords: []string{"javascript", "ui"}, Priority: 8},
			{Name: "Go", Keywords: []string{"backend", "performance"}, Priority: 9},
			{Name: "Python", Keywords: []string{"ai", "data"}, Priority: 7},
		}

		// Get baseline memory
		runtime.GC()
		var m1 runtime.MemStats
		runtime.ReadMemStats(&m1)
		initialMemory := m1.Alloc

		// Process many combinations of template and topic
		numOperations := 10000
		for i := 0; i < numOperations; i++ {
			template := commonTemplates[i%len(commonTemplates)]
			topic := commonTopics[i%len(commonTopics)]
			topic.IdeasCount = i%5 + 1

			// This will fail until optimization is implemented
			t.Fatal("implement memory optimization for frequent patterns - FAILING IN TDD RED PHASE")

			_, err := processor.ProcessTemplate(template, topic, nil)
			require.NoError(tt, err)
		}

		// Check final memory
		runtime.GC()
		var m2 runtime.MemStats
		runtime.ReadMemStats(&m2)
		finalMemory := m2.Alloc

		memoryUsed := finalMemory - initialMemory
		avgMemoryPerOp := float64(memoryUsed) / float64(numOperations)

		tt.Logf("Memory for %d operations: %d bytes", numOperations, memoryUsed)
		tt.Logf("Average per operation: %.2f bytes", avgMemoryPerOp)

		// THEN should be very efficient for common patterns
		assert.Less(tt, avgMemoryPerOp, 50.0, "Should be very efficient for common patterns")
		assert.Less(tt, memoryUsed, uint64(2*1024*1024), "Total should be minimal for frequent ops")
	})
}

// TestUserContextMemoryEfficiency tests memory usage of user context processing
func TestUserContextMemoryEfficiency(tt *testing.T) {
	tt.Run("should efficiently process user context strings", func(t *testing.T) {
		// GIVEN user context processor
		contextProcessor := NewUserContextProcessor()

		// Create users with varying context sizes
		users := make([]*entities.User, 100)
		for i := range users {
			users[i] = &entities.User{
				ID:   primitive.NewObjectID(),
				Name: fmt.Sprintf("User %d", i),
				Configuration: map[string]interface{}{
					"name":             fmt.Sprintf("User %d", i),
					"expertise":        fmt.Sprintf("Expertise %d", i),
					"tone_preference":  fmt.Sprintf("Tone %d", i%3),
					"industry":         fmt.Sprintf("Industry %d", i%5),
					"experience_years":  i,
					"specializations":   []string{fmt.Sprintf("Spec %d", i), fmt.Sprintf("Spec %d", i+1)},
				},
			}
		}

		// Get baseline memory
		runtime.GC()
		var m1 runtime.MemStats
		runtime.ReadMemStats(&m1)
		initialMemory := m1.Alloc

		// Process user contexts multiple times
		iterations := 5
		for round := 0; round < iterations; round++ {
			for _, user := range users {
				// This will fail until context processing is implemented
				t.Fatal("implement memory-efficient user context processing - FAILING IN TDD RED PHASE")

				contextStr, err := contextProcessor.BuildUserContext(user)
				require.NoError(tt, err)
				require.NotEmpty(tt, contextStr)
			}
		}

		// Check final memory
		runtime.GC()
		var m2 runtime.MemStats
		runtime.ReadMemStats(&m2)
		finalMemory := m2.Alloc

		memoryUsed := finalMemory - initialMemory
		totalOperations := len(users) * iterations
		avgMemoryPerOp := float64(memoryUsed) / float64(totalOperations)

		tt.Logf("Context processing memory: %d bytes for %d operations", memoryUsed, totalOperations)
		tt.Logf("Average per operation: %.2f bytes", avgMemoryPerOp)

		// THEN should be very efficient
		assert.Less(tt, avgMemoryPerOp, 200.0, "Context processing should be memory efficient")
		assert.Less(tt, memoryUsed, uint64(3*1024*1024), "Total should be minimal")
	})

	tt.Run("should handle nested user configuration without memory issues", func(t *testing.T) {
		// GIVEN user with complex nested configuration
		contextProcessor := NewUserContextProcessor()

		// Create complex nested configuration
		complexConfig := map[string]interface{}{
			"profile": map[string]interface{}{
				"name":      "Complex User",
				"title":     "Senior Developer",
				"company":   "Tech Corp",
				"location":  "Remote",
			},
			"preferences": map[string]interface{}{
				"tone":        "Professional",
				"style":       "Technical",
				"language":    "Spanish",
				"format":      "Detailed",
			},
			"expertise": map[string]interface{}{
				"primary":   "Backend Development",
				"secondary": "Cloud Architecture",
				"skills":    []string{"Go", "Kubernetes", "AWS", "Docker"},
				"years": map[string]interface{}{
					"backend":  8,
					"cloud":    5,
					"total":    12,
				},
			},
			"network": map[string]interface{}{
				"connections": 1500,
				"followers":   2500,
				"engagement":  map[string]interface{}{
					"rate":  0.85,
					"reach": 10000,
				},
			},
		}

		user := &entities.User{
			ID:           primitive.NewObjectID(),
			Name:         "Complex User",
			Configuration: complexConfig,
		}

		iterations := 1000

		// Get baseline memory
		runtime.GC()
		var m1 runtime.MemStats
		runtime.ReadMemStats(&m1)
		initialMemory := m1.Alloc

		// Process complex configuration repeatedly
		for i := 0; i < iterations; i++ {
			// This will fail until nested config processing is implemented
			t.Fatal("implement memory-efficient nested config processing - FAILING IN TDD RED PHASE")

			_, err := contextProcessor.BuildUserContext(user)
			require.NoError(tt, err)
		}

		// Check memory usage
		runtime.GC()
		var m2 runtime.MemStats
		runtime.ReadMemStats(&m2)
		finalMemory := m2.Alloc

		memoryUsed := finalMemory - initialMemory
		avgMemoryPerOp := float64(memoryUsed) / float64(iterations)

		tt.Logf("Complex config processing: %d bytes for %d operations", memoryUsed, iterations)
		tt.Logf("Average per operation: %.2f bytes", avgMemoryPerOp)

		// THEN should handle complexity efficiently
		assert.Less(tt, avgMemoryPerOp, 500.0, "Should handle complex configs efficiently")
		assert.Less(tt, memoryUsed, uint64(5*1024*1024), "Should not blow up with complexity")
	})
}

// TestVariableProcessorGarbageCollection tests garbage collection behavior
func TestVariableProcessorGarbageCollection(tt *testing.T) {
	tt.Run("should release memory properly after processing", func(t *testing.T) {
		// GIVEN variable processor
		processor := NewVariableProcessor()

		// Get initial memory
		runtime.GC()
		var mInitial runtime.MemStats
		runtime.ReadMemStats(&mInitial)

		// Process many templates and collect memory stats
		memorySnapshots := []MemorySnapshot{}
		
		for batch := 0; batch < 5; batch++ {
			// Process a batch of templates
			for i := 0; i < 1000; i++ {
				template := fmt.Sprintf("Batch %d - Generate {ideas} ideas about {name}", batch)
				topic := &entities.Topic{
					Name:       fmt.Sprintf("Topic %d-%d", batch, i),
					IdeasCount: i % 5 + 1,
					Keywords:   []string{"keyword1", "keyword2", "keyword3"},
				}

				// This will fail until memory management is implemented
				t.Fatal("implement proper garbage collection in variable processor - FAILING IN TDD RED PHASE")

				_, err := processor.ProcessTemplate(template, topic, nil)
				require.NoError(tt, err)
			}

			// Force garbage collection and capture memory
			runtime.GC()
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			
			memorySnapshots = append(memorySnapshots, MemorySnapshot{
				Batch:   batch,
				Memory:  m.Alloc,
				Time:    time.Now(),
			})

			tt.Logf("After batch %d: Memory = %d bytes", batch, m.Alloc)
		}

		// Analyze memory growth
		maxMemory := uint64(0)
		minMemory := uint64(999999999)
		
		for _, snapshot := range memorySnapshots {
			if snapshot.Memory > maxMemory {
				maxMemory = snapshot.Memory
			}
			if snapshot.Memory < minMemory {
				minMemory = snapshot.Memory
			}
		}

		memoryRange := maxMemory - minMemory
		tt.Logf("Memory range across batches: %d bytes", memoryRange)

		// THEN should release memory properly
		assert.Less(tt, memoryRange, uint64(5*1024*1024), "Memory usage should be stable across batches")
		
		// Memory should not continuously grow
		if len(memorySnapshots) > 3 {
			sorted := make([]MemorySnapshot, len(memorySnapshots))
			copy(sorted, memorySnapshots)
			sort.Slice(sorted, func(i, j int) bool {
				return sorted[i].Memory < sorted[j].Memory
			})
			
			lowerQuartile := sorted[len(sorted)/4]
			upperQuartile := sorted[3*len(sorted)/4]
			growth := upperQuartile.Memory - lowerQuartile.Memory
			
			tt.Logf("Memory growth between quartiles: %d bytes", growth)
			assert.Less(tt, growth, uint64(2*1024*1024), "Memory growth should be limited")
		}
	})
}

// TestConcurrentMemoryAccess tests memory behavior under concurrent access
func TestConcurrentMemoryAccess(tt *testing.T) {
	tt.Run("should handle concurrent access without memory bloat", func(t *testing.T) {
		// GIVEN variable processor
		processor := NewVariableProcessor()

		numGoroutines := 20
		operationsPerGoroutine := 500

		// Get baseline memory
		runtime.GC()
		var mInitial runtime.MemStats
		runtime.ReadMemStats(&mInitial)

		// Run concurrent operations
		var wg sync.WaitGroup
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()

				for j := 0; j < operationsPerGoroutine; j++ {
					template := fmt.Sprintf("Goroutine %d - Template {name}", goroutineID)
					topic := &entities.Topic{
						Name:     fmt.Sprintf("Topic %d-%d", goroutineID, j),
						Keywords: []string{fmt.Sprintf("key %d", j)},
					}

					// This will fail until concurrent memory access is implemented
					t.Fatal("implement thread-safe memory management - FAILING IN TDD RED PHASE")

					_, err := processor.ProcessTemplate(template, topic, nil)
					if err != nil {
						tt.Errorf("Error in goroutine %d, operation %d: %v", goroutineID, j, err)
					}
				}
			}(i)
		}

		wg.Wait()

		// Check final memory
		runtime.GC()
		var mFinal runtime.MemStats
		runtime.ReadMemStats(&mFinal)

		memoryUsed := mFinal.Alloc - mInitial.Alloc
		totalOperations := numGoroutines * operationsPerGoroutine
		avgMemoryPerOp := float64(memoryUsed) / float64(totalOperations)

		tt.Logf("Concurrent memory usage: %d bytes for %d operations", memoryUsed, totalOperations)
		tt.Logf("Average per operation: %.2f bytes", avgMemoryPerOp)

		// THEN should handle concurrency efficiently
		assert.Less(tt, avgMemoryPerOp, 200.0, "Concurrent operations should be memory efficient")
		assert.Less(tt, memoryUsed, uint64(20*1024*1024), "Total concurrent memory usage should be reasonable")
	})
}

// Helper types and functions
type MemorySnapshot struct {
	Batch  int
	Memory uint64
	Time   time.Time
}

// Mock processors that will be implemented
func NewVariableProcessor() interface{} {
	t.Fatal("not implemented - FAILING IN TDD RED PHASE")
	return nil
}

func NewUserContextProcessor() interface{} {
	t.Fatal("not implemented - FAILING IN TDD RED PHASE")
	return nil
}
