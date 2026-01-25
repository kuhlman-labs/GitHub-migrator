// Package copilot provides the Copilot chat service integration.
package copilot

import (
	"regexp"
	"strings"
)

// DetectedIntent represents a detected user intent with confidence score
type DetectedIntent struct {
	Tool       string         // Name of the tool to execute
	Args       map[string]any // Extracted arguments
	Confidence float64        // Confidence score (0.0 - 1.0)
	Patterns   []string       // Patterns that matched
}

// IntentDetector detects user intents from natural language messages
type IntentDetector struct {
	patterns map[string][]intentPattern
}

// intentPattern represents a pattern for intent matching
type intentPattern struct {
	regex  *regexp.Regexp
	weight float64
}

// NewIntentDetector creates a new intent detector with predefined patterns
func NewIntentDetector() *IntentDetector {
	detector := &IntentDetector{
		patterns: make(map[string][]intentPattern),
	}
	detector.registerPatterns()
	return detector
}

// DetectIntent analyzes a message and returns the detected intent
func (d *IntentDetector) DetectIntent(message string) *DetectedIntent {
	message = strings.ToLower(message)

	var bestIntent *DetectedIntent

	for tool, patterns := range d.patterns {
		matchedPatterns := []string{}
		totalWeight := 0.0

		for _, p := range patterns {
			if p.regex.MatchString(message) {
				matchedPatterns = append(matchedPatterns, p.regex.String())
				totalWeight += p.weight
			}
		}

		if len(matchedPatterns) > 0 {
			// Calculate confidence based on matched patterns
			confidence := totalWeight / float64(len(patterns))
			if confidence > 1.0 {
				confidence = 1.0
			}

			// Apply boost for multiple pattern matches
			if len(matchedPatterns) > 1 {
				confidence = confidence * (1 + float64(len(matchedPatterns)-1)*0.1)
				if confidence > 1.0 {
					confidence = 1.0
				}
			}

			if bestIntent == nil || confidence > bestIntent.Confidence {
				bestIntent = &DetectedIntent{
					Tool:       tool,
					Args:       d.extractArgs(tool, message),
					Confidence: confidence,
					Patterns:   matchedPatterns,
				}
			}
		}
	}

	return bestIntent
}

