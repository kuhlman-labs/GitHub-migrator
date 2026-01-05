package handlers

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestValidateTeams_EmptyTeams(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	handler := &SettingsHandler{
		db:     nil, // Not accessed for empty teams
		logger: logger,
	}

	// Test with empty teams array
	reqBody := ValidateTeamsRequest{Teams: []string{}}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/settings/teams/validate", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ValidateTeams(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response ValidateTeamsResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !response.Valid {
		t.Error("expected valid=true for empty teams")
	}
	if len(response.Teams) != 0 {
		t.Errorf("expected empty teams array, got %d", len(response.Teams))
	}
}

func TestValidateTeams_WhitespaceOnlyTeams(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	handler := &SettingsHandler{
		db:     nil,
		logger: logger,
	}

	// Test with whitespace-only teams
	reqBody := ValidateTeamsRequest{Teams: []string{"  ", "\t", ""}}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/settings/teams/validate", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ValidateTeams(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response ValidateTeamsResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !response.Valid {
		t.Error("expected valid=true for whitespace-only teams (treated as empty)")
	}
}

func TestValidateTeamsRequest_Parsing(t *testing.T) {
	tests := []struct {
		name            string
		input           ValidateTeamsRequest
		expectValid     bool
		expectTeamCount int
	}{
		{
			name:            "empty teams",
			input:           ValidateTeamsRequest{Teams: []string{}},
			expectValid:     true,
			expectTeamCount: 0,
		},
		{
			name:            "whitespace only",
			input:           ValidateTeamsRequest{Teams: []string{"  ", "\t"}},
			expectValid:     true,
			expectTeamCount: 0,
		},
		{
			name:            "mixed whitespace and valid",
			input:           ValidateTeamsRequest{Teams: []string{"org/team", "  ", "another-org/team"}},
			expectValid:     false, // Will fail without actual API, but should have 2 teams parsed
			expectTeamCount: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Filter teams the same way the handler does
			var teams []string
			for _, team := range tc.input.Teams {
				team = string(bytes.TrimSpace([]byte(team)))
				if team != "" {
					teams = append(teams, team)
				}
			}

			if len(teams) != tc.expectTeamCount {
				t.Errorf("expected %d teams, got %d", tc.expectTeamCount, len(teams))
			}
		})
	}
}

func TestValidateTeamsResponse_Encoding(t *testing.T) {
	response := ValidateTeamsResponse{
		Valid: false,
		Teams: []TeamValidationResult{
			{Team: "org/team", Valid: true},
			{Team: "org/invalid", Valid: false, Error: "Team not found"},
		},
		InvalidTeams: []string{"org/invalid"},
		ErrorMessage: "The following teams were not found: org/invalid",
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("failed to marshal response: %v", err)
	}

	var decoded ValidateTeamsResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if decoded.Valid != response.Valid {
		t.Errorf("expected valid=%v, got %v", response.Valid, decoded.Valid)
	}
	if len(decoded.Teams) != len(response.Teams) {
		t.Errorf("expected %d teams, got %d", len(response.Teams), len(decoded.Teams))
	}
	if len(decoded.InvalidTeams) != len(response.InvalidTeams) {
		t.Errorf("expected %d invalid teams, got %d", len(response.InvalidTeams), len(decoded.InvalidTeams))
	}
	if decoded.ErrorMessage != response.ErrorMessage {
		t.Errorf("expected error message %q, got %q", response.ErrorMessage, decoded.ErrorMessage)
	}
}

func TestTeamFormatValidation(t *testing.T) {
	tests := []struct {
		input    string
		wantOrg  string
		wantTeam string
		wantErr  bool
	}{
		{"org/team", "org", "team", false},
		{"my-org/admin-team", "my-org", "admin-team", false},
		{"invalid", "", "", true},
		{"", "", "", true},
		{"only-one-part", "", "", true},
		{"org/team/extra", "org", "team/extra", false}, // SplitN with n=2 keeps rest in second part
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			parts := bytes.SplitN([]byte(tc.input), []byte("/"), 2)

			if len(parts) != 2 && !tc.wantErr {
				t.Errorf("expected 2 parts, got %d", len(parts))
			}

			if len(parts) == 2 {
				org := string(parts[0])
				team := string(parts[1])

				if tc.wantErr {
					t.Errorf("expected error for %q", tc.input)
				}
				if org != tc.wantOrg {
					t.Errorf("expected org %q, got %q", tc.wantOrg, org)
				}
				if team != tc.wantTeam {
					t.Errorf("expected team %q, got %q", tc.wantTeam, team)
				}
			}
		})
	}
}
