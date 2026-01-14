-- +goose Up
-- Core tables

IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'batches')
CREATE TABLE batches (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    name NVARCHAR(MAX) NOT NULL,
    description NVARCHAR(MAX),
    type NVARCHAR(MAX) NOT NULL,
    repository_count INT DEFAULT 0,
    status NVARCHAR(MAX) NOT NULL,
    destination_org NVARCHAR(MAX),
    migration_api NVARCHAR(MAX) NOT NULL DEFAULT 'GEI',
    exclude_releases BIT DEFAULT 0,
    exclude_attachments BIT DEFAULT 0,
    scheduled_at DATETIME2,
    started_at DATETIME2,
    completed_at DATETIME2,
    last_dry_run_at DATETIME2,
    last_migration_attempt_at DATETIME2,
    dry_run_started_at DATETIME2,
    dry_run_completed_at DATETIME2,
    dry_run_duration_seconds INT,
    created_at DATETIME2 NOT NULL DEFAULT GETUTCDATE()
);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_batches_status')
CREATE INDEX idx_batches_status ON batches(status);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_batches_type')
CREATE INDEX idx_batches_type ON batches(type);

IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'sources')
CREATE TABLE sources (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    name NVARCHAR(450) NOT NULL UNIQUE,
    type NVARCHAR(MAX) NOT NULL,
    base_url NVARCHAR(MAX) NOT NULL,
    token NVARCHAR(MAX) NOT NULL,
    organization NVARCHAR(MAX),
    enterprise_slug NVARCHAR(MAX),
    app_id BIGINT,
    app_private_key NVARCHAR(MAX),
    app_installation_id BIGINT,
    is_active BIT DEFAULT 1,
    repository_count INT DEFAULT 0,
    last_sync_at DATETIME2,
    created_at DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    updated_at DATETIME2 NOT NULL DEFAULT GETUTCDATE()
);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_sources_type')
CREATE INDEX idx_sources_type ON sources(type);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_sources_is_active')
CREATE INDEX idx_sources_is_active ON sources(is_active);

-- Core repository table (narrow - optimized for list queries)
IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'repositories')
CREATE TABLE repositories (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    full_name NVARCHAR(450) NOT NULL UNIQUE,
    source NVARCHAR(MAX) NOT NULL,
    source_url NVARCHAR(MAX) NOT NULL,
    source_id BIGINT REFERENCES sources(id),
    status NVARCHAR(MAX) NOT NULL,
    batch_id BIGINT REFERENCES batches(id),
    priority INT DEFAULT 0,
    visibility NVARCHAR(MAX),
    is_archived BIT DEFAULT 0,
    is_fork BIT DEFAULT 0,
    destination_url NVARCHAR(MAX),
    destination_full_name NVARCHAR(MAX),
    source_migration_id BIGINT,
    is_source_locked BIT DEFAULT 0,
    exclude_releases BIT DEFAULT 0,
    exclude_attachments BIT DEFAULT 0,
    exclude_metadata BIT DEFAULT 0,
    exclude_git_data BIT DEFAULT 0,
    exclude_owner_projects BIT DEFAULT 0,
    discovered_at DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    updated_at DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    migrated_at DATETIME2,
    last_discovery_at DATETIME2,
    last_dry_run_at DATETIME2
);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_repos_status')
CREATE INDEX idx_repos_status ON repositories(status);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_repos_batch_id')
CREATE INDEX idx_repos_batch_id ON repositories(batch_id);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_repos_source_id')
CREATE INDEX idx_repos_source_id ON repositories(source_id);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_repos_source_status')
CREATE INDEX idx_repos_source_status ON repositories(source_id, status);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_repos_batch_status')
CREATE INDEX idx_repos_batch_status ON repositories(batch_id, status);

-- Git properties sub-table
IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'repository_git_properties')
CREATE TABLE repository_git_properties (
    repository_id BIGINT PRIMARY KEY REFERENCES repositories(id) ON DELETE CASCADE,
    total_size BIGINT,
    default_branch NVARCHAR(MAX),
    branch_count INT DEFAULT 0,
    commit_count INT DEFAULT 0,
    commits_last_12_weeks INT DEFAULT 0,
    has_lfs BIT DEFAULT 0,
    has_submodules BIT DEFAULT 0,
    has_large_files BIT DEFAULT 0,
    large_file_count INT DEFAULT 0,
    largest_file NVARCHAR(MAX),
    largest_file_size BIGINT,
    largest_commit NVARCHAR(MAX),
    largest_commit_size BIGINT,
    last_commit_sha NVARCHAR(MAX),
    last_commit_date DATETIME2
);