// registerPatterns registers all intent patterns for migration tools
func (d *IntentDetector) registerPatterns() {
	// find_pilot_candidates patterns
	d.patterns["find_pilot_candidates"] = []intentPattern{
		{regex: regexp.MustCompile(`pilot\s*(migration|candidate|repo)`), weight: 0.9},
		{regex: regexp.MustCompile(`(find|identify|show|get|list)\s*(the\s+)?(best|good|suitable|simple)\s*(repos?|repositories?)`), weight: 0.7},
		{regex: regexp.MustCompile(`(suitable|good|best)\s*(for\s+)?(pilot|first|initial)`), weight: 0.8},
		{regex: regexp.MustCompile(`(start|begin|initial)\s*(migration|with)`), weight: 0.5},
		{regex: regexp.MustCompile(`(low|simple|easy)\s*(complexity|risk)`), weight: 0.4},
		{regex: regexp.MustCompile(`where\s+(should|can)\s+(we|i)\s+start`), weight: 0.6},
		{regex: regexp.MustCompile(`first\s+(repos?|repositories?|wave|batch)`), weight: 0.5},
	}

	// analyze_repositories patterns
	d.patterns["analyze_repositories"] = []intentPattern{
		{regex: regexp.MustCompile(`(analyze|analyse|show|list|get)\s*(all\s+)?(repos?|repositories?)`), weight: 0.8},
		{regex: regexp.MustCompile(`(what|which)\s+(repos?|repositories?)`), weight: 0.6},
		{regex: regexp.MustCompile(`repository\s+(list|overview|summary)`), weight: 0.7},
		{regex: regexp.MustCompile(`(pending|in.?progress|completed|failed)\s+repos`), weight: 0.8},
		{regex: regexp.MustCompile(`repos?\s+in\s+(organization|org)\s+`), weight: 0.7},
		{regex: regexp.MustCompile(`complexity\s+(above|below|between|greater|less)`), weight: 0.6},
	}

	// create_batch patterns
	d.patterns["create_batch"] = []intentPattern{
		{regex: regexp.MustCompile(`(create|make|new)\s+(a\s+)?(batch|group)`), weight: 0.9},
		{regex: regexp.MustCompile(`batch\s+(with|containing|from)`), weight: 0.8},
		{regex: regexp.MustCompile(`(add|put)\s+(these|those|the)\s+repos?\s+(to|in)\s+(a\s+)?batch`), weight: 0.8},
		{regex: regexp.MustCompile(`group\s+(these|those|the)\s+repos?`), weight: 0.7},
		{regex: regexp.MustCompile(`(yes|ok|sure|please)\s*,?\s*(create|make)\s+(the\s+)?batch`), weight: 0.95},
		{regex: regexp.MustCompile(`batch\s+(called|named)\s+`), weight: 0.9},
	}

	// check_dependencies patterns
	d.patterns["check_dependencies"] = []intentPattern{
		{regex: regexp.MustCompile(`(check|show|find|get|list)\s+(the\s+)?dependenc`), weight: 0.9},
		{regex: regexp.MustCompile(`(what|which)\s+(does|are)\s+.+\s+depend`), weight: 0.8},
		{regex: regexp.MustCompile(`depends?\s+on`), weight: 0.7},
		{regex: regexp.MustCompile(`(upstream|downstream)\s+dependenc`), weight: 0.8},
		{regex: regexp.MustCompile(`(reverse\s+)?dependenc(y|ies)\s+(of|for)`), weight: 0.8},
	}

	// plan_waves patterns
	d.patterns["plan_waves"] = []intentPattern{
		{regex: regexp.MustCompile(`(plan|create|make|generate)\s+(migration\s+)?waves?`), weight: 0.9},
		{regex: regexp.MustCompile(`wave\s+(plan|strategy|schedule)`), weight: 0.85},
		{regex: regexp.MustCompile(`(migration|rollout)\s+(strategy|plan|waves?)`), weight: 0.8},
		{regex: regexp.MustCompile(`(order|sequence)\s+(of\s+)?migration`), weight: 0.7},
		{regex: regexp.MustCompile(`(minimize|reduce)\s+(downtime|risk|disruption)`), weight: 0.5},
		{regex: regexp.MustCompile(`respect\s+dependenc`), weight: 0.6},
	}

	// get_complexity_breakdown patterns
	d.patterns["get_complexity_breakdown"] = []intentPattern{
		{regex: regexp.MustCompile(`complexity\s+(breakdown|details|score|analysis)`), weight: 0.9},
		{regex: regexp.MustCompile(`(why|how)\s+(is|come)\s+.+\s+(complex|difficult)`), weight: 0.7},
		{regex: regexp.MustCompile(`(analyze|analyse)\s+.+\s+complexity`), weight: 0.8},
		{regex: regexp.MustCompile(`what\s+(makes?|causes?)\s+.+\s+complex`), weight: 0.7},
		{regex: regexp.MustCompile(`(blockers?|warnings?)\s+(for|in)\s+`), weight: 0.6},
	}

	// get_team_repositories patterns
	d.patterns["get_team_repositories"] = []intentPattern{
		{regex: regexp.MustCompile(`(repos?|repositories?)\s+(for|owned\s+by|belonging\s+to)\s+(team|the\s+team)`), weight: 0.9},
		{regex: regexp.MustCompile(`team\s+.+\s+repos?`), weight: 0.8},
		{regex: regexp.MustCompile(`(what|which)\s+repos?\s+(does|do)\s+team`), weight: 0.8},
		{regex: regexp.MustCompile(`(list|show|get)\s+.+\s+team\s+repos?`), weight: 0.8},
	}

	// get_migration_status patterns
	d.patterns["get_migration_status"] = []intentPattern{
		{regex: regexp.MustCompile(`(migration\s+)?status\s+(of|for)\s+`), weight: 0.9},
		{regex: regexp.MustCompile(`(is|are)\s+.+\s+migrated`), weight: 0.7},
		{regex: regexp.MustCompile(`(check|get|show)\s+(the\s+)?status`), weight: 0.6},
		{regex: regexp.MustCompile(`(progress|state)\s+(of|for)\s+`), weight: 0.7},
	}

	// schedule_batch patterns
	d.patterns["schedule_batch"] = []intentPattern{
		{regex: regexp.MustCompile(`schedule\s+(a\s+)?batch`), weight: 0.9},
		{regex: regexp.MustCompile(`(run|execute)\s+(batch|migration)\s+(at|on|for)`), weight: 0.8},
		{regex: regexp.MustCompile(`(set|plan)\s+(the\s+)?execution\s+(date|time)`), weight: 0.7},
		{regex: regexp.MustCompile(`batch\s+.+\s+(at|on|for)\s+\d`), weight: 0.8},
	}

	// configure_batch patterns - for setting destination org, migration API, etc.
	d.patterns["configure_batch"] = []intentPattern{
		{regex: regexp.MustCompile(`(set|configure|change|update)\s+(the\s+)?destination\s+(org|organization)`), weight: 0.95},
		{regex: regexp.MustCompile(`migrate\s+(to|into)\s+(the\s+)?(org|organization)\s+`), weight: 0.9},
		{regex: regexp.MustCompile(`destination\s+(org|organization)\s+(to|for|is)\s+`), weight: 0.9},
		{regex: regexp.MustCompile(`(to|into)\s+["']?[\w-]+["']?\s+(org|organization|destination)`), weight: 0.85},
		{regex: regexp.MustCompile(`(set|use|target)\s+["']?[\w-]+["']?\s+(as\s+)?(the\s+)?destination`), weight: 0.85},
		{regex: regexp.MustCompile(`batch\s+.+\s+(migrate|go)\s+to\s+`), weight: 0.8},
		{regex: regexp.MustCompile(`(configure|set)\s+(the\s+)?batch\s+`), weight: 0.7},
		{regex: regexp.MustCompile(`(use|set)\s+(gei|elm)\s+(api|for)`), weight: 0.8},
	}
}

