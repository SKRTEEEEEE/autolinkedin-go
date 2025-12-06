package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/linkgen-ai/backend/src/domain/errors"
	"go.uber.org/zap"
)

// ErrorCode represents standardized error codes
type ErrorCode string

const (
	ErrorCodeValidation     ErrorCode = "VALIDATION_ERROR"
	ErrorCodeNotFound       ErrorCode = "RESOURCE_NOT_FOUND"
	ErrorCodeUnauthorized   ErrorCode = "UNAUTHORIZED"
	ErrorCodeInternalServer ErrorCode = "INTERNAL_SERVER_ERROR"
	ErrorCodeServiceTimeout ErrorCode = "SERVICE_TIMEOUT"
	ErrorCodeInvalidInput   ErrorCode = "INVALID_INPUT"
	ErrorCodeAlreadyExists  ErrorCode = "RESOURCE_ALREADY_EXISTS"
	ErrorCodeLimitExceeded  ErrorCode = "LIMIT_EXCEEDED"
)

// ErrorResponse represents the standard error response format
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains error details
type ErrorDetail struct {
	Code    ErrorCode              `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// WriteError writes a standardized error response
func WriteError(w http.ResponseWriter, statusCode int, code ErrorCode, message string, details map[string]interface{}, logger *zap.Logger) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
			Details: details,
		},
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		if logger != nil {
			logger.Error("failed to encode error response", zap.Error(err))
		}
	}

	if logger != nil {
		logger.Warn("request error",
			zap.Int("status_code", statusCode),
			zap.String("error_code", string(code)),
			zap.String("message", message),
		)
	}
}

// WriteJSON writes a JSON response
func WriteJSON(w http.ResponseWriter, statusCode int, data interface{}, logger *zap.Logger) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		if logger != nil {
			logger.Error("failed to encode JSON response", zap.Error(err))
		}
		WriteError(w, http.StatusInternalServerError, ErrorCodeInternalServer, "Failed to encode response", nil, logger)
	}
}

// MapDomainError maps domain errors to HTTP status codes and error codes
func MapDomainError(err error, logger *zap.Logger) (statusCode int, code ErrorCode, message string) {
	// Check for specific domain errors
	switch e := err.(type) {
	case *errors.ErrIdeaNotFound:
		return http.StatusNotFound, ErrorCodeNotFound, e.Error()
	case *errors.ErrDraftNotFound:
		return http.StatusNotFound, ErrorCodeNotFound, e.Error()
	case *errors.ErrTopicNotFound:
		return http.StatusNotFound, ErrorCodeNotFound, e.Error()
	case *errors.ErrIdeaExpired:
		return http.StatusGone, ErrorCodeInvalidInput, e.Error()
	case *errors.ErrDraftAlreadyPublished:
		return http.StatusConflict, ErrorCodeAlreadyExists, e.Error()
	case *errors.ErrRefinementLimitExceeded:
		return http.StatusUnprocessableEntity, ErrorCodeLimitExceeded, e.Error()
	case *errors.ErrInvalidDraftType:
		return http.StatusBadRequest, ErrorCodeInvalidInput, e.Error()
	case *errors.ErrInvalidDraftStatus:
		return http.StatusBadRequest, ErrorCodeInvalidInput, e.Error()
	case *errors.ErrUnauthorizedAccess:
		return http.StatusForbidden, ErrorCodeUnauthorized, e.Error()
	case *errors.ErrValidation:
		return http.StatusBadRequest, ErrorCodeValidation, e.Error()
	case *errors.ErrInvalidEmail:
		return http.StatusBadRequest, ErrorCodeValidation, e.Error()
	case *errors.ErrInvalidTransition:
		return http.StatusBadRequest, ErrorCodeInvalidInput, e.Error()
	case *errors.ErrInvalidUserCredentials:
		return http.StatusUnauthorized, ErrorCodeUnauthorized, e.Error()
	default:
		// Default to internal server error
		if logger != nil {
			logger.Error("unmapped domain error", zap.Error(err))
		}
		return http.StatusInternalServerError, ErrorCodeInternalServer, "An unexpected error occurred"
	}
}
