package discovery

import (
	"log/slog"
	"os"
	"testing"
)

func TestNewTeamDiscoverer(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	tests := []struct {
		name            string
		workers         int
		expectedWorkers int
	}{
		{
			name:            "positive workers",
			workers:         10,
			expectedWorkers: 10,
		},
		{
			name:            "zero workers defaults to 5",
			workers:         0,
			expectedWorkers: 5,
		},
		{
			name:            "negative workers defaults to 5",
			workers:         -1,
			expectedWorkers: 5,
		},
		{
			name:            "one worker",
			workers:         1,
			expectedWorkers: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewTeamDiscoverer(nil, logger, tt.workers)

			if d == nil {
				t.Fatal("Expected non-nil TeamDiscoverer")
				return
			}

			if d.workers != tt.expectedWorkers {
				t.Errorf("Expected %d workers, got %d", tt.expectedWorkers, d.workers)
			}

			if d.logger != logger {
				t.Error("Logger not set correctly")
			}
		})
	}
}

func TestTeamResult(t *testing.T) {
	result := teamResult{
		teamSaved:   true,
		memberCount: 5,
		repoCount:   10,
		err:         nil,
	}

	if !result.teamSaved {
		t.Error("Expected teamSaved to be true")
	}

	if result.memberCount != 5 {
		t.Errorf("Expected memberCount 5, got %d", result.memberCount)
	}

	if result.repoCount != 10 {
		t.Errorf("Expected repoCount 10, got %d", result.repoCount)
	}

	if result.err != nil {
		t.Error("Expected err to be nil")
	}
}

func TestStringPtr(t *testing.T) {
	// Test the stringPtr helper function
	result := stringPtr("test")

	if result == nil {
		t.Fatal("Expected non-nil pointer")
		return
	}

	if *result != "test" {
		t.Errorf("Expected 'test', got %q", *result)
	}
}

func TestTeamDiscoverer_WorkerCount(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	// Verify that worker count is accessible and correct
	d := NewTeamDiscoverer(nil, logger, 3)

	if d.workers != 3 {
		t.Errorf("Expected 3 workers, got %d", d.workers)
	}
}
