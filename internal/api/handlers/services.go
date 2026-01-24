// Package handlers provides HTTP request handlers for the API.
//
// # Service Interfaces
//
// This file defines service interfaces that can be used for dependency injection
// and testing. The DataStore interface composes domain-specific interfaces from
// the storage package, enabling handlers to use either the full DataStore or
// focused interfaces for their specific needs.
//
// # Interface Composition
//
// DataStore is composed of these focused interfaces from storage package:
//   - storage.RepositoryStore: Repository CRUD operations
//   - storage.BatchStore: Batch management operations
//   - storage.MigrationHistoryStore: Migration history and logs
//   - storage.DependencyStore: Repository dependency operations
//   - storage.AnalyticsStore: Statistics and analytics
//   - storage.UserStore: User operations
//   - storage.UserMappingStore: User mapping operations
//   - storage.TeamStore: Team operations
//   - storage.TeamMappingStore: Team mapping operations
//   - storage.ADOStore: Azure DevOps operations
//   - storage.DiscoveryStore: Discovery progress tracking
//   - storage.SetupStore: Setup status operations
//   - storage.DatabaseAccess: Low-level DB access
//
// # Usage in Handlers
//
// Handlers can use the full DataStore interface or request smaller interfaces:
//
//	// Full access (current pattern)
//	type Handler struct {
//	    db DataStore
//	}
//
//	// Focused access (preferred for new handlers)
//	type RepositoryHandler struct {
//	    repos storage.RepositoryStore
//	    deps  storage.DependencyStore
//	}
//
// # Usage in Tests
//
// For testing, implement only the interfaces you need:
//
//	type mockRepoStore struct {
//	    repos map[string]*models.Repository
//	}
//
//	func (m *mockRepoStore) GetRepository(ctx context.Context, fullName string) (*models.Repository, error) {
//	    return m.repos[fullName], nil
//	}
//	// ... implement other RepositoryStore methods
package handlers

import (
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

// DataStore composes all domain-specific storage interfaces.
// This interface enables dependency injection and easier unit testing through mocking.
//
// For new code, prefer using focused interfaces from the storage package directly:
//   - storage.RepositoryStore for repository operations
//   - storage.BatchStore for batch operations
//   - storage.AnalyticsStore for analytics operations
//   - etc.
//
// storage.Database implements this interface and all composed interfaces.
type DataStore interface {
	// Core domain stores
	storage.RepositoryStore
	storage.BatchStore
	storage.MigrationHistoryStore
	storage.DependencyStore
	storage.AnalyticsStore

	// User and team stores
	storage.UserStore
	storage.UserMappingStore
	storage.UserMannequinStore
	storage.TeamStore
	storage.TeamMappingStore

	// Source stores
	storage.SourceStore

	// Platform-specific stores
	storage.ADOStore
	storage.DiscoveryStore
	storage.SetupStore

	// Settings store
	storage.SettingsStore

	// Database access
	storage.DatabaseAccess
}

// Compile-time check that storage.Database implements DataStore
var _ DataStore = (*storage.Database)(nil)

// TimeProvider is an interface for getting current time.
// Useful for testing time-dependent logic.
type TimeProvider interface {
	Now() time.Time
}

// RealTimeProvider implements TimeProvider with real system time.
type RealTimeProvider struct{}

// Now returns the current time.
func (RealTimeProvider) Now() time.Time {
	return time.Now()
}

// MockTimeProvider implements TimeProvider with a fixed time for testing.
type MockTimeProvider struct {
	FixedTime time.Time
}

// Now returns the fixed time.
func (m MockTimeProvider) Now() time.Time {
	return m.FixedTime
}