// extractArgs extracts arguments from the message based on the tool type.
// Each tool has unique argument extraction patterns, so complexity is inherent.
// nolint:gocyclo
func (d *IntentDetector) extractArgs(tool, message string) map[string]any {
	args := make(map[string]any)

	switch tool {
	case "find_pilot_candidates":
		// Extract max_count if mentioned
		if countMatch := regexp.MustCompile(`(\d+)\s*(repos?|repositories?|candidates?)`).FindStringSubmatch(message); len(countMatch) > 1 {
			args["max_count"] = countMatch[1]
		}
		// Extract organization if mentioned
		if orgMatch := regexp.MustCompile(`(org|organization)\s+["']?(\w+[-/\w]*)["']?`).FindStringSubmatch(message); len(orgMatch) > 2 {
			args["organization"] = orgMatch[2]
		}

	case ToolAnalyzeRepositories:
		// Extract status filter
		if strings.Contains(message, "pending") {
			args["status"] = StatusPending
		} else if strings.Contains(message, "completed") || strings.Contains(message, "migrated") {
			args["status"] = StatusCompleted
		} else if strings.Contains(message, "failed") {
			args["status"] = "failed"
		} else if strings.Contains(message, "in progress") || strings.Contains(message, "in-progress") {
			args["status"] = "in_progress"
		}
		// Extract organization
		if orgMatch := regexp.MustCompile(`(org|organization)\s+["']?(\w+[-/\w]*)["']?`).FindStringSubmatch(message); len(orgMatch) > 2 {
			args["organization"] = orgMatch[2]
		}

	case "create_batch":
		// Extract batch name
		if nameMatch := regexp.MustCompile(`(called|named)\s+["']?([^"'\s]+)["']?`).FindStringSubmatch(message); len(nameMatch) > 2 {
			args["name"] = nameMatch[2]
		}
		// Extract destination organization if specified
		destPatterns := []*regexp.Regexp{
			regexp.MustCompile(`destination\s+(?:org(?:anization)?\s+)?(?:to\s+)?["']?([\w][\w-]*[\w])["']?`),
			regexp.MustCompile(`(?:set|configure)\s+(?:the\s+)?destination\s+(?:org(?:anization)?\s+)?(?:to\s+)?["']?([\w][\w-]*[\w])["']?`),
			regexp.MustCompile(`(?:migrate|send)\s+(?:to|into)\s+(?:the\s+)?(?:org(?:anization)?\s+)?["']?([\w][\w-]*[\w])["']?`),
		}
		for _, p := range destPatterns {
			if destMatch := p.FindStringSubmatch(message); len(destMatch) > 1 {
				dest := destMatch[1]
				commonWords := map[string]bool{"the": true, "org": true, "organization": true, "to": true, "a": true, "set": true, "and": true}
				if !commonWords[dest] {
					args["destination_org"] = dest
					break
				}
			}
		}
		// Note: repositories will be filled from previous tool results

	case "check_dependencies", "get_complexity_breakdown":
		// Extract repository name (format: org/repo)
		if repoMatch := regexp.MustCompile(`([\w-]+/[\w.-]+)`).FindStringSubmatch(message); len(repoMatch) > 1 {
			args["repository"] = repoMatch[1]
		}
		// Check for reverse dependencies
		if strings.Contains(message, "reverse") || strings.Contains(message, "depend on this") {
			args["include_reverse"] = true
		}

	case "get_team_repositories":
		// Extract team name (format: org/team-slug)
		if teamMatch := regexp.MustCompile(`team\s+["']?([\w-]+/[\w.-]+)["']?`).FindStringSubmatch(message); len(teamMatch) > 1 {
			args["team"] = teamMatch[1]
		}

	case "plan_waves":
		// Extract wave size
		if sizeMatch := regexp.MustCompile(`(\d+)\s*(repos?|repositories?)\s*(per|each)\s*wave`).FindStringSubmatch(message); len(sizeMatch) > 1 {
			args["wave_size"] = sizeMatch[1]
		}
		// Extract organization
		if orgMatch := regexp.MustCompile(`(org|organization)\s+["']?(\w+[-/\w]*)["']?`).FindStringSubmatch(message); len(orgMatch) > 2 {
			args["organization"] = orgMatch[2]
		}

	case "schedule_batch":
		// Extract batch name
		if nameMatch := regexp.MustCompile(`batch\s+["']?([^"'\s]+)["']?`).FindStringSubmatch(message); len(nameMatch) > 1 {
			args["batch_name"] = nameMatch[1]
		}
		// Extract datetime - look for ISO format or common patterns
		if timeMatch := regexp.MustCompile(`(\d{4}-\d{2}-\d{2}(?:T\d{2}:\d{2}:\d{2}Z?)?)`).FindStringSubmatch(message); len(timeMatch) > 1 {
			args["scheduled_at"] = timeMatch[1]
		}
		// Also extract destination org if mentioned
		if destMatch := regexp.MustCompile(`(?:to|into|destination)\s+(?:org(?:anization)?\s+)?["']?([\w][\w-]*[\w])["']?`).FindStringSubmatch(message); len(destMatch) > 1 {
			args["destination_org"] = destMatch[1]
		}

	case "configure_batch":
		// Extract batch name or ID - be more specific to avoid matching common words
		if nameMatch := regexp.MustCompile(`batch\s+(?:named|called)?\s*["']?([a-zA-Z0-9][\w-]*[a-zA-Z0-9])["']?`).FindStringSubmatch(message); len(nameMatch) > 1 {
			name := nameMatch[1]
			// Filter out common words that aren't batch names
			commonWords := map[string]bool{"to": true, "with": true, "for": true, "the": true, "and": true, "or": true, "set": true, "from": true, "into": true}
			if !commonWords[name] {
				args["batch_name"] = name
			}
		}
		if idMatch := regexp.MustCompile(`batch\s+(?:id\s+)?#?(\d+)`).FindStringSubmatch(message); len(idMatch) > 1 {
			args["batch_id"] = idMatch[1]
		}

		// Extract destination organization - multiple patterns
		destPatterns := []*regexp.Regexp{
			// "destination org kuhlman-labs-org-emu"
			regexp.MustCompile(`destination\s+(?:org(?:anization)?\s+)?["']?([\w][\w-]*[\w])["']?`),
			// "to the kuhlman-labs-org-emu organization"
			regexp.MustCompile(`(?:to|into)\s+(?:the\s+)?["']?([\w][\w-]*[\w])["']?\s+(?:org|organization|dest)`),
			// "migrate to kuhlman-labs-org-emu"
			regexp.MustCompile(`migrate\s+(?:to|into)\s+(?:the\s+)?(?:org(?:anization)?\s+)?["']?([\w][\w-]*[\w])["']?`),
			// "set destination to kuhlman-labs-org-emu"
			regexp.MustCompile(`(?:set|use|target)\s+(?:destination\s+)?(?:to\s+)?["']?([\w][\w-]*[\w])["']?`),
			// "kuhlman-labs-org-emu destination"
			regexp.MustCompile(`["']?([\w][\w-]*[\w])["']?\s+(?:as\s+)?destination`),
		}
		for _, p := range destPatterns {
			if destMatch := p.FindStringSubmatch(message); len(destMatch) > 1 {
				dest := destMatch[1]
				// Filter out common words
				if dest != "the" && dest != "org" && dest != "organization" && dest != "to" && dest != "a" && dest != "set" {
					args["destination_org"] = dest
					break
				}
			}
		}

		// Extract migration API
		if apiMatch := regexp.MustCompile(`(?:use|set|with)\s+(gei|elm)\s*(?:api)?`).FindStringSubmatch(message); len(apiMatch) > 1 {
			args["migration_api"] = strings.ToUpper(apiMatch[1])
		}
	}

	return args
}

