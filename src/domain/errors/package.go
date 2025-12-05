// Package errors defines domain-specific error types for LinkGen AI.
// These errors represent business rule violations and domain-level failures
// that are independent of infrastructure concerns.
//
// Error Categories:
// - ValidationError: Entity validation failures
// - BusinessRuleError: Business rule violations
// - NotFoundError: Entity not found errors
// - ConflictError: Conflict in business operations
package errors

import "fmt"

// ValidationError represents an error in entity validation
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error on field '%s': %s", e.Field, e.Message)
}

// BusinessRuleError represents a violation of business rules
type BusinessRuleError struct {
	Rule    string
	Message string
}

func (e *BusinessRuleError) Error() string {
	return fmt.Sprintf("business rule '%s' violated: %s", e.Rule, e.Message)
}

// NotFoundError represents an entity not found error
type NotFoundError struct {
	Entity string
	ID     string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s with ID '%s' not found", e.Entity, e.ID)
}

// ConflictError represents a conflict in business operations
type ConflictError struct {
	Resource string
	Message  string
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("conflict on resource '%s': %s", e.Resource, e.Message)
}
