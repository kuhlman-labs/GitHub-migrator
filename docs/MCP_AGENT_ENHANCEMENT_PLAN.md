# MCP & Agent Enhancement Plan

## Overview

This plan expands the MCP server from 13 tools (focused on mid-lifecycle batch/migration operations) to a comprehensive agent interface covering the **full migration lifecycle**: discovery, pre-migration readiness, execution, tracking, and post-migration validation.

## Current State

**13 existing MCP tools** — all read-from or write-to the local database:

| Tool | Category |
|------|----------|
| `analyze_repositories` | Planning |
| `get_complexity_breakdown` | Planning |
| `check_dependencies` | Planning |
| `find_pilot_candidates` | Planning |
| `plan_waves` | Planning |
| `get_team_repositories` | Planning |
| `create_batch` | Batching |
| `configure_batch` | Batching |
| `schedule_batch` | Batching |
| `get_migration_status` | Tracking |
| `start_migration` | Execution |
| `cancel_migration` | Execution |
| `get_migration_progress` | Tracking |

**Transport**: SSE (legacy) — `mcp-go v0.44.0` already supports Streamable HTTP.

**No skills, no resources, no prompts** defined.

---

## Phase 1: Discovery Tools (Priority: Critical)

Without discovery, an agent cannot initiate any workflow. These tools expose the existing discovery subsystem.

### New MCP Tools

#### 1.1 `list_sources`
- **Purpose**: List all configured migration sources
- **Params**: `active_only` (bool, optional)
- **DB method**: `ListSources()` / `ListActiveSources()`
- **Returns**: Source ID, name, type, base URL, org, enterprise slug, repo count, last sync, active status
- **Notes**: Masks tokens via `source.ToResponse()`

#### 1.2 `get_source`
- **Purpose**: Get details of a single source
- **Params**: `source_id` (int64, required)
- **DB method**: `GetSource()`
- **Returns**: Full source response (masked token)

#### 1.3 `start_discovery`
- **Purpose**: Trigger repository discovery from a configured source
- **Params**:
  - `source_id` (int64, required)
  - `organization` (string, optional — mutually exclusive with `enterprise_slug`)
  - `enterprise_slug` (string, optional)
- **Flow**:
  1. Load source from DB via `GetSource()`
  2. Check for active discovery via `GetActiveDiscoveryProgress()`
  3. Create GitHub/ADO client from source config
  4. Create `Collector` and set source ID
  5. Create `DiscoveryProgress` record
  6. Launch async discovery (same as `discovery_handlers.go`)
  7. Return progress ID
- **Dependencies**: Needs access to `github.NewClient()`, `source.NewProviderFromConfig()`, and `discovery.NewCollector()`. The MCP server will need a factory or the handler's `getCollectorForSource` logic extracted into a shared service.
- **Returns**: `{ progress_id, status: "started", message }`

#### 1.4 `get_discovery_progress`
- **Purpose**: Check status of running/completed discovery
- **Params**: `progress_id` (int64, optional — defaults to latest)
- **DB methods**: `GetActiveDiscoveryProgress()`, `GetLatestDiscoveryProgress()`
- **Returns**: Phase, processed/total repos, processed/total orgs, current org, error count, status

#### 1.5 `cancel_discovery`
- **Purpose**: Cancel a running discovery
- **Params**: none (cancels active discovery)
- **DB method**: `MarkDiscoveryCancelled()`
- **Returns**: `{ success, message }`

### Implementation Notes — Phase 1

**Key challenge**: `start_discovery` requires creating GitHub/ADO clients, which currently lives in the HTTP handler layer (`getCollectorForSource`). Two options:

- **Option A (Recommended)**: Extract a `DiscoveryService` from `discovery_handlers.go` that both the HTTP handlers and MCP server can use. This service encapsulates: source loading → client creation → collector setup → async execution → progress tracking.
- **Option B**: Have the MCP server call the REST API internally. Simpler but adds HTTP round-trip overhead and auth complications.