// IsConfident returns true if the intent has sufficient confidence
// The threshold is set to 0.1 since confidence is calculated as a ratio of
// matched pattern weights to total patterns, and even a single strong match
// indicates user intent.
func (d *DetectedIntent) IsConfident() bool {
	return d != nil && d.Confidence >= 0.1
}

// IsFollowUpBatchCreate checks if this is a follow-up to create a batch from previous results
func (d *IntentDetector) IsFollowUpBatchCreate(message string) bool {
	message = strings.ToLower(message)
	patterns := []string{
		`^(yes|ok|sure|please|yeah|yep|absolutely)`,            // Simple affirmative at start
		`(yes|ok|sure|please|yeah|yep|absolutely)\s*[,.]?\s*$`, // Simple affirmative at end
		`create\s+(a\s+|the\s+)?batch`,                         // "create the batch" or "create a batch"
		`(batch|group)\s+(these|those|them)`,
		`sounds?\s+good`,
		`let'?s?\s+do\s+(it|that)`,
		`go\s+ahead`,
		`make\s+(a\s+|the\s+)?batch`,
	}

	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, message); matched {
			return true
		}
	}
	return false
}

// ExtractBatchNameFromFollowUp extracts a batch name from a follow-up message.
// Only extracts names that are EXPLICITLY specified by the user.
// Returns empty string to use the default name if no explicit name is given.
func (d *IntentDetector) ExtractBatchNameFromFollowUp(message string) string {
	message = strings.ToLower(message)

	// Only match explicit naming patterns - don't try to extract names from general text
	patterns := []*regexp.Regexp{
		// "batch called my-batch" or "batch named my-batch"
		regexp.MustCompile(`batch\s+(?:called|named)\s+["']?([a-zA-Z0-9][\w-]*[a-zA-Z0-9])["']?`),
		// "called my-batch" or "named my-batch"
		regexp.MustCompile(`(?:called|named)\s+["']?([a-zA-Z0-9][\w-]*[a-zA-Z0-9])["']?`),
		// "name it my-batch" or "name: my-batch"
		regexp.MustCompile(`name\s+(?:it\s+)?["']?([a-zA-Z0-9][\w-]*[a-zA-Z0-9])["']?`),
		// Explicit quoted name after "batch": 'my-batch' or "my-batch"
		regexp.MustCompile(`batch\s+["']([a-zA-Z0-9][\w-]*[a-zA-Z0-9])["']`),
	}

	// Common words that should never be batch names
	commonWords := map[string]bool{
		"with": true, "these": true, "those": true, "them": true, "it": true,
		"please": true, "yes": true, "and": true, "or": true, "the": true,
		"for": true, "to": true, "from": true, "into": true, "set": true,
		"create": true, "make": true, "batch": true, "a": true, "an": true,
	}

	for _, p := range patterns {
		if match := p.FindStringSubmatch(message); len(match) > 1 {
			name := match[1]
			// Only return if it's not a common word and has reasonable length
			if !commonWords[name] && len(name) >= 2 {
				return name
			}
		}
	}

	return ""
}

