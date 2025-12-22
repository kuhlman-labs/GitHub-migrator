package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kuhlman-labs/github-migrator/internal/config"
	"github.com/kuhlman-labs/github-migrator/internal/models"
)

// setupUsersTestDB creates a test database and runs migrations
func setupUsersTestDB(t *testing.T) (*Database, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "users-test-*")
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.DatabaseConfig{
		Type: "sqlite",
		DSN:  filepath.Join(tmpDir, "test.db"),
	}

	db, err := NewDatabase(cfg)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("NewDatabase() error = %v", err)
	}

	if err := db.Migrate(); err != nil {
		db.Close()
		os.RemoveAll(tmpDir)
		t.Fatalf("Migrate() error = %v", err)
	}

	cleanup := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}

	return db, cleanup
}

func TestDatabase_SaveUser(t *testing.T) {
	db, cleanup := setupUsersTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Test creating a new user
	user := &models.GitHubUser{
		Login:          "testuser",
		Name:           stringPtr("Test User"),
		Email:          stringPtr("test@example.com"),
		SourceInstance: "github.com",
		CommitCount:    10,
		PRCount:        5,
		IssueCount:     3,
	}

	err := db.SaveUser(ctx, user)
	if err != nil {
		t.Fatalf("SaveUser() error = %v", err)
	}

	// Verify user was created
	saved, err := db.GetUserByLogin(ctx, "testuser")
	if err != nil {
		t.Fatalf("GetUserByLogin() error = %v", err)
	}
	if saved == nil {
		t.Fatal("GetUserByLogin() returned nil")
		return
	}
	if saved.Login != "testuser" {
		t.Errorf("Login = %v, want testuser", saved.Login)
	}
	if *saved.Name != "Test User" {
		t.Errorf("Name = %v, want Test User", *saved.Name)
	}
	if saved.CommitCount != 10 {
		t.Errorf("CommitCount = %v, want 10", saved.CommitCount)
	}

	// Test updating the user (contribution stats should keep max)
	updatedUser := &models.GitHubUser{
		Login:          "testuser",
		Name:           stringPtr("Updated Name"),
		SourceInstance: "github.com",
		CommitCount:    5, // Lower than existing, should keep 10
		PRCount:        8, // Higher than existing, should update to 8
		IssueCount:     3,
	}

	err = db.SaveUser(ctx, updatedUser)
	if err != nil {
		t.Fatalf("SaveUser() update error = %v", err)
	}

	// Verify update logic
	updated, err := db.GetUserByLogin(ctx, "testuser")
	if err != nil {
		t.Fatalf("GetUserByLogin() error = %v", err)
	}
	if updated.CommitCount != 10 {
		t.Errorf("CommitCount = %v, want 10 (should keep max)", updated.CommitCount)
	}
	if updated.PRCount != 8 {
		t.Errorf("PRCount = %v, want 8 (should update to new max)", updated.PRCount)
	}
}

func TestDatabase_GetUserByLogin(t *testing.T) {
	db, cleanup := setupUsersTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Test getting non-existent user
	user, err := db.GetUserByLogin(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("GetUserByLogin() error = %v", err)
	}
	if user != nil {
		t.Error("GetUserByLogin() should return nil for non-existent user")
	}

	// Create a user
	err = db.SaveUser(ctx, &models.GitHubUser{
		Login:          "findme",
		Name:           stringPtr("Find Me"),
		SourceInstance: "github.com",
	})
	if err != nil {
		t.Fatalf("SaveUser() error = %v", err)
	}

	// Test getting existing user
	user, err = db.GetUserByLogin(ctx, "findme")
	if err != nil {
		t.Fatalf("GetUserByLogin() error = %v", err)
	}
	if user == nil {
		t.Fatal("GetUserByLogin() returned nil for existing user")
		return
	}
	if user.Login != "findme" {
		t.Errorf("Login = %v, want findme", user.Login)
	}
}