**Files to create/modify**:
- `internal/discovery/service.go` — new shared DiscoveryService
- `internal/mcp/server.go` — register 5 new tools
- `internal/mcp/handlers_discovery.go` — new file for discovery handlers
- `internal/mcp/types.go` — add discovery input/output types
- `internal/api/handlers/discovery_handlers.go` — refactor to use DiscoveryService

---

## Phase 2: Pre-Migration Readiness Tools (Priority: High)

These tools let an agent assess and prepare for migration — user mappings, team mappings, blockers, and readiness checks.

### New MCP Tools

#### 2.1 `get_migration_readiness`
- **Purpose**: Pre-flight check answering "are we ready to migrate?"
- **Params**: `organization` (string, optional), `batch_id` (int64, optional), `source_id` (int64, optional)
- **Aggregates from**:
  - `GetUsersWithMappingsStats()` — user mapping completeness
  - `GetTeamsWithMappingsStats()` — team mapping completeness
  - `ListRepositories(status=pending)` — repos awaiting migration
  - `ListRepositories(status=remediation_required)` — repos needing remediation
  - Repository blocker counts (oversized commits, blocking files, etc.)
- **Returns**: Readiness score (0-100%), checklist of items with pass/fail/warning, blocking issues, recommended next steps

#### 2.2 `get_user_mapping_stats`
- **Purpose**: Summary of identity mapping progress
- **Params**: `source_org` (string, optional), `source_id` (int64, optional)
- **DB method**: `GetUsersWithMappingsStats()`
- **Returns**: Total users, mapped count, unmapped count, mapping percentage, mannequin stats

#### 2.3 `get_team_mapping_stats`
- **Purpose**: Summary of team mapping progress
- **Params**: `source_org` (string, optional), `source_id` (int64, optional)
- **DB method**: `GetTeamsWithMappingsStats()`
- **Returns**: Total teams, mapped count, unmapped count, skipped count, mapping percentage

#### 2.4 `get_repository_blockers`
- **Purpose**: List repositories with migration blockers requiring remediation
- **Params**: `organization` (string, optional), `limit` (int, default 20)
- **DB method**: `ListRepositories()` with validation filters for blocking flags
- **Returns**: Repos with blocker details (oversized commits, blocking files, oversized repo, long refs), remediation guidance per blocker type

#### 2.5 `suggest_user_mappings`
- **Purpose**: Auto-suggest identity mappings for unmapped users
- **Params**: `destination_org` (string, required), `emu_shortcode` (string, optional)
- **Flow**: Mirrors `user_mapping_handlers.go` FetchMannequins logic — fetch mannequins, match by login/email/name
- **Dependencies**: Needs destination GitHub client. Same service extraction pattern as Phase 1.
- **Returns**: Suggested mappings with confidence scores and match reasons

#### 2.6 `suggest_team_mappings`
- **Purpose**: Auto-suggest team mappings
- **Params**: `destination_org` (string, required)
- **DB method**: `SuggestTeamMappings()`
- **Returns**: Suggested mappings with match reasons

### Implementation Notes — Phase 2

**Files to create/modify**:
- `internal/mcp/handlers_readiness.go` — new file for readiness/mapping handlers
- `internal/mcp/types.go` — add readiness input/output types
- `internal/mcp/server.go` — register 6 new tools

The `get_migration_readiness` tool is the most valuable — it gives an agent a single call to understand overall readiness instead of making 5+ separate queries.

`suggest_user_mappings` has the same challenge as `start_discovery` — it needs a destination GitHub client. The extracted service pattern from Phase 1 should be extended to handle destination client creation as well.

---

## Phase 3: Post-Migration & Analytics Tools (Priority: Medium)

These tools give the agent visibility into outcomes, history, and executive reporting.

### New MCP Tools

#### 3.1 `get_executive_summary`
- **Purpose**: High-level migration progress report
- **Params**: `organization` (string, optional), `source_id` (int64, optional)
- **DB methods**: `GetRepositoryStatsByStatus()`, `GetMigrationVelocity()`, `GetAverageMigrationTime()`, `GetOrganizationStats()`
- **Returns**: Total repos, completion %, success rate, velocity (repos/day, repos/week), estimated completion date, org-level breakdown, risk summary

