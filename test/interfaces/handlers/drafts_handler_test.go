package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestDraftsHandlerGenerateDrafts_Success validates successful POST /v1/drafts/generate requests
// This test will FAIL until DraftsHandler.GenerateDrafts is implemented
func TestDraftsHandlerGenerateDrafts_Success(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		expectJobID    bool
	}{
		{
			name: "successfully queue draft generation with user_id and idea_id",
			requestBody: map[string]interface{}{
				"user_id": "675337baf901e2d790aabbcc",
				"idea_id": "675337baf901e2d790aabbdd",
			},
			expectedStatus: 202,
			expectJobID:    true,
		},
		{
			name: "successfully queue draft generation with only user_id",
			requestBody: map[string]interface{}{
				"user_id": "675337baf901e2d790aabbcc",
			},
			expectedStatus: 202,
			expectJobID:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			bodyBytes, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/v1/drafts/generate", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			_ = req
			_ = w
			_ = req
			_ = w

			// Expected response format:
			// {
			//   "message": "Draft generation started",
			//   "job_id": "..."
			// }

			// Will fail: DraftsHandler.GenerateDrafts doesn't exist yet
			t.Fatal("DraftsHandler.GenerateDrafts not implemented yet - TDD Red phase")
		})
	}
}

// TestDraftsHandlerGenerateDrafts_ValidationErrors validates input validation
// This test will FAIL until input validation is implemented
func TestDraftsHandlerGenerateDrafts_ValidationErrors(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "error on empty request body",
			requestBody:    map[string]interface{}{},
			expectedStatus: 400,
			expectedError:  "user_id is required",
		},
		{
			name: "error on empty user_id",
			requestBody: map[string]interface{}{
				"user_id": "",
			},
			expectedStatus: 400,
			expectedError:  "user_id is required",
		},
		{
			name: "error on invalid user_id format",
			requestBody: map[string]interface{}{
				"user_id": "invalid-objectid",
			},
			expectedStatus: 400,
			expectedError:  "invalid user_id format",
		},
		{
			name: "error on invalid idea_id format",
			requestBody: map[string]interface{}{
				"user_id": "675337baf901e2d790aabbcc",
				"idea_id": "invalid-objectid",
			},
			expectedStatus: 400,
			expectedError:  "invalid idea_id format",
		},
		{
			name:           "error on malformed JSON",
			requestBody:    nil, // Will send malformed JSON
			expectedStatus: 400,
			expectedError:  "invalid JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			var req *http.Request
			if tt.requestBody == nil {
				req = httptest.NewRequest(http.MethodPost, "/v1/drafts/generate", bytes.NewReader([]byte("invalid json")))
			} else {
				bodyBytes, _ := json.Marshal(tt.requestBody)
				req = httptest.NewRequest(http.MethodPost, "/v1/drafts/generate", bytes.NewReader(bodyBytes))
			}
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			_ = req
			_ = w
			_ = req
			_ = w

			// Will fail: Input validation doesn't exist yet
			t.Fatal("DraftsHandler.GenerateDrafts validation not implemented yet - TDD Red phase")
		})
	}
}