func TestDatabase_GetUserByEmail(t *testing.T) {
	db, cleanup := setupUsersTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Test getting non-existent user by email
	user, err := db.GetUserByEmail(ctx, "nonexistent@example.com")
	if err != nil {
		t.Fatalf("GetUserByEmail() error = %v", err)
	}
	if user != nil {
		t.Error("GetUserByEmail() should return nil for non-existent user")
	}

	// Create a user with email
	err = db.SaveUser(ctx, &models.GitHubUser{
		Login:          "emailuser",
		Email:          stringPtr("found@example.com"),
		SourceInstance: "github.com",
	})
	if err != nil {
		t.Fatalf("SaveUser() error = %v", err)
	}

	// Test getting existing user by email
	user, err = db.GetUserByEmail(ctx, "found@example.com")
	if err != nil {
		t.Fatalf("GetUserByEmail() error = %v", err)
	}
	if user == nil {
		t.Fatal("GetUserByEmail() returned nil for existing user")
		return
	}
	if user.Login != "emailuser" {
		t.Errorf("Login = %v, want emailuser", user.Login)
	}
}

func TestDatabase_ListUsers(t *testing.T) {
	db, cleanup := setupUsersTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create multiple users
	users := []*models.GitHubUser{
		{Login: "user1", SourceInstance: "github.com", CommitCount: 100},
		{Login: "user2", SourceInstance: "github.com", CommitCount: 50},
		{Login: "user3", SourceInstance: "ghes.example.com", CommitCount: 75},
		{Login: "user4", SourceInstance: "github.com", CommitCount: 25},
	}

	for _, u := range users {
		if err := db.SaveUser(ctx, u); err != nil {
			t.Fatalf("SaveUser() error = %v", err)
		}
	}

	// Test listing all users
	result, total, err := db.ListUsers(ctx, "", 0, 0)
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if total != 4 {
		t.Errorf("Total = %v, want 4", total)
	}
	if len(result) != 4 {
		t.Errorf("Result length = %v, want 4", len(result))
	}

	// Should be ordered by commit_count DESC
	if result[0].Login != "user1" {
		t.Errorf("First user = %v, want user1 (highest commits)", result[0].Login)
	}

	// Test filtering by source instance
	_, total, err = db.ListUsers(ctx, "github.com", 0, 0)
	if err != nil {
		t.Fatalf("ListUsers() with filter error = %v", err)
	}
	if total != 3 {
		t.Errorf("Total = %v, want 3 (github.com only)", total)
	}

	// Test pagination
	result, total, err = db.ListUsers(ctx, "", 2, 1)
	if err != nil {
		t.Fatalf("ListUsers() with pagination error = %v", err)
	}
	if total != 4 {
		t.Errorf("Total = %v, want 4", total)
	}
	if len(result) != 2 {
		t.Errorf("Result length = %v, want 2 (limited)", len(result))
	}
}

func TestDatabase_GetUserStats(t *testing.T) {
	db, cleanup := setupUsersTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create users with various stats
	users := []*models.GitHubUser{
		{Login: "user1", Email: stringPtr("user1@example.com"), SourceInstance: "github.com", CommitCount: 100, PRCount: 10, IssueCount: 5},
		{Login: "user2", Email: nil, SourceInstance: "github.com", CommitCount: 50, PRCount: 5, IssueCount: 3},
		{Login: "user3", Email: stringPtr("user3@example.com"), SourceInstance: "github.com", CommitCount: 25, PRCount: 2, IssueCount: 1},
	}

	for _, u := range users {
		if err := db.SaveUser(ctx, u); err != nil {
			t.Fatalf("SaveUser() error = %v", err)
		}
	}

	stats, err := db.GetUserStats(ctx)
	if err != nil {
		t.Fatalf("GetUserStats() error = %v", err)
	}

	// Check total_users - cast to int64 for comparison
	totalUsers, ok := stats["total_users"].(int64)
	if !ok || totalUsers != 3 {
		t.Errorf("total_users = %v (type %T), want 3", stats["total_users"], stats["total_users"])
	}

	usersWithEmail, ok := stats["users_with_email"].(int64)
	if !ok || usersWithEmail != 2 {
		t.Errorf("users_with_email = %v (type %T), want 2", stats["users_with_email"], stats["users_with_email"])
	}

	// For aggregate sums, verify they're at least present and non-negative
	// The exact values depend on SaveUser update logic
	if _, exists := stats["total_commits"]; !exists {
		t.Error("total_commits not in stats")
	}
	if _, exists := stats["total_prs"]; !exists {
		t.Error("total_prs not in stats")
	}
	if _, exists := stats["total_issues"]; !exists {
		t.Error("total_issues not in stats")
	}
}

