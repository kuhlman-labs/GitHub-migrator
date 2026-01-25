package copilot

import (
	"testing"
)

func TestIntentDetector_DetectIntent(t *testing.T) {
	detector := NewIntentDetector()

	tests := []struct {
		name         string
		message      string
		expectedTool string
		shouldMatch  bool
	}{
		// find_pilot_candidates tests - focus on correct tool detection
		{
			name:         "pilot migration query",
			message:      "Find repositories suitable for a pilot migration",
			expectedTool: "find_pilot_candidates",
			shouldMatch:  true,
		},
		{
			name:         "pilot candidates explicit",
			message:      "What are good pilot candidates?",
			expectedTool: "find_pilot_candidates",
			shouldMatch:  true,
		},
		{
			name:         "best repos to start",
			message:      "Find the best repos to start with",
			expectedTool: "find_pilot_candidates",
			shouldMatch:  true,
		},
		{
			name:         "simple repos for first wave",
			message:      "Show me simple repositories for the first wave",
			expectedTool: "find_pilot_candidates",
			shouldMatch:  true,
		},

		// analyze_repositories tests
		{
			name:         "analyze repos query",
			message:      "Analyze all repositories in the organization",
			expectedTool: "analyze_repositories",
			shouldMatch:  true,
		},
		{
			name:         "list pending repos",
			message:      "Show me pending repos",
			expectedTool: "analyze_repositories",
			shouldMatch:  true,
		},
		{
			name:         "repository overview",
			message:      "Give me a repository overview",
			expectedTool: "analyze_repositories",
			shouldMatch:  true,
		},

		// create_batch tests
		{
			name:         "create batch explicit",
			message:      "Create a batch with these repositories",
			expectedTool: "create_batch",
			shouldMatch:  true,
		},
		{
			name:         "batch named",
			message:      "Create a batch called pilot-wave-1",
			expectedTool: "create_batch",
			shouldMatch:  true,
		},
		{
			name:         "yes create batch follow-up",
			message:      "Yes, create the batch",
			expectedTool: "create_batch",
			shouldMatch:  true,
		},

		// check_dependencies tests
		{
			name:         "check dependencies",
			message:      "Check dependencies for org/repo",
			expectedTool: "check_dependencies",
			shouldMatch:  true,
		},
		{
			name:         "what depends on",
			message:      "What does my-org/my-repo depend on?",
			expectedTool: "check_dependencies",
			shouldMatch:  true,
		},

		// plan_waves tests
		{
			name:         "plan migration waves",
			message:      "Plan migration waves for the organization",
			expectedTool: "plan_waves",
			shouldMatch:  true,
		},
		{
			name:         "wave strategy",
			message:      "Create a wave strategy to minimize downtime",
			expectedTool: "plan_waves",
			shouldMatch:  true,
		},

		// get_complexity_breakdown tests
		{
			name:         "complexity breakdown",
			message:      "Show complexity breakdown for org/complex-repo",
			expectedTool: "get_complexity_breakdown",
			shouldMatch:  true,
		},
		{
			name:         "why is repo complex",
			message:      "Why is my-org/large-repo so complex?",
			expectedTool: "get_complexity_breakdown",
			shouldMatch:  true,
		},

		// get_team_repositories tests
		{
			name:         "team repos",
			message:      "Show repositories for team engineering/platform",
			expectedTool: "get_team_repositories",
			shouldMatch:  true,
		},

		// schedule_batch tests
		{
			name:         "schedule batch",
			message:      "Schedule batch pilot-wave-1 for 2024-01-15T09:00:00Z",
			expectedTool: "schedule_batch",
			shouldMatch:  true,
		},

		// No match tests
		{
			name:         "general question",
			message:      "How does this migration tool work?",
			expectedTool: "",
			shouldMatch:  false,
		},
		{
			name:         "greeting",
			message:      "Hello, can you help me?",
			expectedTool: "",
			shouldMatch:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intent := detector.DetectIntent(tt.message)

			if tt.shouldMatch {
				if intent == nil {
					t.Errorf("Expected intent for message %q, got nil", tt.message)
					return
				}
				if intent.Tool != tt.expectedTool {
					t.Errorf("Expected tool %q, got %q for message %q", tt.expectedTool, intent.Tool, tt.message)
				}
				// Verify confidence is positive (intent was detected)
				if intent.Confidence <= 0 {
					t.Errorf("Expected positive confidence, got %v for message %q", intent.Confidence, tt.message)
				}
			} else {
				// For non-matching messages, either nil or very low confidence
				if intent != nil && intent.Confidence >= 0.5 {
					t.Errorf("Expected no confident match for message %q, got tool=%q confidence=%v", tt.message, intent.Tool, intent.Confidence)
				}
			}
		})
	}
}