#### 3.2 `get_migration_history`
- **Purpose**: History of completed/failed migrations
- **Params**: `repository` (string, optional), `limit` (int, default 20), `status_filter` (string, optional — "completed"/"failed")
- **DB methods**: `GetCompletedMigrations()`, `GetMigrationHistory()`
- **Returns**: Migration records with timestamps, duration, outcome, error messages

#### 3.3 `get_migration_logs`
- **Purpose**: Detailed logs for a specific migration (debugging failures)
- **Params**: `repository` (string, required), `level` (string, optional — "error"/"warn"/"info"), `phase` (string, optional)
- **DB method**: `GetMigrationLogs()`
- **Returns**: Log entries with timestamps, level, phase, message

#### 3.4 `get_action_items`
- **Purpose**: What needs attention right now
- **Aggregates from**:
  - Failed migrations needing retry
  - Repos needing remediation
  - Unmapped users/teams
  - Stale discoveries
  - Batches ready but not started
- **Returns**: Prioritized action items with severity and recommended actions

#### 3.5 `rollback_migration`
- **Purpose**: Roll back a completed migration
- **Params**: `repository` (string, required), `reason` (string, optional)
- **DB method**: `RollbackRepository()`
- **Returns**: `{ success, previous_status, new_status, message }`
- **Notes**: Only allowed on `StatusComplete` repos. Guards same as `repository_handler.go`.

### Implementation Notes — Phase 3

**Files to create/modify**:
- `internal/mcp/handlers_analytics.go` — new file for analytics/post-migration handlers
- `internal/mcp/types.go` — add analytics input/output types
- `internal/mcp/server.go` — register 5 new tools

`get_action_items` is the most complex — it aggregates across multiple domains. Consider implementing it as a single DB query or a service method to avoid N+1 queries.

---

## Phase 4: Transport Upgrade (Priority: Medium)

### Upgrade SSE → Streamable HTTP

`mcp-go v0.44.0` already supports Streamable HTTP transport via `server.NewStreamableHTTPServer()`.

**Changes**:
- `internal/mcp/server.go`: Replace `server.NewSSEServer()` with `server.NewStreamableHTTPServer()`
- Update endpoint configuration (single `/mcp` endpoint replaces `/sse` + `/message`)
- Keep SSE as fallback for clients that don't support Streamable HTTP (the library handles this automatically)

**Testing**: Verify with Claude Code MCP client, Cursor, and any other MCP consumers.

---

## Phase 5: MCP Resources (Priority: Low)

Resources provide read-only contextual data to agents without requiring explicit tool calls.

### Proposed Resources

#### 5.1 `migration://dashboard`
- **URI**: `migration://dashboard`
- **Content**: Current migration dashboard state — active discoveries, batch progress, recent completions, alerts
- **Updates**: Dynamic (re-read on each access)

#### 5.2 `migration://readiness/{org}`
- **URI template**: `migration://readiness/{org}`
- **Content**: Pre-flight readiness report for an organization
- **Data**: Same as `get_migration_readiness` tool but as a resource

#### 5.3 `migration://source/{id}/summary`
- **URI template**: `migration://source/{id}/summary`
- **Content**: Source summary with repo counts, last sync, status breakdown

### Implementation Notes — Phase 5

**Files to create/modify**:
- `internal/mcp/resources.go` — new file for resource handlers
- `internal/mcp/server.go` — register resources via `mcpServer.AddResource()` / `mcpServer.AddResourceTemplate()`

---

## Phase 6: Claude Code Skills (Priority: Low)

Skills are prompt-layer orchestrations that chain multiple MCP tool calls into cohesive workflows. These would be defined as Claude Code skill files.

### Proposed Skills

#### 6.1 `/discover`
- **Trigger**: User says "discover repos from source X" or runs `/discover`
- **Flow**:
  1. Call `list_sources` to show available sources
  2. Call `start_discovery` with selected source
  3. Poll `get_discovery_progress` until complete
  4. Call `analyze_repositories` to summarize results
  5. Present summary with next-step recommendations

