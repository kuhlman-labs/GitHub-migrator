package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// GetRepositoryDependencies returns all dependencies for a repository
func (h *Handler) GetRepositoryDependencies(w http.ResponseWriter, r *http.Request) {
	fullName := r.PathValue("fullName")
	if fullName == "" {
		WriteError(w, ErrMissingField.WithField("fullName"))
		return
	}
	h.getRepositoryDependencies(w, r, fullName)
}

// getRepositoryDependencies is the internal implementation
func (h *Handler) getRepositoryDependencies(w http.ResponseWriter, r *http.Request, fullName string) {
	decodedFullName, err := url.QueryUnescape(fullName)
	if err != nil {
		h.logger.Warn("Failed to decode repository name", "fullName", fullName, "error", err)
		decodedFullName = fullName
	}

	dependencies, err := h.db.GetRepositoryDependenciesByFullName(r.Context(), decodedFullName)
	if err != nil {
		h.logger.Error("Failed to get repository dependencies",
			"repo", decodedFullName,
			"error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("dependencies"))
		return
	}

	summary := struct {
		Total    int            `json:"total"`
		Local    int            `json:"local"`
		External int            `json:"external"`
		ByType   map[string]int `json:"by_type"`
	}{
		Total:  len(dependencies),
		ByType: make(map[string]int),
	}

	for _, dep := range dependencies {
		if dep.IsLocal {
			summary.Local++
		} else {
			summary.External++
		}
		summary.ByType[dep.DependencyType]++
	}

	response := map[string]interface{}{
		"dependencies": dependencies,
		"summary":      summary,
	}

	h.sendJSON(w, http.StatusOK, response)
}

// GetRepositoryDependents returns repositories that depend on the specified repository
// GET /api/v1/repositories/{fullName}/dependents
func (h *Handler) GetRepositoryDependents(w http.ResponseWriter, r *http.Request) {
	fullName := r.PathValue("fullName")
	if fullName == "" {
		WriteError(w, ErrMissingField.WithField("fullName"))
		return
	}
	h.getRepositoryDependents(w, r, fullName)
}

// getRepositoryDependents is the internal implementation
func (h *Handler) getRepositoryDependents(w http.ResponseWriter, r *http.Request, fullName string) {
	decodedFullName, err := url.QueryUnescape(fullName)
	if err != nil {
		h.logger.Warn("Failed to decode repository name", "fullName", fullName, "error", err)
		decodedFullName = fullName
	}

	// Check if the target repository exists
	repo, err := h.db.GetRepository(r.Context(), decodedFullName)
	if err != nil || repo == nil {
		WriteError(w, ErrRepositoryNotFound)
		return
	}

	dependents, err := h.db.GetDependentRepositories(r.Context(), decodedFullName)
	if err != nil {
		h.logger.Error("Failed to get dependent repositories",
			"repo", decodedFullName,
			"error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("dependent repositories"))
		return
	}

	type DependentRepo struct {
		ID              int64    `json:"id"`
		FullName        string   `json:"full_name"`
		SourceURL       string   `json:"source_url"`
		Status          string   `json:"status"`
		DependencyTypes []string `json:"dependency_types"`
	}

	result := make([]DependentRepo, 0, len(dependents))
	for _, repo := range dependents {
		deps, err := h.db.GetRepositoryDependencies(r.Context(), repo.ID)
		if err != nil {
			h.logger.Warn("Failed to get dependencies for repo", "repo", repo.FullName, "error", err)
			continue
		}

		depTypes := make([]string, 0)
		seen := make(map[string]bool)
		for _, dep := range deps {
			if dep.DependencyFullName == decodedFullName && !seen[dep.DependencyType] {
				depTypes = append(depTypes, dep.DependencyType)
				seen[dep.DependencyType] = true
			}
		}

		// Only include repositories that actually have dependencies on the target
		if len(depTypes) == 0 {
			continue
		}

		result = append(result, DependentRepo{
			ID:              repo.ID,
			FullName:        repo.FullName,
			SourceURL:       repo.SourceURL,
			Status:          repo.Status,
			DependencyTypes: depTypes,
		})
	}

	response := map[string]interface{}{
		"dependents": result,
		"total":      len(result),
		"target":     decodedFullName,
	}

	h.sendJSON(w, http.StatusOK, response)
}

// GetDependencyGraph returns enterprise-wide local dependency graph data
// GET /api/v1/dependencies/graph
func (h *Handler) GetDependencyGraph(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	dependencyTypeFilter := r.URL.Query().Get("dependency_type")
	var dependencyTypes []string
	if dependencyTypeFilter != "" {
		dependencyTypes = strings.Split(dependencyTypeFilter, ",")
	}

	edges, err := h.db.GetAllLocalDependencyPairs(ctx, dependencyTypes)
	if err != nil {
		h.logger.Error("Failed to get dependency graph", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("dependency graph"))
		return
	}

	nodeMap := make(map[string]bool)
	for _, edge := range edges {
		nodeMap[edge.SourceRepo] = true
		nodeMap[edge.TargetRepo] = true
	}

	type GraphNode struct {
		ID              string `json:"id"`
		FullName        string `json:"full_name"`
		Organization    string `json:"organization"`
		Status          string `json:"status"`
		DependsOnCount  int    `json:"depends_on_count"`
		DependedByCount int    `json:"depended_by_count"`
	}

	type GraphEdge struct {
		Source         string `json:"source"`
		Target         string `json:"target"`
		DependencyType string `json:"dependency_type"`
	}

	dependsOnCount := make(map[string]int)
	dependedByCount := make(map[string]int)
	for _, edge := range edges {
		dependsOnCount[edge.SourceRepo]++
		dependedByCount[edge.TargetRepo]++
	}

	nodes := make([]GraphNode, 0, len(nodeMap))
	for fullName := range nodeMap {
		repo, err := h.db.GetRepository(ctx, fullName)
		status := "unknown"
		org := ""
		if err == nil && repo != nil {
			status = repo.Status
			parts := strings.Split(repo.FullName, "/")
			if len(parts) > 0 {
				org = parts[0]
			}
		} else {
			parts := strings.Split(fullName, "/")
			if len(parts) > 0 {
				org = parts[0]
			}
		}

		nodes = append(nodes, GraphNode{
			ID:              fullName,
			FullName:        fullName,
			Organization:    org,
			Status:          status,
			DependsOnCount:  dependsOnCount[fullName],
			DependedByCount: dependedByCount[fullName],
		})
	}

	graphEdges := make([]GraphEdge, 0, len(edges))
	for _, edge := range edges {
		graphEdges = append(graphEdges, GraphEdge{
			Source:         edge.SourceRepo,
			Target:         edge.TargetRepo,
			DependencyType: edge.DependencyType,
		})
	}

	stats := map[string]interface{}{
		"total_repos_with_dependencies": len(nodeMap),
		"total_local_dependencies":      len(edges),
	}

	// Count circular dependencies using normalized pair keys to avoid double-counting
	// A pair (A,B) is normalized by sorting: min(A,B) + "|" + max(A,B)
	circularPairs := make(map[string]bool)
	edgeSet := make(map[string]bool)
	for _, edge := range edges {
		key := edge.SourceRepo + "->" + edge.TargetRepo
		reverseKey := edge.TargetRepo + "->" + edge.SourceRepo
		if edgeSet[reverseKey] {
			// Normalize the pair key to avoid counting the same pair twice
			var pairKey string
			if edge.SourceRepo < edge.TargetRepo {
				pairKey = edge.SourceRepo + "|" + edge.TargetRepo
			} else {
				pairKey = edge.TargetRepo + "|" + edge.SourceRepo
			}
			circularPairs[pairKey] = true
		}
		edgeSet[key] = true
	}
	stats["circular_dependency_count"] = len(circularPairs)

	response := map[string]interface{}{
		"nodes": nodes,
		"edges": graphEdges,
		"stats": stats,
	}

	h.sendJSON(w, http.StatusOK, response)
}

// ExportDependencies exports local dependency data in CSV or JSON format
// GET /api/v1/dependencies/export
func (h *Handler) ExportDependencies(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	format := r.URL.Query().Get("format")
	if format == "" {
		format = formatCSV
	}

	dependencyTypeFilter := r.URL.Query().Get("dependency_type")
	var dependencyTypes []string
	if dependencyTypeFilter != "" {
		dependencyTypes = strings.Split(dependencyTypeFilter, ",")
	}

	edges, err := h.db.GetAllLocalDependencyPairs(ctx, dependencyTypes)
	if err != nil {
		h.logger.Error("Failed to get dependencies for export", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("dependencies"))
		return
	}

	type ExportRow struct {
		Repository     string `json:"repository"`
		DependencyName string `json:"dependency_full_name"`
		Direction      string `json:"direction"`
		DependencyType string `json:"dependency_type"`
		DependencyURL  string `json:"dependency_url"`
	}

	exportData := make([]ExportRow, 0, len(edges)*2)

	for _, edge := range edges {
		exportData = append(exportData, ExportRow{
			Repository:     edge.SourceRepo,
			DependencyName: edge.TargetRepo,
			Direction:      "depends_on",
			DependencyType: edge.DependencyType,
			DependencyURL:  edge.DependencyURL,
		})
	}

	for _, edge := range edges {
		exportData = append(exportData, ExportRow{
			Repository:     edge.TargetRepo,
			DependencyName: edge.SourceRepo,
			Direction:      "depended_by",
			DependencyType: edge.DependencyType,
			DependencyURL:  edge.SourceRepoURL,
		})
	}

	if format == formatJSON {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", "attachment; filename=dependencies.json")
		if err := json.NewEncoder(w).Encode(exportData); err != nil {
			h.logger.Error("Failed to encode JSON", "error", err)
		}
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=dependencies.csv")

	fmt.Fprintln(w, "repository,dependency_full_name,direction,dependency_type,dependency_url")

	for _, row := range exportData {
		fmt.Fprintf(w, "%s,%s,%s,%s,%s\n",
			escapeCSV(row.Repository),
			escapeCSV(row.DependencyName),
			escapeCSV(row.Direction),
			escapeCSV(row.DependencyType),
			escapeCSV(row.DependencyURL),
		)
	}
}

// repoDependencyExportRow represents a row in the repository dependency export
type repoDependencyExportRow struct {
	Repository     string `json:"repository"`
	DependencyName string `json:"dependency_full_name"`
	Direction      string `json:"direction"`
	DependencyType string `json:"dependency_type"`
	DependencyURL  string `json:"dependency_url"`
}

// collectRepoDependsOn collects dependencies that this repo depends on (local only)
func (h *Handler) collectRepoDependsOn(ctx context.Context, repoID int64, repoFullName string) []repoDependencyExportRow {
	rows := make([]repoDependencyExportRow, 0)
	deps, err := h.db.GetRepositoryDependencies(ctx, repoID)
	if err != nil {
		h.logger.Error("Failed to get dependencies", "repo", repoFullName, "error", err)
		return rows
	}

	for _, dep := range deps {
		if dep.IsLocal {
			rows = append(rows, repoDependencyExportRow{
				Repository:     repoFullName,
				DependencyName: dep.DependencyFullName,
				Direction:      "depends_on",
				DependencyType: dep.DependencyType,
				DependencyURL:  dep.DependencyURL,
			})
		}
	}
	return rows
}

// collectRepoDependedBy collects repositories that depend on this repo (local only)
func (h *Handler) collectRepoDependedBy(ctx context.Context, repoFullName string) []repoDependencyExportRow {
	rows := make([]repoDependencyExportRow, 0)
	dependents, err := h.db.GetDependentRepositories(ctx, repoFullName)
	if err != nil {
		h.logger.Error("Failed to get dependents", "repo", repoFullName, "error", err)
		return rows
	}

	for _, dependent := range dependents {
		depDeps, err := h.db.GetRepositoryDependencies(ctx, dependent.ID)
		if err != nil {
			continue
		}
		for _, dep := range depDeps {
			if dep.DependencyFullName == repoFullName && dep.IsLocal {
				rows = append(rows, repoDependencyExportRow{
					Repository:     repoFullName,
					DependencyName: dependent.FullName,
					Direction:      "depended_by",
					DependencyType: dep.DependencyType,
					DependencyURL:  dependent.SourceURL,
				})
			}
		}
	}
	return rows
}

// writeRepoDependencyExport writes the export data in the specified format
func (h *Handler) writeRepoDependencyExport(w http.ResponseWriter, format, repoFullName string, data []repoDependencyExportRow) {
	filename := strings.ReplaceAll(repoFullName, "/", "-")

	if format == formatJSON {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s-dependencies.json", filename))
		if err := json.NewEncoder(w).Encode(data); err != nil {
			h.logger.Error("Failed to encode JSON", "error", err)
		}
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s-dependencies.csv", filename))

	fmt.Fprintln(w, "repository,dependency_full_name,direction,dependency_type,dependency_url")
	for _, row := range data {
		fmt.Fprintf(w, "%s,%s,%s,%s,%s\n",
			escapeCSV(row.Repository),
			escapeCSV(row.DependencyName),
			escapeCSV(row.Direction),
			escapeCSV(row.DependencyType),
			escapeCSV(row.DependencyURL),
		)
	}
}

// ExportRepositoryDependencies exports dependencies for a single repository
// GET /api/v1/repositories/{fullName}/dependencies/export
func (h *Handler) ExportRepositoryDependencies(w http.ResponseWriter, r *http.Request) {
	fullName := r.PathValue("fullName")
	if fullName == "" {
		WriteError(w, ErrMissingField.WithField("fullName"))
		return
	}
	h.exportRepositoryDependencies(w, r, fullName)
}

// exportRepositoryDependencies is the internal implementation
func (h *Handler) exportRepositoryDependencies(w http.ResponseWriter, r *http.Request, fullName string) {
	decodedFullName, err := url.QueryUnescape(fullName)
	if err != nil {
		decodedFullName = fullName
	}

	ctx := r.Context()
	format := r.URL.Query().Get("format")
	if format == "" {
		format = formatCSV
	}

	repo, err := h.db.GetRepository(ctx, decodedFullName)
	if err != nil || repo == nil {
		WriteError(w, ErrRepositoryNotFound)
		return
	}

	exportData := h.collectRepoDependsOn(ctx, repo.ID, decodedFullName)
	exportData = append(exportData, h.collectRepoDependedBy(ctx, decodedFullName)...)

	h.writeRepoDependencyExport(w, format, decodedFullName, exportData)
}