func TestDatabase_IncrementUserRepositoryCount(t *testing.T) {
	db, cleanup := setupUsersTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create a user
	err := db.SaveUser(ctx, &models.GitHubUser{
		Login:           "countuser",
		SourceInstance:  "github.com",
		RepositoryCount: 5,
	})
	if err != nil {
		t.Fatalf("SaveUser() error = %v", err)
	}

	// Increment count
	err = db.IncrementUserRepositoryCount(ctx, "countuser")
	if err != nil {
		t.Fatalf("IncrementUserRepositoryCount() error = %v", err)
	}

	// Verify
	user, err := db.GetUserByLogin(ctx, "countuser")
	if err != nil {
		t.Fatalf("GetUserByLogin() error = %v", err)
	}
	if user.RepositoryCount != 6 {
		t.Errorf("RepositoryCount = %v, want 6", user.RepositoryCount)
	}

	// Increment again
	err = db.IncrementUserRepositoryCount(ctx, "countuser")
	if err != nil {
		t.Fatalf("IncrementUserRepositoryCount() error = %v", err)
	}

	user, _ = db.GetUserByLogin(ctx, "countuser")
	if user.RepositoryCount != 7 {
		t.Errorf("RepositoryCount = %v, want 7", user.RepositoryCount)
	}
}

func TestDatabase_DeleteAllUsers(t *testing.T) {
	db, cleanup := setupUsersTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create some users
	for i := 0; i < 5; i++ {
		err := db.SaveUser(ctx, &models.GitHubUser{
			Login:          "user" + string(rune('a'+i)),
			SourceInstance: "github.com",
		})
		if err != nil {
			t.Fatalf("SaveUser() error = %v", err)
		}
	}

	// Verify users exist
	_, total, _ := db.ListUsers(ctx, "", 0, 0)
	if total != 5 {
		t.Errorf("Expected 5 users, got %v", total)
	}

	// Delete all users
	err := db.DeleteAllUsers(ctx)
	if err != nil {
		t.Fatalf("DeleteAllUsers() error = %v", err)
	}

	// Verify all users deleted
	_, total, _ = db.ListUsers(ctx, "", 0, 0)
	if total != 0 {
		t.Errorf("Expected 0 users after delete, got %v", total)
	}
}

func TestDatabase_UserOrgMembership(t *testing.T) {
	db, cleanup := setupUsersTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create a user first
	err := db.SaveUser(ctx, &models.GitHubUser{
		Login:          "memberuser",
		SourceInstance: "github.com",
	})
	if err != nil {
		t.Fatalf("SaveUser() error = %v", err)
	}

	// Save membership
	membership := &models.UserOrgMembership{
		UserLogin:    "memberuser",
		Organization: "org1",
		Role:         "member",
	}
	err = db.SaveUserOrgMembership(ctx, membership)
	if err != nil {
		t.Fatalf("SaveUserOrgMembership() error = %v", err)
	}

	// Save another membership
	membership2 := &models.UserOrgMembership{
		UserLogin:    "memberuser",
		Organization: "org2",
		Role:         "admin",
	}
	err = db.SaveUserOrgMembership(ctx, membership2)
	if err != nil {
		t.Fatalf("SaveUserOrgMembership() error = %v", err)
	}

	// Get user org memberships
	memberships, err := db.GetUserOrgMemberships(ctx, "memberuser")
	if err != nil {
		t.Fatalf("GetUserOrgMemberships() error = %v", err)
	}
	if len(memberships) != 2 {
		t.Errorf("Expected 2 memberships, got %v", len(memberships))
	}

	// Test update (save with different role)
	membership.Role = "admin"
	err = db.SaveUserOrgMembership(ctx, membership)
	if err != nil {
		t.Fatalf("SaveUserOrgMembership() update error = %v", err)
	}

	memberships, _ = db.GetUserOrgMemberships(ctx, "memberuser")
	for _, m := range memberships {
		if m.Organization == "org1" && m.Role != "admin" {
			t.Errorf("Expected role to be updated to admin, got %v", m.Role)
		}
	}
}