-- Features sub-table
IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'repository_features')
CREATE TABLE repository_features (
    repository_id BIGINT PRIMARY KEY REFERENCES repositories(id) ON DELETE CASCADE,
    has_wiki BIT DEFAULT 0,
    has_pages BIT DEFAULT 0,
    has_discussions BIT DEFAULT 0,
    has_actions BIT DEFAULT 0,
    has_projects BIT DEFAULT 0,
    has_packages BIT DEFAULT 0,
    has_rulesets BIT DEFAULT 0,
    branch_protections INT DEFAULT 0,
    tag_protection_count INT DEFAULT 0,
    environment_count INT DEFAULT 0,
    secret_count INT DEFAULT 0,
    variable_count INT DEFAULT 0,
    webhook_count INT DEFAULT 0,
    workflow_count INT DEFAULT 0,
    has_code_scanning BIT DEFAULT 0,
    has_dependabot BIT DEFAULT 0,
    has_secret_scanning BIT DEFAULT 0,
    has_codeowners BIT DEFAULT 0,
    codeowners_content NVARCHAR(MAX),
    codeowners_teams NVARCHAR(MAX),
    codeowners_users NVARCHAR(MAX),
    has_self_hosted_runners BIT DEFAULT 0,
    collaborator_count INT DEFAULT 0,
    installed_apps_count INT DEFAULT 0,
    installed_apps NVARCHAR(MAX),
    release_count INT DEFAULT 0,
    has_release_assets BIT DEFAULT 0,
    contributor_count INT DEFAULT 0,
    top_contributors NVARCHAR(MAX),
    issue_count INT DEFAULT 0,
    pull_request_count INT DEFAULT 0,
    tag_count INT DEFAULT 0,
    open_issue_count INT DEFAULT 0,
    open_pr_count INT DEFAULT 0
);

-- ADO properties sub-table (only populated for ADO repos)
IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'repository_ado_properties')
CREATE TABLE repository_ado_properties (
    repository_id BIGINT PRIMARY KEY REFERENCES repositories(id) ON DELETE CASCADE,
    project NVARCHAR(MAX),
    is_git BIT DEFAULT 1,
    has_boards BIT DEFAULT 0,
    has_pipelines BIT DEFAULT 0,
    has_ghas BIT DEFAULT 0,
    pipeline_count INT DEFAULT 0,
    yaml_pipeline_count INT DEFAULT 0,
    classic_pipeline_count INT DEFAULT 0,
    pipeline_run_count INT DEFAULT 0,
    has_service_connections BIT DEFAULT 0,
    has_variable_groups BIT DEFAULT 0,
    has_self_hosted_agents BIT DEFAULT 0,
    pull_request_count INT DEFAULT 0,
    open_pr_count INT DEFAULT 0,
    pr_with_linked_work_items INT DEFAULT 0,
    pr_with_attachments INT DEFAULT 0,
    work_item_count INT DEFAULT 0,
    work_item_linked_count INT DEFAULT 0,
    active_work_item_count INT DEFAULT 0,
    work_item_types NVARCHAR(MAX),
    branch_policy_count INT DEFAULT 0,
    branch_policy_types NVARCHAR(MAX),
    required_reviewer_count INT DEFAULT 0,
    build_validation_policies INT DEFAULT 0,
    has_wiki BIT DEFAULT 0,
    wiki_page_count INT DEFAULT 0,
    test_plan_count INT DEFAULT 0,
    test_case_count INT DEFAULT 0,
    package_feed_count INT DEFAULT 0,
    has_artifacts BIT DEFAULT 0,
    service_hook_count INT DEFAULT 0,
    installed_extensions NVARCHAR(MAX)
);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_ado_project')
CREATE INDEX idx_ado_project ON repository_ado_properties(project);

