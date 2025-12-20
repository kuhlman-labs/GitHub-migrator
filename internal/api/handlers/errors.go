package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// APIError represents a standardized API error response.
// It includes an HTTP status code and a user-facing message.
type APIError struct {
	Code    int    `json:"-"`                 // HTTP status code
	Message string `json:"error"`             // User-facing error message
	Details string `json:"details,omitempty"` // Optional additional details
	Field   string `json:"field,omitempty"`   // Optional field name for validation errors
}

// Error implements the error interface.
func (e APIError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s", e.Message, e.Details)
	}
	return e.Message
}

// StatusCode returns the HTTP status code for this error.
func (e APIError) StatusCode() int {
	if e.Code == 0 {
		return http.StatusInternalServerError
	}
	return e.Code
}

// WithDetails returns a copy of the error with additional details.
func (e APIError) WithDetails(details string) APIError {
	return APIError{
		Code:    e.Code,
		Message: e.Message,
		Details: details,
		Field:   e.Field,
	}
}

// WithField returns a copy of the error with a field name.
func (e APIError) WithField(field string) APIError {
	return APIError{
		Code:    e.Code,
		Message: e.Message,
		Details: e.Details,
		Field:   field,
	}
}

// Common API errors - use these for consistent error responses
var (
	// 400 Bad Request errors
	ErrBadRequest = APIError{
		Code:    http.StatusBadRequest,
		Message: "Bad request",
	}
	ErrInvalidJSON = APIError{
		Code:    http.StatusBadRequest,
		Message: "Invalid JSON in request body",
	}
	ErrMissingField = APIError{
		Code:    http.StatusBadRequest,
		Message: "Required field is missing",
	}
	ErrInvalidField = APIError{
		Code:    http.StatusBadRequest,
		Message: "Invalid field value",
	}
	ErrInvalidID = APIError{
		Code:    http.StatusBadRequest,
		Message: "Invalid ID format",
	}

	// 401 Unauthorized errors
	ErrUnauthorized = APIError{
		Code:    http.StatusUnauthorized,
		Message: "Authentication required",
	}
	ErrInvalidToken = APIError{
		Code:    http.StatusUnauthorized,
		Message: "Invalid or expired token",
	}

	// 403 Forbidden errors
	ErrForbidden = APIError{
		Code:    http.StatusForbidden,
		Message: "Access denied",
	}
	ErrInsufficientPermissions = APIError{
		Code:    http.StatusForbidden,
		Message: "Insufficient permissions for this operation",
	}

	// 404 Not Found errors
	ErrNotFound = APIError{
		Code:    http.StatusNotFound,
		Message: "Resource not found",
	}
	ErrRepositoryNotFound = APIError{
		Code:    http.StatusNotFound,
		Message: "Repository not found",
	}
	ErrBatchNotFound = APIError{
		Code:    http.StatusNotFound,
		Message: "Batch not found",
	}
	ErrUserNotFound = APIError{
		Code:    http.StatusNotFound,
		Message: "User not found",
	}
	ErrTeamNotFound = APIError{
		Code:    http.StatusNotFound,
		Message: "Team not found",
	}

	// 409 Conflict errors
	ErrConflict = APIError{
		Code:    http.StatusConflict,
		Message: "Resource already exists",
	}
	ErrBatchAlreadyStarted = APIError{
		Code:    http.StatusConflict,
		Message: "Batch has already started and cannot be modified",
	}
	ErrRepositoryLocked = APIError{
		Code:    http.StatusConflict,
		Message: "Repository is locked for migration",
	}

	// 422 Unprocessable Entity errors
	ErrUnprocessable = APIError{
		Code:    http.StatusUnprocessableEntity,
		Message: "Request cannot be processed",
	}
	ErrValidationFailed = APIError{
		Code:    http.StatusUnprocessableEntity,
		Message: "Validation failed",
	}

	// 500 Internal Server Error
	ErrInternal = APIError{
		Code:    http.StatusInternalServerError,
		Message: "An internal error occurred",
	}
	ErrDatabaseFetch = APIError{
		Code:    http.StatusInternalServerError,
		Message: "Failed to retrieve data",
	}
	ErrDatabaseSave = APIError{
		Code:    http.StatusInternalServerError,
		Message: "Failed to save data",
	}
	ErrDatabaseUpdate = APIError{
		Code:    http.StatusInternalServerError,
		Message: "Failed to update data",
	}
	ErrDatabaseDelete = APIError{
		Code:    http.StatusInternalServerError,
		Message: "Failed to delete data",
	}

	// 503 Service Unavailable
	ErrServiceUnavailable = APIError{
		Code:    http.StatusServiceUnavailable,
		Message: "Service temporarily unavailable",
	}
	ErrClientNotConfigured = APIError{
		Code:    http.StatusServiceUnavailable,
		Message: "Source or destination client not configured",
	}
	ErrDiscoveryInProgress = APIError{
		Code:    http.StatusServiceUnavailable,
		Message: "Discovery is already in progress",
	}
)

// WriteError writes an APIError to the response writer.
func WriteError(w http.ResponseWriter, err APIError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.StatusCode())
	_ = json.NewEncoder(w).Encode(err)
}

// WriteErrorFromErr writes an error to the response writer.
// If the error is an APIError, it uses its status code.
// Otherwise, it returns a 500 Internal Server Error.
func WriteErrorFromErr(w http.ResponseWriter, err error) {
	var apiErr APIError
	if errors.As(err, &apiErr) {
		WriteError(w, apiErr)
		return
	}
	WriteError(w, ErrInternal.WithDetails(err.Error()))
}

// NewValidationError creates a validation error for a specific field.
func NewValidationError(field, message string) APIError {
	return APIError{
		Code:    http.StatusUnprocessableEntity,
		Message: message,
		Field:   field,
	}
}

// NewNotFoundError creates a not found error for a specific resource.
func NewNotFoundError(resource, identifier string) APIError {
	return APIError{
		Code:    http.StatusNotFound,
		Message: fmt.Sprintf("%s not found: %s", resource, identifier),
	}
}

// NewConflictError creates a conflict error with a specific message.
func NewConflictError(message string) APIError {
	return APIError{
		Code:    http.StatusConflict,
		Message: message,
	}
}