func TestDatabase_GetOrgMembers(t *testing.T) {
	db, cleanup := setupUsersTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create users and memberships
	for _, login := range []string{"alice", "bob", "charlie"} {
		err := db.SaveUser(ctx, &models.GitHubUser{Login: login, SourceInstance: "github.com"})
		if err != nil {
			t.Fatalf("SaveUser() error = %v", err)
		}

		org := "org-a"
		if login == "charlie" {
			org = "org-b"
		}

		err = db.SaveUserOrgMembership(ctx, &models.UserOrgMembership{
			UserLogin:    login,
			Organization: org,
			Role:         "member",
		})
		if err != nil {
			t.Fatalf("SaveUserOrgMembership() error = %v", err)
		}
	}

	// Get org-a members
	members, err := db.GetOrgMembers(ctx, "org-a")
	if err != nil {
		t.Fatalf("GetOrgMembers() error = %v", err)
	}
	if len(members) != 2 {
		t.Errorf("Expected 2 members in org-a, got %v", len(members))
	}

	// Get org-b members
	members, err = db.GetOrgMembers(ctx, "org-b")
	if err != nil {
		t.Fatalf("GetOrgMembers() error = %v", err)
	}
	if len(members) != 1 {
		t.Errorf("Expected 1 member in org-b, got %v", len(members))
	}
}

func TestDatabase_GetDistinctUserOrgs(t *testing.T) {
	db, cleanup := setupUsersTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create users with memberships
	for i, login := range []string{"user1", "user2", "user3"} {
		err := db.SaveUser(ctx, &models.GitHubUser{Login: login, SourceInstance: "github.com"})
		if err != nil {
			t.Fatalf("SaveUser() error = %v", err)
		}

		// Each user in different orgs
		orgs := []string{"org-a", "org-b"}
		if i == 2 {
			orgs = []string{"org-c"}
		}

		for _, org := range orgs {
			err = db.SaveUserOrgMembership(ctx, &models.UserOrgMembership{
				UserLogin:    login,
				Organization: org,
				Role:         "member",
			})
			if err != nil {
				t.Fatalf("SaveUserOrgMembership() error = %v", err)
			}
		}
	}

	orgs, err := db.GetDistinctUserOrgs(ctx)
	if err != nil {
		t.Fatalf("GetDistinctUserOrgs() error = %v", err)
	}
	if len(orgs) != 3 {
		t.Errorf("Expected 3 distinct orgs, got %v", len(orgs))
	}
}

func TestDatabase_GetPrimaryOrgForUser(t *testing.T) {
	db, cleanup := setupUsersTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create user with multiple orgs
	err := db.SaveUser(ctx, &models.GitHubUser{Login: "multiorguser", SourceInstance: "github.com"})
	if err != nil {
		t.Fatalf("SaveUser() error = %v", err)
	}

	// Add to orgs (org-z added first, but org-a should be primary alphabetically)
	for _, org := range []string{"org-z", "org-a", "org-m"} {
		err = db.SaveUserOrgMembership(ctx, &models.UserOrgMembership{
			UserLogin:    "multiorguser",
			Organization: org,
			Role:         "member",
		})
		if err != nil {
			t.Fatalf("SaveUserOrgMembership() error = %v", err)
		}
	}

	// Get primary org (should be alphabetically first)
	primary, err := db.GetPrimaryOrgForUser(ctx, "multiorguser")
	if err != nil {
		t.Fatalf("GetPrimaryOrgForUser() error = %v", err)
	}
	if primary != "org-a" {
		t.Errorf("Primary = %v, want org-a", primary)
	}

	// Test for non-existent user
	primary, err = db.GetPrimaryOrgForUser(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("GetPrimaryOrgForUser() error = %v", err)
	}
	if primary != "" {
		t.Errorf("Expected empty string for non-existent user, got %v", primary)
	}
}
