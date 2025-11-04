package github

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/go-github/v75/github"
)

var (
	// ErrRateLimitExceeded is returned when the GitHub API rate limit is exceeded
	ErrRateLimitExceeded = errors.New("github rate limit exceeded")

	// ErrUnauthorized is returned when authentication fails
	ErrUnauthorized = errors.New("github authentication failed")

	// ErrNotFound is returned when a resource is not found
	ErrNotFound = errors.New("github resource not found")

	// ErrForbidden is returned when access is forbidden
	ErrForbidden = errors.New("github access forbidden")

	// ErrServerError is returned when GitHub returns a server error
	ErrServerError = errors.New("github server error")

	// ErrBadRequest is returned when the request is malformed
	ErrBadRequest = errors.New("github bad request")

	// ErrTimeout is returned when a request times out
	ErrTimeout = errors.New("github request timeout")
)

// APIError wraps GitHub API errors with additional context
type APIError struct {
	StatusCode int
	Message    string
	URL        string
	Method     string
	Err        error
}

func (e *APIError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("github api error: %s (status: %d, method: %s, url: %s): %v",
			e.Message, e.StatusCode, e.Method, e.URL, e.Err)
	}
	return fmt.Sprintf("github api error: %s (status: %d, method: %s, url: %s)",
		e.Message, e.StatusCode, e.Method, e.URL)
}

func (e *APIError) Unwrap() error {
	return e.Err
}

// WrapError converts a GitHub API error into a structured APIError
func WrapError(err error, method, url string) error {
	if err == nil {
		return nil
	}

	// Check if it's already a GitHub error response
	var ghErr *github.ErrorResponse
	if errors.As(err, &ghErr) {
		apiErr := &APIError{
			StatusCode: ghErr.Response.StatusCode,
			Message:    ghErr.Message,
			URL:        url,
			Method:     method,
			Err:        err,
		}

		// Map to specific error types based on status code
		apiErr.Err = mapErrorType(ghErr.Response.StatusCode, ghErr.Response.Header)

		return apiErr
	}

	// Try to extract status code from error message for non-JSON responses
	// This handles cases like nginx HTML error pages (502, 503, etc.)
	statusCode := extractStatusCodeFromError(err)

	apiErr := &APIError{
		StatusCode: statusCode,
		Message:    err.Error(),
		URL:        url,
		Method:     method,
		Err:        err,
	}

	// Map to specific error types if we have a valid status code
	if statusCode > 0 {
		apiErr.Err = mapErrorType(statusCode, nil)
	}

	return apiErr
}

// mapErrorType maps HTTP status codes to specific error types
func mapErrorType(statusCode int, header http.Header) error {
	switch statusCode {
	case http.StatusUnauthorized:
		return ErrUnauthorized
	case http.StatusForbidden:
		// Check if it's a rate limit error
		if header != nil && header.Get("X-RateLimit-Remaining") == "0" {
			return ErrRateLimitExceeded
		}
		return ErrForbidden
	case http.StatusNotFound:
		return ErrNotFound
	case http.StatusBadRequest:
		return ErrBadRequest
	case http.StatusTooManyRequests:
		return ErrRateLimitExceeded
	case http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return ErrServerError
	default:
		// Return the original status code if no specific mapping
		return nil
	}
}

// extractStatusCodeFromError tries to extract HTTP status code from error message
// This handles cases where GitHub returns HTML error pages instead of JSON
func extractStatusCodeFromError(err error) int {
	if err == nil {
		return 0
	}

	errMsg := err.Error()

	// Map of error patterns to status codes
	// Check in order of specificity (most specific first)
	statusPatterns := map[string]int{
		"500 Internal Server Error": http.StatusInternalServerError,
		"502 Bad Gateway":           http.StatusBadGateway,
		"503 Service Unavailable":   http.StatusServiceUnavailable,
		"504 Gateway Timeout":       http.StatusGatewayTimeout,
		"429 Too Many Requests":     http.StatusTooManyRequests,
		"403 Forbidden":             http.StatusForbidden,
		"401 Unauthorized":          http.StatusUnauthorized,
		"404 Not Found":             http.StatusNotFound,
		"400 Bad Request":           http.StatusBadRequest,
	}

	for pattern, code := range statusPatterns {
		if strings.Contains(errMsg, pattern) {
			return code
		}
	}

	return 0
}

// IsRateLimitError checks if an error is a rate limit error
func IsRateLimitError(err error) bool {
	return errors.Is(err, ErrRateLimitExceeded)
}

// IsRetryableError checks if an error is retryable
func IsRetryableError(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		// Retry on server errors and rate limit errors
		switch apiErr.StatusCode {
		case http.StatusTooManyRequests,
			http.StatusInternalServerError,
			http.StatusBadGateway,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout:
			return true
		}
	}

	// Also retry on rate limit errors
	if errors.Is(err, ErrRateLimitExceeded) {
		return true
	}

	return false
}

// IsAuthError checks if an error is an authentication error
func IsAuthError(err error) bool {
	return errors.Is(err, ErrUnauthorized) || errors.Is(err, ErrForbidden)
}

// IsNotFoundError checks if an error is a not found error
func IsNotFoundError(err error) bool {
	return errors.Is(err, ErrNotFound)
}