func TestIntentDetector_ExtractArgs(t *testing.T) {
	detector := NewIntentDetector()

	tests := []struct {
		name        string
		message     string
		expectedKey string
		expectedVal any
	}{
		{
			name:        "extract organization",
			message:     "Find pilot candidates in org my-company",
			expectedKey: "organization",
			expectedVal: "my-company",
		},
		{
			name:        "extract batch name",
			message:     "Create a batch called my-batch-name",
			expectedKey: "name",
			expectedVal: "my-batch-name",
		},
		{
			name:        "extract repository",
			message:     "Check dependencies for acme/backend-service",
			expectedKey: "repository",
			expectedVal: "acme/backend-service",
		},
		{
			name:        "extract pending status",
			message:     "Show me pending repositories",
			expectedKey: "status",
			expectedVal: "pending",
		},
		{
			name:        "extract completed status",
			message:     "List all completed repos",
			expectedKey: "status",
			expectedVal: "completed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intent := detector.DetectIntent(tt.message)
			if intent == nil {
				t.Fatalf("Expected intent for message %q, got nil", tt.message)
			}

			if val, ok := intent.Args[tt.expectedKey]; !ok {
				t.Errorf("Expected arg %q to be present for message %q", tt.expectedKey, tt.message)
			} else if val != tt.expectedVal {
				t.Errorf("Expected arg %q=%v, got %v for message %q", tt.expectedKey, tt.expectedVal, val, tt.message)
			}
		})
	}
}

func TestIntentDetector_IsFollowUpBatchCreate(t *testing.T) {
	detector := NewIntentDetector()

	tests := []struct {
		name     string
		message  string
		expected bool
	}{
		{
			name:     "yes response",
			message:  "Yes",
			expected: true,
		},
		{
			name:     "ok response",
			message:  "Ok, create the batch",
			expected: true,
		},
		{
			name:     "please create",
			message:  "Please create a batch with these",
			expected: true,
		},
		{
			name:     "sounds good",
			message:  "Sounds good, let's do it",
			expected: true,
		},
		{
			name:     "go ahead",
			message:  "Go ahead",
			expected: true,
		},
		{
			name:     "negative response",
			message:  "No, show me more options",
			expected: false,
		},
		{
			name:     "different question",
			message:  "What about dependencies?",
			expected: false,
		},
		{
			name:     "create the batch",
			message:  "create the batch",
			expected: true,
		},
		{
			name:     "create the batch with destination",
			message:  "create the batch and set the destination organization to kuhlman-labs-org-emu",
			expected: true,
		},
		{
			name:     "make the batch",
			message:  "make the batch",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.IsFollowUpBatchCreate(tt.message)
			if result != tt.expected {
				t.Errorf("Expected %v for message %q, got %v", tt.expected, tt.message, result)
			}
		})
	}
}

func TestIntentDetector_ExtractBatchNameFromFollowUp(t *testing.T) {
	detector := NewIntentDetector()

	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{
			name:     "batch called name",
			message:  "Yes, create a batch called pilot-v1",
			expected: "pilot-v1",
		},
		{
			name:     "batch named",
			message:  "Create it named my-batch",
			expected: "my-batch",
		},
		{
			name:     "no name",
			message:  "Yes, create the batch",
			expected: "",
		},
		{
			name:     "name with quotes",
			message:  "Create batch 'test-batch'",
			expected: "test-batch",
		},
		{
			name:     "batch and set destination should use default",
			message:  "create the batch and set the destination organization to kuhlman-labs-org-emu",
			expected: "",
		},
		{
			name:     "please create the batch should use default",
			message:  "please create the batch",
			expected: "",
		},
		{
			name:     "explicit name with destination",
			message:  "create a batch called pilot-migration and set destination to my-org",
			expected: "pilot-migration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.ExtractBatchNameFromFollowUp(tt.message)
			if result != tt.expected {
				t.Errorf("Expected %q for message %q, got %q", tt.expected, tt.message, result)
			}
		})
	}
}

func TestIntentDetector_ExtractDestinationOrg(t *testing.T) {
	detector := NewIntentDetector()

	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{
			name:     "destination organization to",
			message:  "set the destination organization to kuhlman-labs-org-emu",
			expected: "kuhlman-labs-org-emu",
		},
		{
			name:     "destination org",
			message:  "destination org my-org",
			expected: "my-org",
		},
		{
			name:     "migrate to org",
			message:  "migrate to the org target-org-123",
			expected: "target-org-123",
		},
		{
			name:     "set destination to",
			message:  "set destination to production-org",
			expected: "production-org",
		},
		{
			name:     "no destination",
			message:  "create the batch please",
			expected: "",
		},
		{
			name:     "complex message with destination",
			message:  "create the batch and set the destination organization to kuhlman-labs-org-emu",
			expected: "kuhlman-labs-org-emu",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.ExtractDestinationOrg(tt.message)
			if result != tt.expected {
				t.Errorf("Expected %q for message %q, got %q", tt.expected, tt.message, result)
			}
		})
	}
}

func TestDetectedIntent_IsConfident(t *testing.T) {
	tests := []struct {
		name     string
		intent   *DetectedIntent
		expected bool
	}{
		{
			name:     "nil intent",
			intent:   nil,
			expected: false,
		},
		{
			name: "very low confidence",
			intent: &DetectedIntent{
				Tool:       "test",
				Confidence: 0.05,
			},
			expected: false,
		},
		{
			name: "exactly threshold",
			intent: &DetectedIntent{
				Tool:       "test",
				Confidence: 0.1,
			},
			expected: true,
		},
		{
			name: "above threshold",
			intent: &DetectedIntent{
				Tool:       "test",
				Confidence: 0.3,
			},
			expected: true,
		},
		{
			name: "high confidence",
			intent: &DetectedIntent{
				Tool:       "test",
				Confidence: 0.95,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.intent.IsConfident()
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