// TestDraftsHandlerGenerateDrafts_QueueIntegration validates NATS queue integration
// This test will FAIL until queue integration is implemented
func TestDraftsHandlerGenerateDrafts_QueueIntegration(t *testing.T) {
	t.Run("message published to NATS queue", func(t *testing.T) {
		// Setup
		requestBody := map[string]interface{}{
			"user_id": "675337baf901e2d790aabbcc",
			"idea_id": "675337baf901e2d790aabbdd",
		}
		bodyBytes, _ := json.Marshal(requestBody)
		req := httptest.NewRequest(http.MethodPost, "/v1/drafts/generate", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		_ = req
		_ = w
		_ = req
		_ = w

		// Verify:
		// 1. Handler publishes message to NATS
		// 2. Message contains user_id and idea_id
		// 3. Job ID is generated and returned

		// Will fail: Queue integration doesn't exist yet
		t.Fatal("DraftsHandler.GenerateDrafts queue integration not implemented yet - TDD Red phase")
	})
}

// TestDraftsHandlerListDrafts_Success validates successful GET /v1/drafts/{userId} requests
// This test will FAIL until DraftsHandler.ListDrafts is implemented
func TestDraftsHandlerListDrafts_Success(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		queryParams    map[string]string
		expectedStatus int
		expectedCount  int
	}{
		{
			name:           "get all drafts for user",
			userID:         "675337baf901e2d790aabbcc",
			queryParams:    map[string]string{},
			expectedStatus: 200,
			expectedCount:  5,
		},
		{
			name:   "filter drafts by status",
			userID: "675337baf901e2d790aabbcc",
			queryParams: map[string]string{
				"status": "draft",
			},
			expectedStatus: 200,
			expectedCount:  3,
		},
		{
			name:   "filter drafts by type",
			userID: "675337baf901e2d790aabbcc",
			queryParams: map[string]string{
				"type": "post",
			},
			expectedStatus: 200,
			expectedCount:  4,
		},
		{
			name:   "filter drafts by status and type",
			userID: "675337baf901e2d790aabbcc",
			queryParams: map[string]string{
				"status": "published",
				"type":   "article",
			},
			expectedStatus: 200,
			expectedCount:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			req := httptest.NewRequest(http.MethodGet, "/v1/drafts/"+tt.userID, nil)
			q := req.URL.Query()
			for k, v := range tt.queryParams {
				q.Add(k, v)
			}
			req.URL.RawQuery = q.Encode()
			w := httptest.NewRecorder()

			_ = req

			_ = w

			// Expected response format:
			// {
			//   "drafts": [
			//     {
			//       "id": "...",
			//       "user_id": "...",
			//       "type": "post|article",
			//       "status": "draft|published",
			//       "content": "...",
			//       "created_at": "...",
			//       "updated_at": "..."
			//     }
			//   ]
			// }

			// Will fail: DraftsHandler.ListDrafts doesn't exist yet
			t.Fatal("DraftsHandler.ListDrafts not implemented yet - TDD Red phase")
		})
	}
}

// TestDraftsHandlerListDrafts_ValidationErrors validates input validation
// This test will FAIL until input validation is implemented
func TestDraftsHandlerListDrafts_ValidationErrors(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		queryParams    map[string]string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "error on empty user ID",
			userID:         "",
			queryParams:    map[string]string{},
			expectedStatus: 400,
			expectedError:  "user_id is required",
		},
		{
			name:           "error on invalid user ID format",
			userID:         "invalid-objectid",
			queryParams:    map[string]string{},
			expectedStatus: 400,
			expectedError:  "invalid user_id format",
		},
		{
			name:   "error on invalid status value",
			userID: "675337baf901e2d790aabbcc",
			queryParams: map[string]string{
				"status": "invalid-status",
			},
			expectedStatus: 400,
			expectedError:  "invalid status value",
		},
		{
			name:   "error on invalid type value",
			userID: "675337baf901e2d790aabbcc",
			queryParams: map[string]string{
				"type": "invalid-type",
			},
			expectedStatus: 400,
			expectedError:  "invalid type value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			req := httptest.NewRequest(http.MethodGet, "/v1/drafts/"+tt.userID, nil)
			q := req.URL.Query()
			for k, v := range tt.queryParams {
				q.Add(k, v)
			}
			req.URL.RawQuery = q.Encode()
			w := httptest.NewRecorder()

			_ = req

			_ = w

			// Will fail: Input validation doesn't exist yet
			t.Fatal("DraftsHandler.ListDrafts validation not implemented yet - TDD Red phase")
		})
	}
}

// TestDraftsHandlerRefineDraft_Success validates successful POST /v1/drafts/{draftId}/refine requests
// This test will FAIL until DraftsHandler.RefineDraft is implemented
func TestDraftsHandlerRefineDraft_Success(t *testing.T) {
	tests := []struct {
		name           string
		draftID        string
		requestBody    map[string]interface{}
		expectedStatus int
	}{
		{
			name:    "successfully refine draft with prompt",
			draftID: "675337baf901e2d790aabbee",
			requestBody: map[string]interface{}{
				"prompt": "Make it more engaging and add emojis",
			},
			expectedStatus: 200,
		},
		{
			name:    "successfully refine draft with long prompt",
			draftID: "675337baf901e2d790aabbee",
			requestBody: map[string]interface{}{
				"prompt": "Rewrite this draft to make it more professional, add technical details, and include relevant statistics to support the main argument",
			},
			expectedStatus: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			bodyBytes, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/v1/drafts/"+tt.draftID+"/refine", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			_ = req

			_ = w

			// Expected response format:
			// {
			//   "draft": {
			//     "id": "...",
			//     "user_id": "...",
			//     "type": "post|article",
			//     "status": "draft",
			//     "content": "...",
			//     "history": [...],
			//     "updated_at": "..."
			//   }
			// }

			// Will fail: DraftsHandler.RefineDraft doesn't exist yet
			t.Fatal("DraftsHandler.RefineDraft not implemented yet - TDD Red phase")
		})
	}
}