-- Validation sub-table
IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'repository_validation')
CREATE TABLE repository_validation (
    repository_id BIGINT PRIMARY KEY REFERENCES repositories(id) ON DELETE CASCADE,
    validation_status NVARCHAR(MAX),
    validation_details NVARCHAR(MAX),
    destination_data NVARCHAR(MAX),
    has_oversized_commits BIT DEFAULT 0,
    oversized_commit_details NVARCHAR(MAX),
    has_long_refs BIT DEFAULT 0,
    long_ref_details NVARCHAR(MAX),
    has_blocking_files BIT DEFAULT 0,
    blocking_file_details NVARCHAR(MAX),
    has_large_file_warnings BIT DEFAULT 0,
    large_file_warning_details NVARCHAR(MAX),
    has_oversized_repository BIT DEFAULT 0,
    oversized_repository_details NVARCHAR(MAX),
    estimated_metadata_size BIGINT,
    metadata_size_details NVARCHAR(MAX),
    complexity_score INT,
    complexity_breakdown NVARCHAR(MAX)
);

-- Migration history and logs
IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'migration_history')
CREATE TABLE migration_history (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    repository_id BIGINT NOT NULL REFERENCES repositories(id),
    status NVARCHAR(MAX) NOT NULL,
    phase NVARCHAR(MAX) NOT NULL,
    message NVARCHAR(MAX),
    error_message NVARCHAR(MAX),
    started_at DATETIME2 NOT NULL,
    completed_at DATETIME2,
    duration_seconds INT
);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_migration_history_repo')
CREATE INDEX idx_migration_history_repo ON migration_history(repository_id);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_migration_history_status')
CREATE INDEX idx_migration_history_status ON migration_history(status);

IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'migration_logs')
CREATE TABLE migration_logs (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    repository_id BIGINT NOT NULL REFERENCES repositories(id),
    history_id BIGINT REFERENCES migration_history(id),
    level NVARCHAR(MAX) NOT NULL,
    phase NVARCHAR(MAX) NOT NULL,
    operation NVARCHAR(MAX) NOT NULL,
    message NVARCHAR(MAX) NOT NULL,
    details NVARCHAR(MAX),
    initiated_by NVARCHAR(MAX),
    timestamp DATETIME2 NOT NULL DEFAULT GETUTCDATE()
);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_migration_logs_repo')
CREATE INDEX idx_migration_logs_repo ON migration_logs(repository_id);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_migration_logs_level')
CREATE INDEX idx_migration_logs_level ON migration_logs(level);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_migration_logs_timestamp')
CREATE INDEX idx_migration_logs_timestamp ON migration_logs(timestamp);

-- Repository dependencies
IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'repository_dependencies')
CREATE TABLE repository_dependencies (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    repository_id BIGINT NOT NULL REFERENCES repositories(id),
    dependency_full_name NVARCHAR(MAX) NOT NULL,
    dependency_type NVARCHAR(MAX) NOT NULL,
    dependency_url NVARCHAR(MAX) NOT NULL,
    is_local BIT DEFAULT 0,
    discovered_at DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    metadata NVARCHAR(MAX)
);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_repo_deps_repo')
CREATE INDEX idx_repo_deps_repo ON repository_dependencies(repository_id);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_repo_deps_dep_name')
CREATE INDEX idx_repo_deps_dep_name ON repository_dependencies(dependency_full_name);

-- ADO Projects
IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'ado_projects')
CREATE TABLE ado_projects (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    source_id BIGINT REFERENCES sources(id),
    organization NVARCHAR(450) NOT NULL,
    name NVARCHAR(450) NOT NULL,
    description NVARCHAR(MAX),
    repository_count INT DEFAULT 0,
    state NVARCHAR(MAX),
    visibility NVARCHAR(MAX),
    discovered_at DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    updated_at DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    UNIQUE(organization, name)
);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_ado_projects_org')
CREATE INDEX idx_ado_projects_org ON ado_projects(organization);

-- GitHub Teams
IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'github_teams')
CREATE TABLE github_teams (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    source_id BIGINT REFERENCES sources(id),
    organization NVARCHAR(450) NOT NULL,
    slug NVARCHAR(450) NOT NULL,
    name NVARCHAR(MAX) NOT NULL,
    description NVARCHAR(MAX),
    privacy NVARCHAR(MAX) NOT NULL DEFAULT 'closed',
    discovered_at DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    updated_at DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    UNIQUE(organization, slug)
);

IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'github_team_repositories')
CREATE TABLE github_team_repositories (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    team_id BIGINT NOT NULL REFERENCES github_teams(id),
    repository_id BIGINT NOT NULL REFERENCES repositories(id),
    permission NVARCHAR(MAX) NOT NULL DEFAULT 'pull',
    discovered_at DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    UNIQUE(team_id, repository_id)
);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_team_repos_repo')
CREATE INDEX idx_team_repos_repo ON github_team_repositories(repository_id);

IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'github_team_members')
CREATE TABLE github_team_members (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    team_id BIGINT NOT NULL REFERENCES github_teams(id),
    login NVARCHAR(MAX) NOT NULL,
    role NVARCHAR(MAX) NOT NULL,
    discovered_at DATETIME2 NOT NULL DEFAULT GETUTCDATE()
);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_team_members_team')
CREATE INDEX idx_team_members_team ON github_team_members(team_id);

-- GitHub Users
IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'github_users')
CREATE TABLE github_users (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    source_id BIGINT REFERENCES sources(id),
    login NVARCHAR(450) NOT NULL UNIQUE,
    name NVARCHAR(MAX),
    email NVARCHAR(MAX),
    avatar_url NVARCHAR(MAX),
    source_instance NVARCHAR(MAX) NOT NULL,
    commit_count INT DEFAULT 0,
    issue_count INT DEFAULT 0,
    pr_count INT DEFAULT 0,
    comment_count INT DEFAULT 0,
    repository_count INT DEFAULT 0,
    discovered_at DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    updated_at DATETIME2 NOT NULL DEFAULT GETUTCDATE()
);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_github_users_email')
CREATE INDEX idx_github_users_email ON github_users(email);

IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'user_org_memberships')
CREATE TABLE user_org_memberships (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    user_login NVARCHAR(450) NOT NULL,
    organization NVARCHAR(450) NOT NULL,
    role NVARCHAR(MAX) NOT NULL DEFAULT 'member',
    discovered_at DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    UNIQUE(user_login, organization)
);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_user_org_org')
CREATE INDEX idx_user_org_org ON user_org_memberships(organization);

-- User and Team Mappings
IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'user_mappings')
CREATE TABLE user_mappings (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    source_id BIGINT REFERENCES sources(id),
    source_login NVARCHAR(450) NOT NULL UNIQUE,
    source_email NVARCHAR(MAX),
    source_name NVARCHAR(MAX),
    source_org NVARCHAR(MAX),
    destination_login NVARCHAR(MAX),
    destination_email NVARCHAR(MAX),
    mapping_status NVARCHAR(MAX) NOT NULL DEFAULT 'unmapped',
    mannequin_id NVARCHAR(MAX),
    mannequin_login NVARCHAR(MAX),
    mannequin_org NVARCHAR(MAX),
    reclaim_status NVARCHAR(MAX),
    reclaim_error NVARCHAR(MAX),
    match_confidence INT,
    match_reason NVARCHAR(MAX),
    created_at DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    updated_at DATETIME2 NOT NULL DEFAULT GETUTCDATE()
);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_user_mappings_status')
CREATE INDEX idx_user_mappings_status ON user_mappings(mapping_status);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_user_mappings_source_org')
CREATE INDEX idx_user_mappings_source_org ON user_mappings(source_org);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_user_mappings_dest')
CREATE INDEX idx_user_mappings_dest ON user_mappings(destination_login);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_user_mappings_mannequin_org')
CREATE INDEX idx_user_mappings_mannequin_org ON user_mappings(mannequin_org);

IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'team_mappings')
CREATE TABLE team_mappings (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    source_id BIGINT REFERENCES sources(id),
    source_org NVARCHAR(450) NOT NULL,
    source_team_slug NVARCHAR(450) NOT NULL,
    source_team_name NVARCHAR(MAX),
    destination_org NVARCHAR(MAX),
    destination_team_slug NVARCHAR(MAX),
    destination_team_name NVARCHAR(MAX),
    mapping_status NVARCHAR(MAX) NOT NULL DEFAULT 'unmapped',
    auto_created BIT DEFAULT 0,
    migration_status NVARCHAR(MAX) DEFAULT 'pending',
    migrated_at DATETIME2,
    error_message NVARCHAR(MAX),
    repos_synced INT DEFAULT 0,
    total_source_repos INT DEFAULT 0,
    repos_eligible INT DEFAULT 0,
    team_created_in_dest BIT DEFAULT 0,
    last_synced_at DATETIME2,
    created_at DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    updated_at DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    UNIQUE(source_org, source_team_slug)
);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_team_mappings_status')
CREATE INDEX idx_team_mappings_status ON team_mappings(mapping_status);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_team_mappings_mig_status')
CREATE INDEX idx_team_mappings_mig_status ON team_mappings(migration_status);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_team_mappings_dest_org')
CREATE INDEX idx_team_mappings_dest_org ON team_mappings(destination_org);

-- Discovery Progress
IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'discovery_progress')
CREATE TABLE discovery_progress (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    discovery_type NVARCHAR(MAX) NOT NULL,
    target NVARCHAR(MAX) NOT NULL,
    status NVARCHAR(MAX) NOT NULL DEFAULT 'in_progress',
    started_at DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    completed_at DATETIME2,
    total_orgs INT DEFAULT 0,
    processed_orgs INT DEFAULT 0,
    current_org NVARCHAR(MAX),
    total_repos INT DEFAULT 0,
    processed_repos INT DEFAULT 0,
    phase NVARCHAR(MAX) DEFAULT 'listing_repos',
    error_count INT DEFAULT 0,
    last_error NVARCHAR(MAX)
);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_discovery_status')
CREATE INDEX idx_discovery_status ON discovery_progress(status);

-- Setup Status
IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'setup_status')
CREATE TABLE setup_status (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    setup_completed BIT NOT NULL DEFAULT 0,
    completed_at DATETIME2,
    updated_at DATETIME2 NOT NULL DEFAULT GETUTCDATE()
);

-- Settings
IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'settings')
CREATE TABLE settings (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    -- Destination GitHub configuration
    destination_base_url NVARCHAR(MAX) NOT NULL DEFAULT 'https://api.github.com',
    destination_token NVARCHAR(MAX),
    destination_app_id BIGINT,
    destination_app_private_key NVARCHAR(MAX),
    destination_app_installation_id BIGINT,
    destination_enterprise_slug NVARCHAR(MAX),
    -- Migration settings
    migration_workers INT NOT NULL DEFAULT 5,
    migration_poll_interval_seconds INT NOT NULL DEFAULT 30,
    migration_dest_repo_exists_action NVARCHAR(50) NOT NULL DEFAULT 'fail',
    migration_visibility_public NVARCHAR(50) NOT NULL DEFAULT 'private',
    migration_visibility_internal NVARCHAR(50) NOT NULL DEFAULT 'private',
    -- Auth settings
    auth_enabled BIT NOT NULL DEFAULT 0,
    auth_github_oauth_client_id NVARCHAR(MAX),
    auth_github_oauth_client_secret NVARCHAR(MAX),
    auth_session_secret NVARCHAR(MAX),
    auth_session_duration_hours INT NOT NULL DEFAULT 24,
    auth_callback_url NVARCHAR(MAX),
    auth_frontend_url NVARCHAR(MAX) NOT NULL DEFAULT 'http://localhost:3000',
    -- Authorization rules
    auth_migration_admin_teams NVARCHAR(MAX),
    auth_allow_org_admin_migrations BIT NOT NULL DEFAULT 0,
    auth_allow_enterprise_admin_migrations BIT NOT NULL DEFAULT 0,
    auth_enable_self_service BIT NOT NULL DEFAULT 0,
    -- Timestamps
    created_at DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    updated_at DATETIME2 NOT NULL DEFAULT GETUTCDATE()
);

-- Authorization Rules
IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'authorization_rules')
CREATE TABLE authorization_rules (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    name NVARCHAR(450) NOT NULL UNIQUE,
    rule_type NVARCHAR(MAX) NOT NULL,
    pattern NVARCHAR(MAX) NOT NULL,
    role NVARCHAR(MAX) NOT NULL,
    is_active BIT DEFAULT 1,
    created_at DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    updated_at DATETIME2 NOT NULL DEFAULT GETUTCDATE()
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

