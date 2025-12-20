// Package ado provides utilities for working with Azure DevOps URLs and resources.
package ado

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// Host constants for Azure DevOps
const (
	HostDevAzure       = "dev.azure.com"
	HostSSHDevAzure    = "ssh.dev.azure.com"
	SuffixVisualStudio = ".visualstudio.com"

	// GitPathSegment is the URL path segment that identifies a Git repository in ADO URLs
	GitPathSegment = "_git"
)

// ParsedURL contains extracted components from an Azure DevOps URL.
type ParsedURL struct {
	Organization string // ADO organization name
	Project      string // ADO project name
	Repository   string // Repository name
	Host         string // The host (dev.azure.com, ssh.dev.azure.com, or org.visualstudio.com)
	IsSSH        bool   // True if this is an SSH URL
}

// Precompiled regex patterns for efficiency
var (
	// Pattern for dev.azure.com: https://dev.azure.com/{org}/{project}/_git/{repo}
	// Also matches: https://user@dev.azure.com/{org}/{project}/_git/{repo}
	devAzurePattern = regexp.MustCompile(`(?:https://)?(?:[^@]+@)?dev\.azure\.com/([^/]+)/([^/]+)/_git/([^/"'\s]+)`)

	// Pattern for visualstudio.com: https://{org}.visualstudio.com/{project}/_git/{repo}
	vstsPattern = regexp.MustCompile(`https://([^.]+)\.visualstudio\.com/([^/]+)/_git/([^/"'\s]+)`)

	// Pattern for SSH: git@ssh.dev.azure.com:v3/{org}/{project}/{repo}
	sshPattern = regexp.MustCompile(`git@ssh\.dev\.azure\.com:v3/([^/]+)/([^/]+)/([^/"'\s]+)`)
)

// IsADOURL checks if a URL is an Azure DevOps URL.
// It recognizes dev.azure.com, visualstudio.com, and ssh.dev.azure.com URLs.
func IsADOURL(gitURL string) bool {
	return strings.Contains(gitURL, "dev.azure.com") ||
		strings.Contains(gitURL, "visualstudio.com") ||
		strings.Contains(gitURL, "ssh.dev.azure.com")
}

// IsADOHost checks if a host is an Azure DevOps host.
func IsADOHost(host string) bool {
	return host == HostDevAzure ||
		host == HostSSHDevAzure ||
		strings.HasSuffix(host, SuffixVisualStudio)
}

// Parse extracts organization, project, and repository from an Azure DevOps Git URL.
// It supports:
//   - HTTPS: https://dev.azure.com/{org}/{project}/_git/{repo}
//   - HTTPS with auth: https://user@dev.azure.com/{org}/{project}/_git/{repo}
//   - Legacy VSTS: https://{org}.visualstudio.com/{project}/_git/{repo}
//   - SSH: git@ssh.dev.azure.com:v3/{org}/{project}/{repo}
//
// Returns nil if the URL is not a valid ADO Git URL.
func Parse(gitURL string) *ParsedURL {
	// Try dev.azure.com pattern
	if matches := devAzurePattern.FindStringSubmatch(gitURL); len(matches) >= 4 {
		return &ParsedURL{
			Organization: matches[1],
			Project:      matches[2],
			Repository:   strings.TrimSuffix(matches[3], ".git"),
			Host:         HostDevAzure,
			IsSSH:        false,
		}
	}

	// Try visualstudio.com pattern
	if matches := vstsPattern.FindStringSubmatch(gitURL); len(matches) >= 4 {
		return &ParsedURL{
			Organization: matches[1],
			Project:      matches[2],
			Repository:   strings.TrimSuffix(matches[3], ".git"),
			Host:         matches[1] + SuffixVisualStudio,
			IsSSH:        false,
		}
	}

	// Try SSH pattern
	if matches := sshPattern.FindStringSubmatch(gitURL); len(matches) >= 4 {
		return &ParsedURL{
			Organization: matches[1],
			Project:      matches[2],
			Repository:   strings.TrimSuffix(matches[3], ".git"),
			Host:         HostSSHDevAzure,
			IsSSH:        true,
		}
	}

	return nil
}

// ParseStrict extracts components from an ADO URL and returns an error if invalid.
// This is useful for validation scenarios where you need to know why parsing failed.
func ParseStrict(gitURL string) (*ParsedURL, error) {
	if gitURL == "" {
		return nil, fmt.Errorf("empty URL")
	}

	if !IsADOURL(gitURL) {
		return nil, fmt.Errorf("not an Azure DevOps URL")
	}

	result := Parse(gitURL)
	if result == nil {
		return nil, fmt.Errorf("invalid ADO URL format - expected: https://dev.azure.com/{org}/{project}/_git/{repo}")
	}

	return result, nil
}

// ParseFromSourceURL parses an ADO source URL (like those stored in repository.SourceURL).
// This is specifically for URLs in the format: https://dev.azure.com/{org}/{project}/_git/{repo}
// Returns an error with a descriptive message if parsing fails.
func ParseFromSourceURL(sourceURL string) (*ParsedURL, error) {
	if sourceURL == "" {
		return nil, fmt.Errorf("source URL is empty")
	}

	parsedURL, err := url.Parse(sourceURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse source URL: %w", err)
	}

	pathParts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
	if len(pathParts) < 4 || pathParts[2] != GitPathSegment {
		return nil, fmt.Errorf("invalid ADO URL format - expected: https://dev.azure.com/{org}/{project}/%s/{repo}", GitPathSegment)
	}

	return &ParsedURL{
		Organization: pathParts[0],
		Project:      pathParts[1],
		Repository:   pathParts[3],
		Host:         parsedURL.Host,
		IsSSH:        false,
	}, nil
}

// FullSlug returns the full repository identifier in the format org/project/repo.
func (p *ParsedURL) FullSlug() string {
	return p.Organization + "/" + p.Project + "/" + p.Repository
}

// ProjectSlug returns the project identifier in the format org/project.
func (p *ParsedURL) ProjectSlug() string {
	return p.Organization + "/" + p.Project
}
