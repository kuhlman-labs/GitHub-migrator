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

	// ErrSecondaryRateLimitExceeded is returned when GitHub's secondary rate limit is hit
	// Secondary rate limits are triggered by concurrent requests regardless of primary quota
	// See: https://docs.github.com/rest/overview/rate-limits-for-the-rest-api#about-secondary-rate-limits
	ErrSecondaryRateLimitExceeded = errors.New("github secondary rate limit exceeded")

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

	// ErrStreamError is returned when an HTTP/2 stream is cancelled or reset
	// This typically happens when the server terminates the request prematurely
	// (e.g., due to server-side timeout, complexity limits, or transient issues)
	ErrStreamError = errors.New("github stream error")
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

		// Check for secondary rate limit first (403 with specific message)
		if ghErr.Response.StatusCode == http.StatusForbidden && isSecondaryRateLimitMessage(ghErr.Message) {
			apiErr.Err = ErrSecondaryRateLimitExceeded
			return apiErr
		}

		// Map to specific error types based on status code
		apiErr.Err = mapErrorType(ghErr.Response.StatusCode, ghErr.Response.Header)

		return apiErr
	}

	// Try to extract status code from error message for non-JSON responses
	// This handles cases like nginx HTML error pages (502, 503, etc.)
	statusCode := extractStatusCodeFromError(err)
	errMsg := err.Error()

	apiErr := &APIError{
		StatusCode: statusCode,
		Message:    errMsg,
		URL:        url,
		Method:     method,
		Err:        err,
	}

	// Check for secondary rate limit in the error message
	if statusCode == http.StatusForbidden && isSecondaryRateLimitMessage(errMsg) {
		apiErr.Err = ErrSecondaryRateLimitExceeded
		return apiErr
	}

	// Map to specific error types if we have a valid status code
	if statusCode > 0 {
		apiErr.Err = mapErrorType(statusCode, nil)
	}

	return apiErr
}

// isSecondaryRateLimitMessage checks if an error message indicates a secondary rate limit
func isSecondaryRateLimitMessage(message string) bool {
	msgLower := strings.ToLower(message)
	return strings.Contains(msgLower, "secondary rate limit") ||
		strings.Contains(msgLower, "exceeded a secondary rate limit")
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

// IsRateLimitError checks if an error is a rate limit error (primary or secondary)
func IsRateLimitError(err error) bool {
	return errors.Is(err, ErrRateLimitExceeded) || IsSecondaryRateLimitError(err)
}

// IsSecondaryRateLimitError checks if an error is a secondary rate limit error
// Secondary rate limits are triggered by concurrent/burst requests regardless of primary quota
// Detection is based on the error message pattern from GitHub:
// "You have exceeded a secondary rate limit. Please wait a few minutes before you try again."
func IsSecondaryRateLimitError(err error) bool {
	if err == nil {
		return false
	}

	// Check for the sentinel error
	if errors.Is(err, ErrSecondaryRateLimitExceeded) {
		return true
	}

	// Check error message for secondary rate limit patterns
	errMsg := strings.ToLower(err.Error())

	// Patterns that indicate a secondary rate limit
	secondaryRateLimitPatterns := []string{
		"secondary rate limit",
		"exceeded a secondary rate limit",
		"rate-limits-for-the-rest-api#about-secondary-rate-limits",
	}

	for _, pattern := range secondaryRateLimitPatterns {
		if strings.Contains(errMsg, pattern) {
			return true
		}
	}

	return false
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

		// Check for secondary rate limit (403 with specific message)
		if apiErr.StatusCode == http.StatusForbidden && IsSecondaryRateLimitError(err) {
			return true
		}

		// Check for stream errors (status code 0 with stream error patterns)
		if apiErr.StatusCode == 0 && IsStreamError(err) {
			return true
		}
	}

	// Also retry on rate limit errors (primary and secondary)
	if errors.Is(err, ErrRateLimitExceeded) || errors.Is(err, ErrSecondaryRateLimitExceeded) {
		return true
	}

	// Check for secondary rate limit by message pattern
	if IsSecondaryRateLimitError(err) {
		return true
	}

	// Check for stream errors at the top level
	if IsStreamError(err) {
		return true
	}

	return false
}

// IsStreamError checks if an error is an HTTP/2 stream error
// Stream errors occur when the HTTP/2 connection's stream is cancelled or reset,
// typically due to server-side timeouts, complexity limits, or transient network issues.
// These errors have no HTTP status code (status 0) but are often transient and worth retrying.
func IsStreamError(err error) bool {
	if err == nil {
		return false
	}

	// Check for the sentinel error
	if errors.Is(err, ErrStreamError) {
		return true
	}

	errMsg := err.Error()

	// HTTP/2 stream error patterns
	// Example: "stream error: stream ID 3401; CANCEL; received from peer"
	streamErrorPatterns := []string{
		"stream error",            // Generic HTTP/2 stream error
		"CANCEL",                  // Stream was cancelled (RST_STREAM with CANCEL)
		"INTERNAL_ERROR",          // Server internal error on stream
		"REFUSED_STREAM",          // Server refused to process the stream
		"stream ID",               // Mentions stream ID (HTTP/2 specific)
		"RST_STREAM",              // HTTP/2 RST_STREAM frame received
		"GOAWAY",                  // HTTP/2 GOAWAY frame (connection being shut down)
		"received from peer",      // Common suffix for stream errors
		"http2: server sent",      // Go's HTTP/2 client error prefix
		"client disconnected",     // Client-side disconnection
		"connection reset",        // TCP reset
		"connection was forcibly", // Windows TCP reset
		"broken pipe",             // Unix write to closed connection
		"use of closed network",   // Write to closed connection
	}

	for _, pattern := range streamErrorPatterns {
		if strings.Contains(errMsg, pattern) {
			return true
		}
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
