-- +goose Up
-- Core tables

CREATE TABLE IF NOT EXISTS batches (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    type TEXT NOT NULL,
    repository_count INTEGER DEFAULT 0,
    status TEXT NOT NULL,
    destination_org TEXT,
    migration_api TEXT NOT NULL DEFAULT 'GEI',
    exclude_releases BOOLEAN DEFAULT FALSE,
    exclude_attachments BOOLEAN DEFAULT FALSE,
    scheduled_at TIMESTAMP,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    last_dry_run_at TIMESTAMP,
    last_migration_attempt_at TIMESTAMP,
    dry_run_started_at TIMESTAMP,
    dry_run_completed_at TIMESTAMP,
    dry_run_duration_seconds INTEGER,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_batches_status ON batches(status);
CREATE INDEX idx_batches_type ON batches(type);

CREATE TABLE IF NOT EXISTS sources (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    type TEXT NOT NULL,
    base_url TEXT NOT NULL,
    token TEXT NOT NULL,
    organization TEXT,
    enterprise_slug TEXT,
    app_id BIGINT,
    app_private_key TEXT,
    app_installation_id BIGINT,
    is_active BOOLEAN DEFAULT TRUE,
    repository_count INTEGER DEFAULT 0,
    last_sync_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_sources_type ON sources(type);
CREATE INDEX idx_sources_is_active ON sources(is_active);

-- Core repository table (narrow - optimized for list queries)
CREATE TABLE IF NOT EXISTS repositories (
    id BIGSERIAL PRIMARY KEY,
    full_name TEXT NOT NULL UNIQUE,
    source TEXT NOT NULL,
    source_url TEXT NOT NULL,
    source_id BIGINT REFERENCES sources(id),
    status TEXT NOT NULL,
    batch_id BIGINT REFERENCES batches(id),
    priority INTEGER DEFAULT 0,
    visibility TEXT,
    is_archived BOOLEAN DEFAULT FALSE,
    is_fork BOOLEAN DEFAULT FALSE,
    destination_url TEXT,
    destination_full_name TEXT,
    source_migration_id BIGINT,
    is_source_locked BOOLEAN DEFAULT FALSE,
    exclude_releases BOOLEAN DEFAULT FALSE,
    exclude_attachments BOOLEAN DEFAULT FALSE,
    exclude_metadata BOOLEAN DEFAULT FALSE,
    exclude_git_data BOOLEAN DEFAULT FALSE,
    exclude_owner_projects BOOLEAN DEFAULT FALSE,
    discovered_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    migrated_at TIMESTAMP,
    last_discovery_at TIMESTAMP,
    last_dry_run_at TIMESTAMP
);

CREATE INDEX idx_repos_status ON repositories(status);
CREATE INDEX idx_repos_batch_id ON repositories(batch_id);
CREATE INDEX idx_repos_source_id ON repositories(source_id);
CREATE INDEX idx_repos_source_status ON repositories(source_id, status);
CREATE INDEX idx_repos_batch_status ON repositories(batch_id, status);

-- Git properties sub-table
CREATE TABLE IF NOT EXISTS repository_git_properties (
    repository_id BIGINT PRIMARY KEY REFERENCES repositories(id) ON DELETE CASCADE,
    total_size BIGINT,
    default_branch TEXT,
    branch_count INTEGER DEFAULT 0,
    commit_count INTEGER DEFAULT 0,
    commits_last_12_weeks INTEGER DEFAULT 0,
    has_lfs BOOLEAN DEFAULT FALSE,
    has_submodules BOOLEAN DEFAULT FALSE,
    has_large_files BOOLEAN DEFAULT FALSE,
    large_file_count INTEGER DEFAULT 0,
    largest_file TEXT,
    largest_file_size BIGINT,
    largest_commit TEXT,
    largest_commit_size BIGINT,
    last_commit_sha TEXT,
    last_commit_date TIMESTAMP
);

-- Features sub-table
CREATE TABLE IF NOT EXISTS repository_features (
    repository_id BIGINT PRIMARY KEY REFERENCES repositories(id) ON DELETE CASCADE,
    has_wiki BOOLEAN DEFAULT FALSE,
    has_pages BOOLEAN DEFAULT FALSE,
    has_discussions BOOLEAN DEFAULT FALSE,
    has_actions BOOLEAN DEFAULT FALSE,
    has_projects BOOLEAN DEFAULT FALSE,
    has_packages BOOLEAN DEFAULT FALSE,
    has_rulesets BOOLEAN DEFAULT FALSE,
    branch_protections INTEGER DEFAULT 0,
    tag_protection_count INTEGER DEFAULT 0,
    environment_count INTEGER DEFAULT 0,
    secret_count INTEGER DEFAULT 0,
    variable_count INTEGER DEFAULT 0,
    webhook_count INTEGER DEFAULT 0,
    workflow_count INTEGER DEFAULT 0,
    has_code_scanning BOOLEAN DEFAULT FALSE,
    has_dependabot BOOLEAN DEFAULT FALSE,
    has_secret_scanning BOOLEAN DEFAULT FALSE,
    has_codeowners BOOLEAN DEFAULT FALSE,
    codeowners_content TEXT,
    codeowners_teams TEXT,
    codeowners_users TEXT,
    has_self_hosted_runners BOOLEAN DEFAULT FALSE,
    collaborator_count INTEGER DEFAULT 0,
    installed_apps_count INTEGER DEFAULT 0,
    installed_apps TEXT,
    release_count INTEGER DEFAULT 0,
    has_release_assets BOOLEAN DEFAULT FALSE,
    contributor_count INTEGER DEFAULT 0,
    top_contributors TEXT,
    issue_count INTEGER DEFAULT 0,
    pull_request_count INTEGER DEFAULT 0,
    tag_count INTEGER DEFAULT 0,
    open_issue_count INTEGER DEFAULT 0,
    open_pr_count INTEGER DEFAULT 0
);

-- ADO properties sub-table (only populated for ADO repos)
CREATE TABLE IF NOT EXISTS repository_ado_properties (
    repository_id BIGINT PRIMARY KEY REFERENCES repositories(id) ON DELETE CASCADE,
    project TEXT,
    is_git BOOLEAN DEFAULT TRUE,
    has_boards BOOLEAN DEFAULT FALSE,
    has_pipelines BOOLEAN DEFAULT FALSE,
    has_ghas BOOLEAN DEFAULT FALSE,
    pipeline_count INTEGER DEFAULT 0,
    yaml_pipeline_count INTEGER DEFAULT 0,
    classic_pipeline_count INTEGER DEFAULT 0,
    pipeline_run_count INTEGER DEFAULT 0,
    has_service_connections BOOLEAN DEFAULT FALSE,
    has_variable_groups BOOLEAN DEFAULT FALSE,
    has_self_hosted_agents BOOLEAN DEFAULT FALSE,
    pull_request_count INTEGER DEFAULT 0,
    open_pr_count INTEGER DEFAULT 0,
    pr_with_linked_work_items INTEGER DEFAULT 0,
    pr_with_attachments INTEGER DEFAULT 0,
    work_item_count INTEGER DEFAULT 0,
    work_item_linked_count INTEGER DEFAULT 0,
    active_work_item_count INTEGER DEFAULT 0,
    work_item_types TEXT,
    branch_policy_count INTEGER DEFAULT 0,
    branch_policy_types TEXT,
    required_reviewer_count INTEGER DEFAULT 0,
    build_validation_policies INTEGER DEFAULT 0,
    has_wiki BOOLEAN DEFAULT FALSE,
    wiki_page_count INTEGER DEFAULT 0,
    test_plan_count INTEGER DEFAULT 0,
    test_case_count INTEGER DEFAULT 0,
    package_feed_count INTEGER DEFAULT 0,
    has_artifacts BOOLEAN DEFAULT FALSE,
    service_hook_count INTEGER DEFAULT 0,
    installed_extensions TEXT
);

CREATE INDEX idx_ado_project ON repository_ado_properties(project);

-- Validation sub-table
CREATE TABLE IF NOT EXISTS repository_validation (
    repository_id BIGINT PRIMARY KEY REFERENCES repositories(id) ON DELETE CASCADE,
    validation_status TEXT,
    validation_details TEXT,
    destination_data TEXT,
    has_oversized_commits BOOLEAN DEFAULT FALSE,
    oversized_commit_details TEXT,
    has_long_refs BOOLEAN DEFAULT FALSE,
    long_ref_details TEXT,
    has_blocking_files BOOLEAN DEFAULT FALSE,
    blocking_file_details TEXT,
    has_large_file_warnings BOOLEAN DEFAULT FALSE,
    large_file_warning_details TEXT,
    has_oversized_repository BOOLEAN DEFAULT FALSE,
    oversized_repository_details TEXT,
    estimated_metadata_size BIGINT,
    metadata_size_details TEXT,
    complexity_score INTEGER,
    complexity_breakdown TEXT
);

-- Migration history and logs
CREATE TABLE IF NOT EXISTS migration_history (
    id BIGSERIAL PRIMARY KEY,
    repository_id BIGINT NOT NULL REFERENCES repositories(id),
    status TEXT NOT NULL,
    phase TEXT NOT NULL,
    message TEXT,
    error_message TEXT,
    started_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP,
    duration_seconds INTEGER
);

CREATE INDEX idx_migration_history_repo ON migration_history(repository_id);
CREATE INDEX idx_migration_history_status ON migration_history(status);

CREATE TABLE IF NOT EXISTS migration_logs (
    id BIGSERIAL PRIMARY KEY,
    repository_id BIGINT NOT NULL REFERENCES repositories(id),
    history_id BIGINT REFERENCES migration_history(id),
    level TEXT NOT NULL,
    phase TEXT NOT NULL,
    operation TEXT NOT NULL,
    message TEXT NOT NULL,
    details TEXT,
    initiated_by TEXT,
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_migration_logs_repo ON migration_logs(repository_id);
CREATE INDEX idx_migration_logs_level ON migration_logs(level);
CREATE INDEX idx_migration_logs_timestamp ON migration_logs(timestamp);

-- Repository dependencies
CREATE TABLE IF NOT EXISTS repository_dependencies (
    id BIGSERIAL PRIMARY KEY,
    repository_id BIGINT NOT NULL REFERENCES repositories(id),
    dependency_full_name TEXT NOT NULL,
    dependency_type TEXT NOT NULL,
    dependency_url TEXT NOT NULL,
    is_local BOOLEAN DEFAULT FALSE,
    discovered_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    metadata TEXT
);

CREATE INDEX idx_repo_deps_repo ON repository_dependencies(repository_id);
CREATE INDEX idx_repo_deps_dep_name ON repository_dependencies(dependency_full_name);

-- ADO Projects
CREATE TABLE IF NOT EXISTS ado_projects (
    id BIGSERIAL PRIMARY KEY,
    source_id BIGINT REFERENCES sources(id),
    organization TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    repository_count INTEGER DEFAULT 0,
    state TEXT,
    visibility TEXT,
    discovered_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(organization, name)
);

CREATE INDEX idx_ado_projects_org ON ado_projects(organization);

-- GitHub Teams
CREATE TABLE IF NOT EXISTS github_teams (
    id BIGSERIAL PRIMARY KEY,
    source_id BIGINT REFERENCES sources(id),
    organization TEXT NOT NULL,
    slug TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    privacy TEXT NOT NULL DEFAULT 'closed',
    discovered_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(organization, slug)
);

CREATE TABLE IF NOT EXISTS github_team_repositories (
    id BIGSERIAL PRIMARY KEY,
    team_id BIGINT NOT NULL REFERENCES github_teams(id),
    repository_id BIGINT NOT NULL REFERENCES repositories(id),
    permission TEXT NOT NULL DEFAULT 'pull',
    discovered_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(team_id, repository_id)
);

CREATE INDEX idx_team_repos_repo ON github_team_repositories(repository_id);

CREATE TABLE IF NOT EXISTS github_team_members (
    id BIGSERIAL PRIMARY KEY,
    team_id BIGINT NOT NULL REFERENCES github_teams(id),
    login TEXT NOT NULL,
    role TEXT NOT NULL,
    discovered_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_team_members_team ON github_team_members(team_id);

-- GitHub Users
CREATE TABLE IF NOT EXISTS github_users (
    id BIGSERIAL PRIMARY KEY,
    source_id BIGINT REFERENCES sources(id),
    login TEXT NOT NULL UNIQUE,
    name TEXT,
    email TEXT,
    avatar_url TEXT,
    source_instance TEXT NOT NULL,
    commit_count INTEGER DEFAULT 0,
    issue_count INTEGER DEFAULT 0,
    pr_count INTEGER DEFAULT 0,
    comment_count INTEGER DEFAULT 0,
    repository_count INTEGER DEFAULT 0,
    discovered_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_github_users_email ON github_users(email);

CREATE TABLE IF NOT EXISTS user_org_memberships (
    id BIGSERIAL PRIMARY KEY,
    user_login TEXT NOT NULL,
    organization TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'member',
    discovered_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_login, organization)
);

CREATE INDEX idx_user_org_org ON user_org_memberships(organization);

-- User and Team Mappings
CREATE TABLE IF NOT EXISTS user_mappings (
    id BIGSERIAL PRIMARY KEY,
    source_id BIGINT REFERENCES sources(id),
    source_login TEXT NOT NULL UNIQUE,
    source_email TEXT,
    source_name TEXT,
    source_org TEXT,
    destination_login TEXT,
    destination_email TEXT,
    mapping_status TEXT NOT NULL DEFAULT 'unmapped',
    mannequin_id TEXT,
    mannequin_login TEXT,
    mannequin_org TEXT,
    reclaim_status TEXT,
    reclaim_error TEXT,
    match_confidence INTEGER,
    match_reason TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_user_mappings_status ON user_mappings(mapping_status);
CREATE INDEX idx_user_mappings_source_org ON user_mappings(source_org);
CREATE INDEX idx_user_mappings_dest ON user_mappings(destination_login);
CREATE INDEX idx_user_mappings_mannequin_org ON user_mappings(mannequin_org);

CREATE TABLE IF NOT EXISTS team_mappings (
    id BIGSERIAL PRIMARY KEY,
    source_id BIGINT REFERENCES sources(id),
    source_org TEXT NOT NULL,
    source_team_slug TEXT NOT NULL,
    source_team_name TEXT,
    destination_org TEXT,
    destination_team_slug TEXT,
    destination_team_name TEXT,
    mapping_status TEXT NOT NULL DEFAULT 'unmapped',
    auto_created BOOLEAN DEFAULT FALSE,
    migration_status TEXT DEFAULT 'pending',
    migrated_at TIMESTAMP,
    error_message TEXT,
    repos_synced INTEGER DEFAULT 0,
    total_source_repos INTEGER DEFAULT 0,
    repos_eligible INTEGER DEFAULT 0,
    team_created_in_dest BOOLEAN DEFAULT FALSE,
    last_synced_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_org, source_team_slug)
);

CREATE INDEX idx_team_mappings_status ON team_mappings(mapping_status);
CREATE INDEX idx_team_mappings_mig_status ON team_mappings(migration_status);
CREATE INDEX idx_team_mappings_dest_org ON team_mappings(destination_org);

-- Discovery Progress
CREATE TABLE IF NOT EXISTS discovery_progress (
    id BIGSERIAL PRIMARY KEY,
    discovery_type TEXT NOT NULL,
    target TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'in_progress',
    started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    total_orgs INTEGER DEFAULT 0,
    processed_orgs INTEGER DEFAULT 0,
    current_org TEXT,
    total_repos INTEGER DEFAULT 0,
    processed_repos INTEGER DEFAULT 0,
    phase TEXT DEFAULT 'listing_repos',
    error_count INTEGER DEFAULT 0,
    last_error TEXT
);

CREATE INDEX idx_discovery_status ON discovery_progress(status);

-- Setup Status
CREATE TABLE IF NOT EXISTS setup_status (
    id BIGSERIAL PRIMARY KEY,
    setup_completed BOOLEAN NOT NULL DEFAULT FALSE,
    completed_at TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Settings
CREATE TABLE IF NOT EXISTS settings (
    id BIGSERIAL PRIMARY KEY,
    -- Destination GitHub configuration
    destination_base_url TEXT NOT NULL DEFAULT 'https://api.github.com',
    destination_token TEXT,
    destination_app_id BIGINT,
    destination_app_private_key TEXT,
    destination_app_installation_id BIGINT,
    destination_enterprise_slug TEXT,
    -- Migration settings
    migration_workers INTEGER NOT NULL DEFAULT 5,
    migration_poll_interval_seconds INTEGER NOT NULL DEFAULT 30,
    migration_dest_repo_exists_action TEXT NOT NULL DEFAULT 'fail',
    migration_visibility_public TEXT NOT NULL DEFAULT 'private',
    migration_visibility_internal TEXT NOT NULL DEFAULT 'private',
    -- Auth settings
    auth_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    auth_github_oauth_client_id TEXT,
    auth_github_oauth_client_secret TEXT,
    auth_session_secret TEXT,
    auth_session_duration_hours INTEGER NOT NULL DEFAULT 24,
    auth_callback_url TEXT,
    auth_frontend_url TEXT NOT NULL DEFAULT 'http://localhost:3000',
    -- Authorization rules
    auth_migration_admin_teams TEXT,
    auth_allow_org_admin_migrations BOOLEAN NOT NULL DEFAULT FALSE,
    auth_allow_enterprise_admin_migrations BOOLEAN NOT NULL DEFAULT FALSE,
    auth_enable_self_service BOOLEAN NOT NULL DEFAULT FALSE,
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Authorization Rules
CREATE TABLE IF NOT EXISTS authorization_rules (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    rule_type TEXT NOT NULL,
    pattern TEXT NOT NULL,
    role TEXT NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE IF EXISTS authorization_rules;
DROP TABLE IF EXISTS settings;
DROP TABLE IF EXISTS setup_status;
DROP TABLE IF EXISTS discovery_progress;
DROP TABLE IF EXISTS team_mappings;
DROP TABLE IF EXISTS user_mappings;
DROP TABLE IF EXISTS user_org_memberships;
DROP TABLE IF EXISTS github_users;
DROP TABLE IF EXISTS github_team_members;
DROP TABLE IF EXISTS github_team_repositories;
DROP TABLE IF EXISTS github_teams;
DROP TABLE IF EXISTS ado_projects;
DROP TABLE IF EXISTS repository_dependencies;
DROP TABLE IF EXISTS migration_logs;
DROP TABLE IF EXISTS migration_history;
DROP TABLE IF EXISTS repository_validation;
DROP TABLE IF EXISTS repository_ado_properties;
DROP TABLE IF EXISTS repository_features;
DROP TABLE IF EXISTS repository_git_properties;
DROP TABLE IF EXISTS repositories;
DROP TABLE IF EXISTS sources;
DROP TABLE IF EXISTS batches;

