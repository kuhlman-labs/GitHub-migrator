package handlers

import (
	"net/http"
	"net/url"
	"testing"
)

func TestParseCommaSeparatedList(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "single value",
			input:    "value1",
			expected: []string{"value1"},
		},
		{
			name:     "multiple values",
			input:    "value1,value2,value3",
			expected: []string{"value1", "value2", "value3"},
		},
		{
			name:     "values with spaces",
			input:    "value1, value2 , value3",
			expected: []string{"value1", "value2", "value3"},
		},
		{
			name:     "empty values filtered",
			input:    "value1,,value2",
			expected: []string{"value1", "value2"},
		},
		{
			name:     "only commas",
			input:    ",,,",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCommaSeparatedList(tt.input)
			if !strSliceEqual(result, tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestParseBoolPtr(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *bool
	}{
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "true",
			input:    "true",
			expected: boolPtr(true),
		},
		{
			name:     "false",
			input:    "false",
			expected: boolPtr(false),
		},
		{
			name:     "other value",
			input:    "yes",
			expected: boolPtr(false), // Not "true" so false
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseBoolPtr(tt.input)
			if tt.expected == nil {
				if result != nil {
					t.Errorf("Expected nil, got %v", *result)
				}
			} else {
				if result == nil {
					t.Errorf("Expected %v, got nil", *tt.expected)
				} else if *result != *tt.expected {
					t.Errorf("Expected %v, got %v", *tt.expected, *result)
				}
			}
		})
	}
}

func TestParseInt64Ptr(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *int64
	}{
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "valid number",
			input:    "123",
			expected: int64Ptr(123),
		},
		{
			name:     "negative number",
			input:    "-456",
			expected: int64Ptr(-456),
		},
		{
			name:     "invalid number",
			input:    "abc",
			expected: nil,
		},
		{
			name:     "zero",
			input:    "0",
			expected: int64Ptr(0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseInt64Ptr(tt.input)
			if tt.expected == nil {
				if result != nil {
					t.Errorf("Expected nil, got %v", *result)
				}
			} else {
				if result == nil {
					t.Errorf("Expected %v, got nil", *tt.expected)
				} else if *result != *tt.expected {
					t.Errorf("Expected %v, got %v", *tt.expected, *result)
				}
			}
		})
	}
}

func TestParseIntPtr(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *int
	}{
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "valid number",
			input:    "42",
			expected: intPtr(42),
		},
		{
			name:     "invalid number",
			input:    "not-a-number",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseIntPtr(tt.input)
			if tt.expected == nil {
				if result != nil {
					t.Errorf("Expected nil, got %v", *result)
				}
			} else {
				if result == nil {
					t.Errorf("Expected %v, got nil", *tt.expected)
				} else if *result != *tt.expected {
					t.Errorf("Expected %v, got %v", *tt.expected, *result)
				}
			}
		})
	}
}

func TestRepositoryFilters_ToMap(t *testing.T) {
	limit := 10
	offset := 20
	batchID := int64(5)
	hasLFS := true

	filters := &RepositoryFilters{
		Status:            []string{"pending", "complete"},
		BatchID:           &batchID,
		Source:            testSourceGitHub,
		Organization:      []string{"org1"},
		Search:            "test",
		Visibility:        "public",
		HasLFS:            &hasLFS,
		SortBy:            "name",
		AvailableForBatch: true,
		Limit:             &limit,
		Offset:            &offset,
	}

	m := filters.ToMap()

	// Check status
	if statuses, ok := m["status"].([]string); ok {
		if len(statuses) != 2 {
			t.Errorf("Expected 2 statuses, got %d", len(statuses))
		}
	} else {
		t.Error("Expected status to be []string")
	}

	// Check batch_id
	if bID, ok := m["batch_id"].(int64); ok {
		if bID != 5 {
			t.Errorf("Expected batch_id 5, got %d", bID)
		}
	} else {
		t.Error("Expected batch_id to be int64")
	}

	// Check source
	if source, ok := m["source"].(string); ok {
		if source != testSourceGitHub {
			t.Errorf("Expected source 'github', got %s", source)
		}
	} else {
		t.Error("Expected source to be string")
	}

	// Check has_lfs
	if lfs, ok := m["has_lfs"].(bool); ok {
		if !lfs {
			t.Error("Expected has_lfs to be true")
		}
	} else {
		t.Error("Expected has_lfs to be bool")
	}

	// Check pagination
	if l, ok := m["limit"].(int); ok {
		if l != 10 {
			t.Errorf("Expected limit 10, got %d", l)
		}
	} else {
		t.Error("Expected limit to be int")
	}
}

