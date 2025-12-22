package models

import (
	"testing"
	"time"
)

func TestADOProject_TableName(t *testing.T) {
	project := ADOProject{}
	tableName := project.TableName()

	if tableName != "ado_projects" {
		t.Errorf("TableName() = %q, want %q", tableName, "ado_projects")
	}
}

func TestADOProject_Structure(t *testing.T) {
	now := time.Now()
	desc := "Test project description"

	project := ADOProject{
		ID:              1,
		Organization:    "test-org",
		Name:            "test-project",
		Description:     &desc,
		RepositoryCount: 10,
		State:           "wellFormed",
		Visibility:      "private",
		DiscoveredAt:    now,
		UpdatedAt:       now,
	}

	if project.ID != 1 {
		t.Errorf("ID = %d, want %d", project.ID, 1)
	}
	if project.Organization != "test-org" {
		t.Errorf("Organization = %q, want %q", project.Organization, "test-org")
	}
	if project.Name != "test-project" {
		t.Errorf("Name = %q, want %q", project.Name, "test-project")
	}
	if project.Description == nil || *project.Description != desc {
		t.Error("Description not set correctly")
	}
	if project.RepositoryCount != 10 {
		t.Errorf("RepositoryCount = %d, want %d", project.RepositoryCount, 10)
	}
	if project.State != "wellFormed" {
		t.Errorf("State = %q, want %q", project.State, "wellFormed")
	}
	if project.Visibility != "private" {
		t.Errorf("Visibility = %q, want %q", project.Visibility, "private")
	}
}

func TestADOProject_NilDescription(t *testing.T) {
	project := ADOProject{
		Organization: "org",
		Name:         "project",
		Description:  nil,
	}

	if project.Description != nil {
		t.Error("Description should be nil")
	}
}

func TestADOProject_States(t *testing.T) {
	// Test various ADO project states
	states := []string{
		"wellFormed",
		"createPending",
		"deleting",
		"new",
		"unchanged",
	}

	for _, state := range states {
		t.Run(state, func(t *testing.T) {
			project := ADOProject{State: state}
			if project.State != state {
				t.Errorf("State = %q, want %q", project.State, state)
			}
		})
	}
}

func TestADOProject_Visibilities(t *testing.T) {
	// Test various ADO project visibilities
	visibilities := []string{
		"private",
		"public",
	}

	for _, vis := range visibilities {
		t.Run(vis, func(t *testing.T) {
			project := ADOProject{Visibility: vis}
			if project.Visibility != vis {
				t.Errorf("Visibility = %q, want %q", project.Visibility, vis)
			}
		})
	}
}

func TestADOProject_ZeroValues(t *testing.T) {
	// Test zero-valued project
	var project ADOProject

	if project.ID != 0 {
		t.Errorf("ID zero value = %d, want 0", project.ID)
	}
	if project.Organization != "" {
		t.Errorf("Organization zero value = %q, want empty", project.Organization)
	}
	if project.Name != "" {
		t.Errorf("Name zero value = %q, want empty", project.Name)
	}
	if project.RepositoryCount != 0 {
		t.Errorf("RepositoryCount zero value = %d, want 0", project.RepositoryCount)
	}
}
