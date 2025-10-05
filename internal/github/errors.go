package github

import (
	"errors"
	"fmt"
	"net/http"

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

		// Map to specific error types
		switch ghErr.Response.StatusCode {
		case http.StatusUnauthorized:
			apiErr.Err = ErrUnauthorized
		case http.StatusForbidden:
			// Check if it's a rate limit error
			if ghErr.Response.Header.Get("X-RateLimit-Remaining") == "0" {
				apiErr.Err = ErrRateLimitExceeded
			} else {
				apiErr.Err = ErrForbidden
			}
		case http.StatusNotFound:
			apiErr.Err = ErrNotFound
		case http.StatusBadRequest:
			apiErr.Err = ErrBadRequest
		case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable:
			apiErr.Err = ErrServerError
		}

		return apiErr
	}

	// Wrap as generic API error
	return &APIError{
		StatusCode: 0,
		Message:    err.Error(),
		URL:        url,
		Method:     method,
		Err:        err,
	}
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