// TestDraftsHandlerRefineDraft_ValidationErrors validates input validation
// This test will FAIL until input validation is implemented
func TestDraftsHandlerRefineDraft_ValidationErrors(t *testing.T) {
	tests := []struct {
		name           string
		draftID        string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name:    "error on empty draft ID",
			draftID: "",
			requestBody: map[string]interface{}{
				"prompt": "Make it better",
			},
			expectedStatus: 400,
			expectedError:  "draft_id is required",
		},
		{
			name:    "error on invalid draft ID format",
			draftID: "invalid-objectid",
			requestBody: map[string]interface{}{
				"prompt": "Make it better",
			},
			expectedStatus: 400,
			expectedError:  "invalid draft_id format",
		},
		{
			name:    "error on empty prompt",
			draftID: "675337baf901e2d790aabbee",
			requestBody: map[string]interface{}{
				"prompt": "",
			},
			expectedStatus: 400,
			expectedError:  "prompt is required",
		},
		{
			name:    "error on short prompt",
			draftID: "675337baf901e2d790aabbee",
			requestBody: map[string]interface{}{
				"prompt": "short",
			},
			expectedStatus: 400,
			expectedError:  "prompt must be at least 10 characters",
		},
		{
			name:    "error on excessive prompt length",
			draftID: "675337baf901e2d790aabbee",
			requestBody: map[string]interface{}{
				"prompt": string(make([]byte, 501)), // 501 characters
			},
			expectedStatus: 400,
			expectedError:  "prompt exceeds maximum of 500 characters",
		},
		{
			name:           "error on malformed JSON",
			draftID:        "675337baf901e2d790aabbee",
			requestBody:    nil, // Will send malformed JSON
			expectedStatus: 400,
			expectedError:  "invalid JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			var req *http.Request
			if tt.requestBody == nil {
				req = httptest.NewRequest(http.MethodPost, "/v1/drafts/"+tt.draftID+"/refine", bytes.NewReader([]byte("invalid json")))
			} else {
				bodyBytes, _ := json.Marshal(tt.requestBody)
				req = httptest.NewRequest(http.MethodPost, "/v1/drafts/"+tt.draftID+"/refine", bytes.NewReader(bodyBytes))
			}
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			_ = req

			_ = w

			// Will fail: Input validation doesn't exist yet
			t.Fatal("DraftsHandler.RefineDraft validation not implemented yet - TDD Red phase")
		})
	}
}

// TestDraftsHandlerRefineDraft_NotFound validates 404 responses
// This test will FAIL until error handling is implemented
func TestDraftsHandlerRefineDraft_NotFound(t *testing.T) {
	tests := []struct {
		name           string
		draftID        string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "return 404 when draft not found",
			draftID:        "675337baf901e2d790aaaaaa",
			expectedStatus: 404,
			expectedError:  "draft not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			requestBody := map[string]interface{}{
				"prompt": "Make it better",
			}
			bodyBytes, _ := json.Marshal(requestBody)
			req := httptest.NewRequest(http.MethodPost, "/v1/drafts/"+tt.draftID+"/refine", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			_ = req

			_ = w

			// Will fail: 404 handling doesn't exist yet
			t.Fatal("DraftsHandler.RefineDraft 404 handling not implemented yet - TDD Red phase")
		})
	}
}

