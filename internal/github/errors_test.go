package github

import (
	"errors"
	"net/http"
	"testing"
	"time"

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

func TestIsSecondaryRateLimitError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "secondary rate limit sentinel error",
			err:  ErrSecondaryRateLimitExceeded,
			want: true,
		},
		{
			name: "secondary rate limit message pattern",
			err:  errors.New("You have exceeded a secondary rate limit. Please wait a few minutes before you try again."),
			want: true,
		},
		{
			name: "secondary rate limit doc URL pattern",
			err:  errors.New("rate-limits-for-the-rest-api#about-secondary-rate-limits"),
			want: true,
		},
		{
			name: "wrapped secondary rate limit error",
			err: &APIError{
				StatusCode: http.StatusForbidden,
				Message:    "You have exceeded a secondary rate limit",
				Err:        ErrSecondaryRateLimitExceeded,
			},
			want: true,
		},
		{
			name: "403 forbidden without secondary rate limit",
			err: &APIError{
				StatusCode: http.StatusForbidden,
				Message:    "Resource not accessible by integration",
				Err:        ErrForbidden,
			},
			want: false,
		},
		{
			name: "primary rate limit error",
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
			if got := IsSecondaryRateLimitError(tt.err); got != tt.want {
				t.Errorf("IsSecondaryRateLimitError() = %v, want %v", got, tt.want)
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

func TestExtractStatusCodeFromError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: 0,
		},
		{
			name:     "502 Bad Gateway in error message",
			err:      errors.New("non-200 OK status code: 502 Bad Gateway body: \"<html>...\""),
			expected: http.StatusBadGateway,
		},
		{
			name:     "503 Service Unavailable in error message",
			err:      errors.New("non-200 OK status code: 503 Service Unavailable"),
			expected: http.StatusServiceUnavailable,
		},
		{
			name:     "500 Internal Server Error in error message",
			err:      errors.New("non-200 OK status code: 500 Internal Server Error"),
			expected: http.StatusInternalServerError,
		},
		{
			name:     "504 Gateway Timeout in error message",
			err:      errors.New("non-200 OK status code: 504 Gateway Timeout"),
			expected: http.StatusGatewayTimeout,
		},
		{
			name:     "429 Too Many Requests in error message",
			err:      errors.New("non-200 OK status code: 429 Too Many Requests"),
			expected: http.StatusTooManyRequests,
		},
		{
			name:     "404 Not Found in error message",
			err:      errors.New("non-200 OK status code: 404 Not Found"),
			expected: http.StatusNotFound,
		},
		{
			name:     "403 Forbidden in error message",
			err:      errors.New("non-200 OK status code: 403 Forbidden"),
			expected: http.StatusForbidden,
		},
		{
			name:     "401 Unauthorized in error message",
			err:      errors.New("non-200 OK status code: 401 Unauthorized"),
			expected: http.StatusUnauthorized,
		},
		{
			name:     "400 Bad Request in error message",
			err:      errors.New("non-200 OK status code: 400 Bad Request"),
			expected: http.StatusBadRequest,
		},
		{
			name:     "no status code in error message",
			err:      errors.New("some random error"),
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := extractStatusCodeFromError(tt.err)
			if actual != tt.expected {
				t.Errorf("extractStatusCodeFromError() = %d, want %d", actual, tt.expected)
			}
		})
	}
}

func TestWrapError_HTMLErrorPage(t *testing.T) {
	// Test that HTML error pages (like nginx 502) are properly handled
	htmlErr := errors.New("non-200 OK status code: 502 Bad Gateway body: \"<html>\\r\\n<head><title>502 Bad Gateway</title></head>\\r\\n<body>\\r\\n<center><h1>502 Bad Gateway</h1></center>\\r\\n<hr><center>nginx</center>\\r\\n</body>\\r\\n</html>\\r\\n\"")

	wrapped := WrapError(htmlErr, "GetDependencyGraph", "https://api.github.com")

	var apiErr *APIError
	if !errors.As(wrapped, &apiErr) {
		t.Fatal("Expected APIError type")
	}

	// Should extract 502 from the HTML error
	if apiErr.StatusCode != http.StatusBadGateway {
		t.Errorf("Expected status code %d, got %d", http.StatusBadGateway, apiErr.StatusCode)
	}

	// Should be retryable
	if !IsRetryableError(wrapped) {
		t.Error("502 Bad Gateway should be retryable")
	}

	// Should be a server error
	if !errors.Is(wrapped, ErrServerError) {
		t.Error("502 Bad Gateway should be classified as server error")
	}
}