// ExtractDestinationOrg extracts a destination organization from a message
func (d *IntentDetector) ExtractDestinationOrg(message string) string {
	message = strings.ToLower(message)

	// Patterns to match destination organization
	patterns := []*regexp.Regexp{
		// "destination organization to kuhlman-labs-org-emu" or "destination org kuhlman-labs-org-emu"
		regexp.MustCompile(`destination\s+(?:org(?:anization)?\s+)?(?:to\s+)?["']?([\w][\w-]*[\w])["']?`),
		// "set the destination to kuhlman-labs-org-emu"
		regexp.MustCompile(`set\s+(?:the\s+)?destination\s+(?:org(?:anization)?\s+)?(?:to\s+)?["']?([\w][\w-]*[\w])["']?`),
		// "migrate to kuhlman-labs-org-emu"
		regexp.MustCompile(`migrate\s+(?:to|into)\s+(?:the\s+)?(?:org(?:anization)?\s+)?["']?([\w][\w-]*[\w])["']?`),
		// "to the kuhlman-labs-org-emu organization"
		regexp.MustCompile(`(?:to|into)\s+(?:the\s+)?["']?([\w][\w-]*[\w])["']?\s+(?:org|organization)`),
	}

	commonWords := map[string]bool{
		"the": true, "org": true, "organization": true, "to": true, "a": true,
		"set": true, "and": true, "batch": true, "create": true, "with": true,
	}

	for _, p := range patterns {
		if match := p.FindStringSubmatch(message); len(match) > 1 {
			dest := match[1]
			if !commonWords[dest] && len(dest) > 2 {
				return dest
			}
		}
	}

	return ""
}