#### 6.2 `/plan-migration`
- **Trigger**: "Plan migration for org X" or `/plan-migration`
- **Flow**:
  1. Call `analyze_repositories` for the org
  2. Call `get_repository_blockers` to identify issues
  3. Call `check_dependencies` for high-complexity repos
  4. Call `plan_waves` to generate wave plan
  5. Call `find_pilot_candidates` for recommended starting point
  6. Present comprehensive migration plan

#### 6.3 `/migration-status`
- **Trigger**: "How's the migration going?" or `/migration-status`
- **Flow**:
  1. Call `get_executive_summary`
  2. Call `get_action_items`
  3. Call `get_migration_progress` for active batches
  4. Present status dashboard with action items

#### 6.4 `/pre-flight`
- **Trigger**: "Are we ready to migrate batch X?" or `/pre-flight`
- **Flow**:
  1. Call `get_migration_readiness` for the target scope
  2. Call `get_user_mapping_stats` and `get_team_mapping_stats`
  3. Call `get_repository_blockers` for the batch repos
  4. Present go/no-go checklist with blocking issues highlighted

### Implementation Notes — Phase 6

Skills are defined as markdown prompt files in `.claude/skills/` or via the plugin system. Each skill file contains the orchestration instructions that Claude follows when the skill is invoked.

**Files to create**:
- `.claude/skills/discover.md`
- `.claude/skills/plan-migration.md`
- `.claude/skills/migration-status.md`
- `.claude/skills/pre-flight.md`

---

## Implementation Order & Dependencies

```
Phase 1: Discovery Tools ─────────────────────┐
  ├─ Extract DiscoveryService                  │
  ├─ 5 new tools                               │
  └─ Refactor HTTP handlers to share service   │
                                               │
Phase 2: Pre-Migration Readiness ──────────────┤
  ├─ 6 new tools                               │
  └─ Depends on Phase 1 service pattern        │
                                               │
Phase 3: Post-Migration & Analytics ───────────┤
  ├─ 5 new tools                               │
  └─ Independent of Phases 1-2                 │
                                               │
Phase 4: Transport Upgrade ────────────────────┤
  └─ Independent, can be done anytime          │
                                               │
Phase 5: MCP Resources ───────────────────────┤
  └─ Depends on Phases 1-3 (uses same data)    │
                                               │
Phase 6: Claude Code Skills ───────────────────┘
  └─ Depends on Phases 1-3 (orchestrates tools)
```

**Phases 1-3 can be parallelized** — Phase 1 and Phase 3 are independent, and Phase 2 only depends on the service extraction pattern from Phase 1 (for `suggest_user_mappings`).

## Total New Tools: 22

| Phase | Tools | Priority |
|-------|-------|----------|
| Phase 1: Discovery | 5 | Critical |
| Phase 2: Readiness | 6 | High |
| Phase 3: Analytics | 5 | Medium |
| Phase 4: Transport | 0 (infra) | Medium |
| Phase 5: Resources | 3 | Low |
| Phase 6: Skills | 4 | Low |

Combined with the existing 13, this brings the MCP server to **35 tools + 3 resources + 4 skills**.

## Testing Strategy

- **Unit tests**: Each new handler gets a test file mirroring existing `server_test.go` patterns
- **Integration tests**: Test tool registration and JSON serialization
- **End-to-end**: Test full workflows (discovery → analysis → batch → migration) via MCP client
- **Regression**: Ensure existing 13 tools are unaffected by refactoring

## File Structure (Final)

```
internal/mcp/
├── server.go                  # Tool + resource registration, server lifecycle
├── handlers.go                # Existing 13 tool handlers (unchanged)
├── handlers_discovery.go      # Phase 1: Discovery tools
├── handlers_readiness.go      # Phase 2: Pre-migration readiness tools
├── handlers_analytics.go      # Phase 3: Post-migration & analytics tools
├── resources.go               # Phase 5: MCP resources
├── types.go                   # All input/output types
└── server_test.go             # Tests

internal/discovery/
├── service.go                 # New: Shared DiscoveryService (Phase 1)
└── ... (existing files)

.claude/skills/
├── discover.md                # Phase 6
├── plan-migration.md          # Phase 6
├── migration-status.md        # Phase 6
└── pre-flight.md              # Phase 6
```