func TestIsStreamError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "sentinel stream error",
			err:  ErrStreamError,
			want: true,
		},
		{
			name: "HTTP/2 stream cancel from peer",
			err:  errors.New("stream error: stream ID 3401; CANCEL; received from peer"),
			want: true,
		},
		{
			name: "HTTP/2 stream ID mention",
			err:  errors.New("failed with stream ID 123"),
			want: true,
		},
		{
			name: "HTTP/2 RST_STREAM",
			err:  errors.New("received RST_STREAM with error code CANCEL"),
			want: true,
		},
		{
			name: "HTTP/2 GOAWAY",
			err:  errors.New("http2: server sent GOAWAY and closed the connection"),
			want: true,
		},
		{
			name: "HTTP/2 REFUSED_STREAM",
			err:  errors.New("stream error: REFUSED_STREAM"),
			want: true,
		},
		{
			name: "HTTP/2 INTERNAL_ERROR",
			err:  errors.New("stream error: INTERNAL_ERROR"),
			want: true,
		},
		{
			name: "connection reset",
			err:  errors.New("read tcp: connection reset by peer"),
			want: true,
		},
		{
			name: "broken pipe",
			err:  errors.New("write: broken pipe"),
			want: true,
		},
		{
			name: "use of closed connection",
			err:  errors.New("use of closed network connection"),
			want: true,
		},
		{
			name: "http2 server sent error",
			err:  errors.New("http2: server sent GOAWAY and closed the connection; LastStreamID=1, ErrCode=NO_ERROR"),
			want: true,
		},
		{
			name: "regular not found error - not a stream error",
			err:  errors.New("resource not found"),
			want: false,
		},
		{
			name: "regular timeout error - not a stream error",
			err:  errors.New("context deadline exceeded"),
			want: false,
		},
		{
			name: "rate limit error - not a stream error",
			err:  ErrRateLimitExceeded,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsStreamError(tt.err); got != tt.want {
				t.Errorf("IsStreamError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsRetryableError_StreamErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "HTTP/2 stream cancel is retryable",
			err:  errors.New("stream error: stream ID 3401; CANCEL; received from peer"),
			want: true,
		},
		{
			name: "wrapped stream error with status 0 is retryable",
			err: &APIError{
				StatusCode: 0,
				Message:    "stream error: stream ID 3401; CANCEL; received from peer",
				Err:        errors.New("stream error: stream ID 3401; CANCEL; received from peer"),
			},
			want: true,
		},
		{
			name: "connection reset is retryable",
			err:  errors.New("read tcp: connection reset by peer"),
			want: true,
		},
		{
			name: "broken pipe is retryable",
			err:  errors.New("write: broken pipe"),
			want: true,
		},
		{
			name: "GOAWAY is retryable",
			err:  errors.New("http2: server sent GOAWAY"),
			want: true,
		},
		{
			name: "regular 404 error is not retryable",
			err: &APIError{
				StatusCode: http.StatusNotFound,
				Message:    "Not Found",
			},
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

func TestIsRateLimitBlockedError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "rate limit blocked error",
			err:  errors.New("GET https://api.github.com/orgs/test/repos?per_page=100: 403 API rate limit of 5000 still exceeded until 2026-01-06 11:39:25 -0500 EST, not making remote request. [rate reset in 1m56s]"),
			want: true,
		},
		{
			name: "regular error",
			err:  errors.New("something went wrong"),
			want: false,
		},
		{
			name: "rate limit error without blocked pattern",
			err:  ErrRateLimitExceeded,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRateLimitBlockedError(tt.err); got != tt.want {
				t.Errorf("IsRateLimitBlockedError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseRateLimitResetTime(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		wantHasTime    bool
		wantDurationOK bool
	}{
		{
			name:           "nil error",
			err:            nil,
			wantHasTime:    false,
			wantDurationOK: false,
		},
		{
			name:           "error with rate reset in 1m56s",
			err:            errors.New("403 API rate limit exceeded until 2026-01-06. [rate reset in 1m56s]"),
			wantHasTime:    true,
			wantDurationOK: true,
		},
		{
			name:           "error with rate reset in 30s",
			err:            errors.New("Rate limit hit [rate reset in 30s]"),
			wantHasTime:    true,
			wantDurationOK: true,
		},
		{
			name:           "error with rate reset in 2m",
			err:            errors.New("Rate limit [rate reset in 2m]"),
			wantHasTime:    true,
			wantDurationOK: true,
		},
		{
			name:           "error without rate reset time",
			err:            errors.New("Rate limit exceeded"),
			wantHasTime:    false,
			wantDurationOK: false,
		},
		{
			name:           "regular error",
			err:            errors.New("something went wrong"),
			wantHasTime:    false,
			wantDurationOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetTime, hasTime := ParseRateLimitResetTime(tt.err)
			if hasTime != tt.wantHasTime {
				t.Errorf("ParseRateLimitResetTime() hasTime = %v, want %v", hasTime, tt.wantHasTime)
			}
			if tt.wantHasTime && tt.wantDurationOK {
				// Verify the reset time is in the future (within reason)
				now := time.Now()
				if resetTime.Before(now) {
					t.Errorf("ParseRateLimitResetTime() returned past time %v, expected future time", resetTime)
				}
				// Should be within 5 minutes of now (for reasonable durations)
				if resetTime.After(now.Add(5 * time.Minute)) {
					t.Logf("ParseRateLimitResetTime() returned time %v which is more than 5 minutes in the future", resetTime)
				}
			}
		})
	}
}