// TestDraftsHandlerRefineDraft_UseCaseIntegration validates use case integration
// This test will FAIL until use case integration is implemented
func TestDraftsHandlerRefineDraft_UseCaseIntegration(t *testing.T) {
	t.Run("calls RefineDraftUseCase correctly", func(t *testing.T) {
		// Setup
		requestBody := map[string]interface{}{
			"prompt": "Make it more professional",
		}
		bodyBytes, _ := json.Marshal(requestBody)
		req := httptest.NewRequest(http.MethodPost, "/v1/drafts/675337baf901e2d790aabbee/refine", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		_ = req

		_ = w

		// Verify:
		// 1. Handler extracts draftID from URL path
		// 2. Handler parses request body and extracts prompt
		// 3. Handler calls RefineDraftUseCase.Execute with draftID and prompt
		// 4. Handler formats refined draft in response

		// Will fail: Use case integration doesn't exist yet
		t.Fatal("DraftsHandler.RefineDraft use case integration not implemented yet - TDD Red phase")
	})
}

// TestDraftsHandlerRefineDraft_LLMTimeout validates LLM timeout handling
// This test will FAIL until timeout handling is implemented
func TestDraftsHandlerRefineDraft_LLMTimeout(t *testing.T) {
	t.Run("return 503 when LLM times out", func(t *testing.T) {
		// Setup
		requestBody := map[string]interface{}{
			"prompt": "Make it better",
		}
		bodyBytes, _ := json.Marshal(requestBody)
		req := httptest.NewRequest(http.MethodPost, "/v1/drafts/675337baf901e2d790aabbee/refine", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		_ = req

		_ = w

		// Will fail: LLM timeout handling doesn't exist yet
		t.Fatal("DraftsHandler.RefineDraft timeout handling not implemented yet - TDD Red phase")
	})
}

// TestDraftsHandler_MethodNotAllowed validates HTTP method restrictions
// This test will FAIL until method validation is implemented
func TestDraftsHandler_MethodNotAllowed(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{
			name:           "GET not allowed on /v1/drafts/generate",
			method:         http.MethodGet,
			path:           "/v1/drafts/generate",
			expectedStatus: 405,
		},
		{
			name:           "DELETE not allowed on /v1/drafts/{draftId}/refine",
			method:         http.MethodDelete,
			path:           "/v1/drafts/675337baf901e2d790aabbee/refine",
			expectedStatus: 405,
		},
		{
			name:           "POST not allowed on /v1/drafts/{userId}",
			method:         http.MethodPost,
			path:           "/v1/drafts/675337baf901e2d790aabbcc",
			expectedStatus: 405,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			_ = req

			_ = w

			_ = req

			_ = w

			// Will fail: Method validation doesn't exist yet
			t.Fatal("DraftsHandler method validation not implemented yet - TDD Red phase")
		})
	}
}

// TestDraftsHandler_ContentType validates Content-Type header handling
// This test will FAIL until content-type handling is implemented
func TestDraftsHandler_ContentType(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		path        string
		contentType string
		shouldFail  bool
	}{
		{
			name:        "accept application/json for POST requests",
			method:      http.MethodPost,
			path:        "/v1/drafts/generate",
			contentType: "application/json",
			shouldFail:  false,
		},
		{
			name:        "reject non-JSON content type for POST requests",
			method:      http.MethodPost,
			path:        "/v1/drafts/generate",
			contentType: "text/plain",
			shouldFail:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			requestBody := map[string]interface{}{
				"user_id": "675337baf901e2d790aabbcc",
			}
			bodyBytes, _ := json.Marshal(requestBody)
			req := httptest.NewRequest(tt.method, tt.path, bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", tt.contentType)
			w := httptest.NewRecorder()

			_ = req

			_ = w

			// Will fail: Content-Type validation doesn't exist yet
			t.Fatal("DraftsHandler Content-Type validation not implemented yet - TDD Red phase")
		})
	}
}

// TestDraftsHandler_ContextHandling validates context propagation
// This test will FAIL until context handling is implemented
func TestDraftsHandler_ContextHandling(t *testing.T) {
	t.Run("context cancelled during refine operation", func(t *testing.T) {
		// Setup
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		requestBody := map[string]interface{}{
			"prompt": "Make it better",
		}
		bodyBytes, _ := json.Marshal(requestBody)
		req := httptest.NewRequest(http.MethodPost, "/v1/drafts/675337baf901e2d790aabbee/refine", bytes.NewReader(bodyBytes))
		req = req.WithContext(ctx)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		_ = req

		_ = w

		// Will fail: Context handling doesn't exist yet
		t.Fatal("DraftsHandler context handling not implemented yet - TDD Red phase")
	})
}