func TestRepositoryFilters_HasPagination(t *testing.T) {
	tests := []struct {
		name     string
		limit    *int
		expected bool
	}{
		{
			name:     "nil limit",
			limit:    nil,
			expected: false,
		},
		{
			name:     "zero limit",
			limit:    intPtr(0),
			expected: false,
		},
		{
			name:     "positive limit",
			limit:    intPtr(10),
			expected: true,
		},
		{
			name:     "negative limit",
			limit:    intPtr(-1),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filters := &RepositoryFilters{Limit: tt.limit}
			result := filters.HasPagination()
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestParseRepositoryFilters(t *testing.T) {
	// Create a mock HTTP request with query parameters
	reqURL, _ := url.Parse("http://example.com/repos?status=pending,complete&batch_id=123&source=github&has_lfs=true&limit=50&offset=10")
	req := &http.Request{URL: reqURL}

	filters := ParseRepositoryFilters(req)

	// Check status
	if len(filters.Status) != 2 {
		t.Errorf("Expected 2 statuses, got %d", len(filters.Status))
	}

	// Check batch_id
	if filters.BatchID == nil || *filters.BatchID != 123 {
		t.Error("Expected batch_id 123")
	}

	// Check source
	if filters.Source != testSourceGitHub {
		t.Errorf("Expected source 'github', got %s", filters.Source)
	}

	// Check has_lfs
	if filters.HasLFS == nil || !*filters.HasLFS {
		t.Error("Expected has_lfs to be true")
	}

	// Check limit
	if filters.Limit == nil || *filters.Limit != 50 {
		t.Error("Expected limit 50")
	}

	// Check offset
	if filters.Offset == nil || *filters.Offset != 10 {
		t.Error("Expected offset 10")
	}
}

func TestAddSliceOrSingleFilter(t *testing.T) {
	tests := []struct {
		name     string
		values   []string
		expected any
	}{
		{
			name:     "empty slice",
			values:   []string{},
			expected: nil,
		},
		{
			name:     "single value",
			values:   []string{"value1"},
			expected: "value1",
		},
		{
			name:     "multiple values",
			values:   []string{"value1", "value2"},
			expected: []string{"value1", "value2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := make(map[string]any)
			addSliceOrSingleFilter(m, "key", tt.values)

			if tt.expected == nil {
				if _, ok := m["key"]; ok {
					t.Error("Expected key to not be set")
				}
			} else {
				switch exp := tt.expected.(type) {
				case string:
					if val, ok := m["key"].(string); !ok || val != exp {
						t.Errorf("Expected %q, got %v", exp, m["key"])
					}
				case []string:
					if val, ok := m["key"].([]string); !ok || !strSliceEqual(val, exp) {
						t.Errorf("Expected %v, got %v", exp, m["key"])
					}
				}
			}
		})
	}
}

func TestAddBoolFilter(t *testing.T) {
	t.Run("nil value", func(t *testing.T) {
		m := make(map[string]any)
		addBoolFilter(m, "key", nil)
		if _, ok := m["key"]; ok {
			t.Error("Expected key to not be set for nil value")
		}
	})

	t.Run("true value", func(t *testing.T) {
		m := make(map[string]any)
		val := true
		addBoolFilter(m, "key", &val)
		if v, ok := m["key"].(bool); !ok || !v {
			t.Error("Expected key to be true")
		}
	})

	t.Run("false value", func(t *testing.T) {
		m := make(map[string]any)
		val := false
		addBoolFilter(m, "key", &val)
		if v, ok := m["key"].(bool); !ok || v {
			t.Error("Expected key to be false")
		}
	})
}

func TestAddStringFilter(t *testing.T) {
	t.Run("empty string", func(t *testing.T) {
		m := make(map[string]any)
		addStringFilter(m, "key", "")
		if _, ok := m["key"]; ok {
			t.Error("Expected key to not be set for empty string")
		}
	})

	t.Run("non-empty string", func(t *testing.T) {
		m := make(map[string]any)
		addStringFilter(m, "key", "value")
		if v, ok := m["key"].(string); !ok || v != "value" {
			t.Errorf("Expected 'value', got %v", m["key"])
		}
	})
}

// Helper functions
func boolPtr(b bool) *bool {
	return &b
}

func int64Ptr(i int64) *int64 {
	return &i
}

func intPtr(i int) *int {
	return &i
}

func strSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
