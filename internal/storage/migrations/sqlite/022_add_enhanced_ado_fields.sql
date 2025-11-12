-- Add enhanced Azure DevOps specific fields to repositories table

-- Enhanced Pipeline Data
ALTER TABLE repositories ADD COLUMN ado_pipeline_count INTEGER DEFAULT 0;
ALTER TABLE repositories ADD COLUMN ado_yaml_pipeline_count INTEGER DEFAULT 0;
ALTER TABLE repositories ADD COLUMN ado_classic_pipeline_count INTEGER DEFAULT 0;
ALTER TABLE repositories ADD COLUMN ado_pipeline_run_count INTEGER DEFAULT 0;
ALTER TABLE repositories ADD COLUMN ado_has_service_connections BOOLEAN DEFAULT FALSE;
ALTER TABLE repositories ADD COLUMN ado_has_variable_groups BOOLEAN DEFAULT FALSE;
ALTER TABLE repositories ADD COLUMN ado_has_self_hosted_agents BOOLEAN DEFAULT FALSE;

-- Enhanced Work Item Data
ALTER TABLE repositories ADD COLUMN ado_work_item_linked_count INTEGER DEFAULT 0;
ALTER TABLE repositories ADD COLUMN ado_active_work_item_count INTEGER DEFAULT 0;
ALTER TABLE repositories ADD COLUMN ado_work_item_types TEXT;

-- Pull Request Details
ALTER TABLE repositories ADD COLUMN ado_open_pr_count INTEGER DEFAULT 0;
ALTER TABLE repositories ADD COLUMN ado_pr_with_linked_work_items INTEGER DEFAULT 0;
ALTER TABLE repositories ADD COLUMN ado_pr_with_attachments INTEGER DEFAULT 0;

-- Enhanced Branch Policy Data
ALTER TABLE repositories ADD COLUMN ado_branch_policy_types TEXT;
ALTER TABLE repositories ADD COLUMN ado_required_reviewer_count INTEGER DEFAULT 0;
ALTER TABLE repositories ADD COLUMN ado_build_validation_policies INTEGER DEFAULT 0;

-- Wiki & Documentation
ALTER TABLE repositories ADD COLUMN ado_has_wiki BOOLEAN DEFAULT FALSE;
ALTER TABLE repositories ADD COLUMN ado_wiki_page_count INTEGER DEFAULT 0;

-- Test Plans
ALTER TABLE repositories ADD COLUMN ado_test_plan_count INTEGER DEFAULT 0;
ALTER TABLE repositories ADD COLUMN ado_test_case_count INTEGER DEFAULT 0;

-- Artifacts & Packages
ALTER TABLE repositories ADD COLUMN ado_package_feed_count INTEGER DEFAULT 0;
ALTER TABLE repositories ADD COLUMN ado_has_artifacts BOOLEAN DEFAULT FALSE;

-- Service Hooks & Extensions
ALTER TABLE repositories ADD COLUMN ado_service_hook_count INTEGER DEFAULT 0;
ALTER TABLE repositories ADD COLUMN ado_installed_extensions TEXT;