// TestDraftsHandler_ErrorResponseFormat validates consistent error response structure
// This test will FAIL until error response formatting is implemented
func TestDraftsHandler_ErrorResponseFormat(t *testing.T) {
	tests := []struct {
		name          string
		draftID       string
		requestBody   map[string]interface{}
		expectedError map[string]interface{}
	}{
		{
			name:    "validation error has consistent format",
			draftID: "invalid-id",
			requestBody: map[string]interface{}{
				"prompt": "Make it better",
			},
			expectedError: map[string]interface{}{
				"error": map[string]interface{}{
					"code":    "INVALID_INPUT",
					"message": "invalid draft_id format",
					"details": map[string]interface{}{},
				},
			},
		},
		{
			name:    "not found error has consistent format",
			draftID: "675337baf901e2d790aaaaaa",
			requestBody: map[string]interface{}{
				"prompt": "Make it better",
			},
			expectedError: map[string]interface{}{
				"error": map[string]interface{}{
					"code":    "RESOURCE_NOT_FOUND",
					"message": "draft not found",
					"details": map[string]interface{}{},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			bodyBytes, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/v1/drafts/"+tt.draftID+"/refine", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			_ = req

			_ = w

			// Will fail: Error response formatting doesn't exist yet
			t.Fatal("DraftsHandler error response formatting not implemented yet - TDD Red phase")
		})
	}
}

// TestDraftsHandler_EndToEnd validates complete request-response cycles
// This test will FAIL until full handler implementation is complete
func TestDraftsHandler_EndToEnd(t *testing.T) {
	t.Run("complete generate drafts workflow", func(t *testing.T) {
		// Steps:
		// 1. Receive HTTP POST request with JSON body
		// 2. Parse and validate request body (user_id, idea_id)
		// 3. Publish message to NATS queue
		// 4. Generate job ID
		// 5. Return 202 Accepted with job ID

		requestBody := map[string]interface{}{
			"user_id": "675337baf901e2d790aabbcc",
			"idea_id": "675337baf901e2d790aabbdd",
		}
		bodyBytes, _ := json.Marshal(requestBody)
		req := httptest.NewRequest(http.MethodPost, "/v1/drafts/generate", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		_ = req

		_ = w

		// Will fail: Full workflow doesn't exist yet
		t.Fatal("DraftsHandler.GenerateDrafts workflow not implemented yet - TDD Red phase")
	})

	t.Run("complete list drafts workflow", func(t *testing.T) {
		// Steps:
		// 1. Receive HTTP GET request with userID in path
		// 2. Parse query parameters (status, type)
		// 3. Validate all inputs
		// 4. Call repository.ListByUserID with filters
		// 5. Format response as JSON
		// 6. Return 200 with drafts array

		req := httptest.NewRequest(http.MethodGet, "/v1/drafts/675337baf901e2d790aabbcc?status=draft&type=post", nil)
		w := httptest.NewRecorder()

		_ = req

		_ = w

		_ = req

		_ = w

		// Will fail: Full workflow doesn't exist yet
		t.Fatal("DraftsHandler.ListDrafts workflow not implemented yet - TDD Red phase")
	})

	t.Run("complete refine draft workflow", func(t *testing.T) {
		// Steps:
		// 1. Receive HTTP POST request with draftID in path
		// 2. Parse and validate request body (prompt)
		// 3. Call RefineDraftUseCase
		// 4. Format refined draft in response
		// 5. Return 200 with updated draft

		requestBody := map[string]interface{}{
			"prompt": "Make it more engaging and professional",
		}
		bodyBytes, _ := json.Marshal(requestBody)
		req := httptest.NewRequest(http.MethodPost, "/v1/drafts/675337baf901e2d790aabbee/refine", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		_ = req

		_ = w

		// Will fail: Full workflow doesn't exist yet
		t.Fatal("DraftsHandler.RefineDraft workflow not implemented yet - TDD Red phase")
	})
}

// Helper functions for tests (will be used once implementation exists)

func parseDraftResponse(t *testing.T, body string) map[string]interface{} {
	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("Failed to parse draft response: %v", err)
	}
	return resp
}

func parseDraftsListResponse(t *testing.T, body string) map[string]interface{} {
	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("Failed to parse drafts list response: %v", err)
	}
	return resp
}

func parseGenerateResponse(t *testing.T, body string) map[string]interface{} {
	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("Failed to parse generate response: %v", err)
	}
	return resp
}
