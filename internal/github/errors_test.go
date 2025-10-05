package github

import (
	"errors"
	"net/http"
	"testing"

	"github.com/google/go-github/v75/github"
)

func TestWrapError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		method         string
		url            string
		expectedErr    error
		expectedStatus int
	}{
		{
			name:   "nil error returns nil",
			err:    nil,
			method: "GET",
			url:    "https://api.github.com",
		},
		{
			name: "unauthorized error",
			err: &github.ErrorResponse{
				Response: &http.Response{StatusCode: http.StatusUnauthorized},
				Message:  "Bad credentials",
			},
			method:         "GET",
			url:            "https://api.github.com",
			expectedErr:    ErrUnauthorized,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "not found error",
			err: &github.ErrorResponse{
				Response: &http.Response{StatusCode: http.StatusNotFound},
				Message:  "Not Found",
			},
			method:         "GET",
			url:            "https://api.github.com",
			expectedErr:    ErrNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "rate limit error",
			err: &github.ErrorResponse{
				Response: &http.Response{
					StatusCode: http.StatusForbidden,
					Header:     map[string][]string{"X-Ratelimit-Remaining": {"0"}},
				},
				Message: "API rate limit exceeded",
			},
			method:         "GET",
			url:            "https://api.github.com",
			expectedErr:    ErrRateLimitExceeded,
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "forbidden error",
			err: &github.ErrorResponse{
				Response: &http.Response{StatusCode: http.StatusForbidden},
				Message:  "Forbidden",
			},
			method:         "GET",
			url:            "https://api.github.com",
			expectedErr:    ErrForbidden,
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "server error",
			err: &github.ErrorResponse{
				Response: &http.Response{StatusCode: http.StatusInternalServerError},
				Message:  "Internal Server Error",
			},
			method:         "GET",
			url:            "https://api.github.com",
			expectedErr:    ErrServerError,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "bad request error",
			err: &github.ErrorResponse{
				Response: &http.Response{StatusCode: http.StatusBadRequest},
				Message:  "Bad Request",
			},
			method:         "GET",
			url:            "https://api.github.com",
			expectedErr:    ErrBadRequest,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WrapError(tt.err, tt.method, tt.url)

			if tt.err == nil {
				if result != nil {
					t.Errorf("WrapError() = %v, want nil", result)
				}
				return
			}

			var apiErr *APIError
			if !errors.As(result, &apiErr) {
				t.Errorf("WrapError() did not return an APIError")
				return
			}

			if apiErr.StatusCode != tt.expectedStatus {
				t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, tt.expectedStatus)
			}

			if apiErr.Method != tt.method {
				t.Errorf("Method = %s, want %s", apiErr.Method, tt.method)
			}

			if apiErr.URL != tt.url {
				t.Errorf("URL = %s, want %s", apiErr.URL, tt.url)
			}

			if tt.expectedErr != nil && !errors.Is(result, tt.expectedErr) {
				t.Errorf("Error does not match expected type: got %v, want %v", apiErr.Err, tt.expectedErr)
			}
		})
	}
}

func TestIsRateLimitError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "rate limit error",
			err:  ErrRateLimitExceeded,
			want: true,
		},
		{
			name: "wrapped rate limit error",
			err: &APIError{
				StatusCode: http.StatusForbidden,
				Err:        ErrRateLimitExceeded,
			},
			want: true,
		},
		{
			name: "other error",
			err:  ErrUnauthorized,
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRateLimitError(tt.err); got != tt.want {
				t.Errorf("IsRateLimitError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "rate limit error is retryable",
			err:  ErrRateLimitExceeded,
			want: true,
		},
		{
			name: "server error is retryable",
			err: &APIError{
				StatusCode: http.StatusInternalServerError,
			},
			want: true,
		},
		{
			name: "bad gateway is retryable",
			err: &APIError{
				StatusCode: http.StatusBadGateway,
			},
			want: true,
		},
		{
			name: "service unavailable is retryable",
			err: &APIError{
				StatusCode: http.StatusServiceUnavailable,
			},
			want: true,
		},
		{
			name: "too many requests is retryable",
			err: &APIError{
				StatusCode: http.StatusTooManyRequests,
			},
			want: true,
		},
		{
			name: "unauthorized is not retryable",
			err: &APIError{
				StatusCode: http.StatusUnauthorized,
			},
			want: false,
		},
		{
			name: "not found is not retryable",
			err: &APIError{
				StatusCode: http.StatusNotFound,
			},
			want: false,
		},
		{
			name: "nil error is not retryable",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRetryableError(tt.err); got != tt.want {
				t.Errorf("IsRetryableError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsAuthError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "unauthorized error",
			err:  ErrUnauthorized,
			want: true,
		},
		{
			name: "forbidden error",
			err:  ErrForbidden,
			want: true,
		},
		{
			name: "rate limit error",
			err:  ErrRateLimitExceeded,
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsAuthError(tt.err); got != tt.want {
				t.Errorf("IsAuthError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "not found error",
			err:  ErrNotFound,
			want: true,
		},
		{
			name: "unauthorized error",
			err:  ErrUnauthorized,
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNotFoundError(tt.err); got != tt.want {
				t.Errorf("IsNotFoundError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		name    string
		apiErr  *APIError
		wantMsg string
	}{
		{
			name: "error with wrapped error",
			apiErr: &APIError{
				StatusCode: 401,
				Message:    "Bad credentials",
				URL:        "https://api.github.com",
				Method:     "GET",
				Err:        ErrUnauthorized,
			},
			wantMsg: "github api error: Bad credentials (status: 401, method: GET, url: https://api.github.com): github authentication failed",
		},
		{
			name: "error without wrapped error",
			apiErr: &APIError{
				StatusCode: 404,
				Message:    "Not Found",
				URL:        "https://api.github.com",
				Method:     "GET",
			},
			wantMsg: "github api error: Not Found (status: 404, method: GET, url: https://api.github.com)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.apiErr.Error()
			if got != tt.wantMsg {
				t.Errorf("APIError.Error() = %v, want %v", got, tt.wantMsg)
			}
		})
	}
}

func TestAPIError_Unwrap(t *testing.T) {
	wrappedErr := ErrUnauthorized
	apiErr := &APIError{
		StatusCode: 401,
		Err:        wrappedErr,
	}

	if unwrapped := apiErr.Unwrap(); unwrapped != wrappedErr {
		t.Errorf("APIError.Unwrap() = %v, want %v", unwrapped, wrappedErr)
	}
}
