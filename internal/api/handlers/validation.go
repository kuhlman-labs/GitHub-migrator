package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/github"
)

// ValidateGitHubConnection validates a GitHub connection and returns a ValidationResponse
func ValidateGitHubConnection(ctx context.Context, baseURL, token string, logger *slog.Logger) ValidationResponse {
	response := ValidationResponse{Details: make(map[string]any)}

	// Create temporary GitHub client
	client, err := github.NewClient(github.ClientConfig{
		BaseURL: baseURL,
		Token:   token,
		Timeout: 10 * time.Second,
		Logger:  logger,
	})
	if err != nil {
		response.Valid = false
		response.Error = fmt.Sprintf("Failed to create GitHub client: %v", err)
		return response
	}

	// Test connection by getting authenticated user
	user, _, err := client.REST().Users.Get(ctx, "")
	if err != nil {
		response.Valid = false
		response.Error = fmt.Sprintf("Failed to connect to GitHub: %v", err)
		return response
	}

	response.Valid = true
	response.Details["username"] = user.GetLogin()
	response.Details["user_id"] = user.GetID()
	response.Details["base_url"] = baseURL

	// Check rate limit
	rateLimits, _, err := client.REST().RateLimit.Get(ctx)
	if err == nil && rateLimits != nil && rateLimits.Core != nil {
		response.Details["rate_limit_remaining"] = rateLimits.Core.Remaining
		response.Details["rate_limit_total"] = rateLimits.Core.Limit

		// Warn if rate limit is low
		if rateLimits.Core.Remaining < 100 {
			response.Warnings = append(response.Warnings,
				fmt.Sprintf("Low rate limit remaining: %d/%d", rateLimits.Core.Remaining, rateLimits.Core.Limit))
		}
	}

	return response
}
