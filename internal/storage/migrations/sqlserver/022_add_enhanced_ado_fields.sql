-- Add enhanced Azure DevOps specific fields to repositories table

-- Enhanced Pipeline Data
ALTER TABLE repositories ADD ado_pipeline_count INT DEFAULT 0;
ALTER TABLE repositories ADD ado_yaml_pipeline_count INT DEFAULT 0;
ALTER TABLE repositories ADD ado_classic_pipeline_count INT DEFAULT 0;
ALTER TABLE repositories ADD ado_pipeline_run_count INT DEFAULT 0;
ALTER TABLE repositories ADD ado_has_service_connections BIT DEFAULT 0;
ALTER TABLE repositories ADD ado_has_variable_groups BIT DEFAULT 0;
ALTER TABLE repositories ADD ado_has_self_hosted_agents BIT DEFAULT 0;

-- Enhanced Work Item Data
ALTER TABLE repositories ADD ado_work_item_linked_count INT DEFAULT 0;
ALTER TABLE repositories ADD ado_active_work_item_count INT DEFAULT 0;
ALTER TABLE repositories ADD ado_work_item_types NVARCHAR(MAX);

-- Pull Request Details
ALTER TABLE repositories ADD ado_open_pr_count INT DEFAULT 0;
ALTER TABLE repositories ADD ado_pr_with_linked_work_items INT DEFAULT 0;
ALTER TABLE repositories ADD ado_pr_with_attachments INT DEFAULT 0;

-- Enhanced Branch Policy Data
ALTER TABLE repositories ADD ado_branch_policy_types NVARCHAR(MAX);
ALTER TABLE repositories ADD ado_required_reviewer_count INT DEFAULT 0;
ALTER TABLE repositories ADD ado_build_validation_policies INT DEFAULT 0;

-- Wiki & Documentation
ALTER TABLE repositories ADD ado_has_wiki BIT DEFAULT 0;
ALTER TABLE repositories ADD ado_wiki_page_count INT DEFAULT 0;

-- Test Plans
ALTER TABLE repositories ADD ado_test_plan_count INT DEFAULT 0;
ALTER TABLE repositories ADD ado_test_case_count INT DEFAULT 0;

-- Artifacts & Packages
ALTER TABLE repositories ADD ado_package_feed_count INT DEFAULT 0;
ALTER TABLE repositories ADD ado_has_artifacts BIT DEFAULT 0;

-- Service Hooks & Extensions
ALTER TABLE repositories ADD ado_service_hook_count INT DEFAULT 0;
ALTER TABLE repositories ADD ado_installed_extensions NVARCHAR(MAX);

