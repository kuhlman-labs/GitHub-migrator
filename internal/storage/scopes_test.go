package storage

import (
	"testing"
)

// TestParseFilterValue tests the parseFilterValue helper function
func TestParseFilterValue(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantOperator string
		wantValue    int
		wantErr      bool
	}{
		// Common patterns
		{
			name:         "greater than zero with space",
			input:        "> 0",
			wantOperator: ">",
			wantValue:    0,
			wantErr:      false,
		},
		{
			name:         "greater than zero without space",
			input:        ">0",
			wantOperator: ">",
			wantValue:    0,
			wantErr:      false,
		},
		{
			name:         "equals zero with space",
			input:        "= 0",
			wantOperator: "=",
			wantValue:    0,
			wantErr:      false,
		},
		{
			name:         "equals zero without space",
			input:        "=0",
			wantOperator: "=",
			wantValue:    0,
			wantErr:      false,
		},
		{
			name:         "just zero",
			input:        "0",
			wantOperator: "=",
			wantValue:    0,
			wantErr:      false,
		},
		// Greater than or equal
		{
			name:         "greater than or equal",
			input:        ">=5",
			wantOperator: ">=",
			wantValue:    5,
			wantErr:      false,
		},
		{
			name:         "greater than or equal with space",
			input:        ">= 10",
			wantOperator: ">=",
			wantValue:    10,
			wantErr:      false,
		},
		// Less than or equal
		{
			name:         "less than or equal",
			input:        "<=100",
			wantOperator: "<=",
			wantValue:    100,
			wantErr:      false,
		},
		{
			name:         "less than or equal with space",
			input:        "<= 50",
			wantOperator: "<=",
			wantValue:    50,
			wantErr:      false,
		},
		// Greater than
		{
			name:         "greater than",
			input:        ">10",
			wantOperator: ">",
			wantValue:    10,
			wantErr:      false,
		},
		{
			name:         "greater than with space",
			input:        "> 25",
			wantOperator: ">",
			wantValue:    25,
			wantErr:      false,
		},
		// Less than
		{
			name:         "less than",
			input:        "<5",
			wantOperator: "<",
			wantValue:    5,
			wantErr:      false,
		},
		{
			name:         "less than with space",
			input:        "< 3",
			wantOperator: "<",
			wantValue:    3,
			wantErr:      false,
		},
		// Equals
		{
			name:         "equals",
			input:        "=42",
			wantOperator: "=",
			wantValue:    42,
			wantErr:      false,
		},
		{
			name:         "equals with space",
			input:        "= 7",
			wantOperator: "=",
			wantValue:    7,
			wantErr:      false,
		},
		// No operator (implicit equals)
		{
			name:         "no operator - implicit equals",
			input:        "15",
			wantOperator: "=",
			wantValue:    15,
			wantErr:      false,
		},
		// Whitespace handling
		{
			name:         "leading whitespace",
			input:        "  > 5",
			wantOperator: ">",
			wantValue:    5,
			wantErr:      false,
		},
		{
			name:         "trailing whitespace",
			input:        ">5  ",
			wantOperator: ">",
			wantValue:    5,
			wantErr:      false,
		},
		// Error cases
		{
			name:         "invalid - non-numeric",
			input:        ">abc",
			wantOperator: "",
			wantValue:    0,
			wantErr:      true,
		},
		{
			name:         "invalid - negative value",
			input:        ">-5",
			wantOperator: "",
			wantValue:    0,
			wantErr:      true,
		},
		{
			name:         "invalid - empty string",
			input:        "",
			wantOperator: "",
			wantValue:    0,
			wantErr:      true,
		},
		{
			name:         "invalid - only operator",
			input:        ">",
			wantOperator: "",
			wantValue:    0,
			wantErr:      true,
		},
		{
			name:         "invalid - float value",
			input:        ">5.5",
			wantOperator: "",
			wantValue:    0,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			operator, value, err := parseFilterValue(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseFilterValue(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}

			if err == nil {
				if operator != tt.wantOperator {
					t.Errorf("parseFilterValue(%q) operator = %q, want %q", tt.input, operator, tt.wantOperator)
				}
				if value != tt.wantValue {
					t.Errorf("parseFilterValue(%q) value = %d, want %d", tt.input, value, tt.wantValue)
				}
			}
		})
	}
}

// TestParseFilterValueEdgeCases tests additional edge cases for parseFilterValue
func TestParseFilterValueEdgeCases(t *testing.T) {
	// Test large numbers
	t.Run("large number", func(t *testing.T) {
		operator, value, err := parseFilterValue(">1000000")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if operator != ">" {
			t.Errorf("Expected operator '>', got %q", operator)
		}
		if value != 1000000 {
			t.Errorf("Expected value 1000000, got %d", value)
		}
	})

	// Test operator parsing order (>= before >)
	t.Run("operator precedence >= vs >", func(t *testing.T) {
		operator, _, err := parseFilterValue(">=5")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if operator != ">=" {
			t.Errorf("Expected operator '>=', got %q", operator)
		}
	})

	// Test operator parsing order (<= before <)
	t.Run("operator precedence <= vs <", func(t *testing.T) {
		operator, _, err := parseFilterValue("<=5")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if operator != "<=" {
			t.Errorf("Expected operator '<=', got %q", operator)
		}
	})
}
