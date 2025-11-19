import type { Repository } from '../../types';
import { ProfileCard } from '../common/ProfileCard';
import { ProfileItem } from '../common/ProfileItem';
import { formatBytes } from '../../utils/format';

interface TechnicalProfileTabProps {
  repository: Repository;
}

export function TechnicalProfileTab({ repository }: TechnicalProfileTabProps) {
  return (
    <div className="space-y-6">
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
      {/* Git Properties */}
      <ProfileCard title="Git Properties">
        <ProfileItem label="Default Branch" value={repository.default_branch} />
        <ProfileItem 
          label="Last Commit SHA" 
          value={repository.last_commit_sha ? (
            <code className="text-xs bg-gray-100 px-2 py-1 rounded">{repository.last_commit_sha.substring(0, 8)}</code>
          ) : 'Unknown'} 
        />
        <ProfileItem label="Total Size" value={formatBytes(repository.total_size)} />
        <ProfileItem label="Branches" value={repository.branch_count} />
        <ProfileItem label="Tags/Releases" value={repository.tag_count} />
        <ProfileItem label="Commits" value={repository.commit_count.toLocaleString()} />
        <ProfileItem label="Has LFS" value={repository.has_lfs ? 'Yes' : 'No'} />
        <ProfileItem label="Has Submodules" value={repository.has_submodules ? 'Yes' : 'No'} />
        
        {/* Largest File */}
        {repository.largest_file && (
          <ProfileItem 
            label="Largest File" 
            value={
              <code className="text-xs bg-gray-100 px-2 py-1 rounded break-all">
                {repository.largest_file}
              </code>
            } 
          />
        )}
        
        {/* Largest File Size */}
        {repository.largest_file_size && (
          <ProfileItem 
            label="Largest File Size" 
            value={formatBytes(repository.largest_file_size)} 
          />
        )}
        
        {/* Largest Commit */}
        {repository.largest_commit && (
          <ProfileItem 
            label="Largest Commit" 
            value={
              <code className="text-xs bg-gray-100 px-2 py-1 rounded">
                {repository.largest_commit.substring(0, 8)}
              </code>
            } 
          />
        )}
        
        {/* Largest Commit Size */}
        {repository.largest_commit_size && (
          <ProfileItem 
            label="Largest Commit Size" 
            value={formatBytes(repository.largest_commit_size)} 
          />
        )}
      </ProfileCard>

      {/* GitHub/ADO Properties */}
      <ProfileCard title={repository.ado_project ? "Azure DevOps Properties" : "GitHub Properties"}>
        {/* Always show - handle empty visibility */}
        {repository.visibility && (
          <ProfileItem label="Visibility" value={repository.visibility} />
        )}
        
        {/* Azure DevOps Specific Properties */}
        {repository.source === 'azuredevops' && repository.ado_project && (
          <>
            <ProfileItem label="ADO Project" value={repository.ado_project} />
            <ProfileItem label="Repository Type" value={repository.ado_is_git ? "Git" : "TFVC (Requires Conversion)"} />
            {!repository.ado_is_git && (
              <div className="px-3 py-2 bg-red-50 border border-red-200 rounded-md">
                <p className="text-xs text-red-700">⚠️ TFVC repositories must be converted to Git before migration</p>
              </div>
            )}
            
            {/* Pull Requests */}
            {repository.ado_pull_request_count > 0 && (
              <ProfileItem 
                label="Pull Requests" 
                value={`${repository.ado_open_pr_count} open / ${repository.ado_pull_request_count} total`} 
              />
            )}
            {repository.ado_pr_with_linked_work_items > 0 && (
              <ProfileItem 
                label="PRs with Work Item Links" 
                value={`${repository.ado_pr_with_linked_work_items} (will migrate)`}
              />
            )}
            {repository.ado_pr_with_attachments > 0 && (
              <ProfileItem 
                label="PRs with Attachments" 
                value={`${repository.ado_pr_with_attachments} (will migrate)`}
              />
            )}
            
            {/* Branch Policies */}
            {repository.ado_branch_policy_count > 0 && (
              <>
                <ProfileItem label="Branch Policies" value={repository.ado_branch_policy_count} />
                {repository.ado_required_reviewer_count > 0 && (
                  <ProfileItem label="Required Reviewer Policies" value={repository.ado_required_reviewer_count} />
                )}
                {repository.ado_build_validation_policies > 0 && (
                  <ProfileItem label="Build Validation Policies" value={repository.ado_build_validation_policies} />
                )}
              </>
            )}
            
            {/* Azure Boards */}
            {repository.ado_has_boards && (
              <>
                <ProfileItem label="Azure Boards" value="Enabled" />
                {repository.ado_active_work_item_count > 0 && (
                  <ProfileItem 
                    label="Active Work Items" 
                    value={`${repository.ado_active_work_item_count} (⚠️ won't migrate)`}
                  />
                )}
                {repository.ado_work_item_linked_count > 0 && (
                  <ProfileItem 
                    label="Linked Work Items" 
                    value={`${repository.ado_work_item_linked_count} (links on PRs will migrate)`}
                  />
                )}
              </>
            )}
            
            {/* Azure Pipelines */}
            {repository.ado_has_pipelines && (
              <>
                <ProfileItem label="Azure Pipelines" value="Enabled" />
                {repository.ado_pipeline_count > 0 && (
                  <ProfileItem label="Total Pipelines" value={repository.ado_pipeline_count} />
                )}
                {repository.ado_yaml_pipeline_count > 0 && (
                  <ProfileItem 
                    label="YAML Pipelines" 
                    value={`${repository.ado_yaml_pipeline_count} (easier to migrate)`}
                  />
                )}
                {repository.ado_classic_pipeline_count > 0 && (
                  <ProfileItem 
                    label="Classic Pipelines" 
                    value={`${repository.ado_classic_pipeline_count} (⚠️ require recreation)`}
                  />
                )}
                {repository.ado_pipeline_run_count > 0 && (
                  <ProfileItem 
                    label="Recent Pipeline Runs" 
                    value={`${repository.ado_pipeline_run_count} (active CI/CD)`}
                  />
                )}
                {repository.ado_has_service_connections && (
                  <ProfileItem label="Service Connections" value="Yes (⚠️ recreate in GitHub)" />
                )}
                {repository.ado_has_variable_groups && (
                  <ProfileItem label="Variable Groups" value="Yes (⚠️ convert to secrets)" />
                )}
                {repository.ado_has_self_hosted_agents && (
                  <ProfileItem label="Self-Hosted Agents" value="Yes" />
                )}
              </>
            )}
            
            {/* Wiki & Documentation */}
            {repository.ado_has_wiki && (
              <>
                <ProfileItem label="Wiki" value="Enabled (⚠️ manual migration required)" />
                {repository.ado_wiki_page_count > 0 && (
                  <ProfileItem label="Wiki Pages" value={repository.ado_wiki_page_count} />
                )}
              </>
            )}
            
            {/* Test Plans */}
            {repository.ado_test_plan_count > 0 && (
              <ProfileItem 
                label="Test Plans" 
                value={`${repository.ado_test_plan_count} (⚠️ no GitHub equivalent)`}
              />
            )}
            
            {/* Package Feeds */}
            {repository.ado_package_feed_count > 0 && (
              <ProfileItem 
                label="Package Feeds" 
                value={`${repository.ado_package_feed_count} (⚠️ separate migration)`}
              />
            )}
            
            {/* Service Hooks */}
            {repository.ado_service_hook_count > 0 && (
              <ProfileItem 
                label="Service Hooks" 
                value={`${repository.ado_service_hook_count} (⚠️ recreate as webhooks)`}
              />
            )}
            
            {/* GitHub Advanced Security */}
            {repository.ado_has_ghas && (
              <ProfileItem label="GHAS for ADO" value="Enabled (enable in GitHub after migration)" />
            )}
          </>
        )}
        
        {/* GitHub-specific properties (only show when NOT ADO) */}
        {!repository.ado_project && (
          <>
            {repository.is_archived && (
              <ProfileItem label="Archived" value="Yes" />
            )}
            {repository.is_fork && (
              <ProfileItem label="Fork" value="Yes" />
            )}
            
            {repository.contributor_count > 0 && (
              <ProfileItem label="Contributors" value={repository.contributor_count} />
            )}
            {repository.issue_count > 0 && (
              <ProfileItem 
                label="Issues" 
                value={`${repository.open_issue_count} open / ${repository.issue_count} total`} 
              />
            )}
            {repository.pull_request_count > 0 && (
              <ProfileItem 
                label="Pull Requests" 
                value={`${repository.open_pr_count} open / ${repository.pull_request_count} total`} 
              />
            )}
            {repository.has_wiki && (
              <ProfileItem label="Wikis" value="Enabled" />
            )}
            {repository.has_pages && (
              <ProfileItem label="Pages" value="Enabled" />
            )}
            {repository.has_discussions && (
              <ProfileItem label="Discussions" value="Enabled" />
            )}
            {repository.has_actions && (
              <ProfileItem label="Actions" value="Enabled" />
            )}
            {repository.workflow_count > 0 && (
              <ProfileItem label="Workflows" value={repository.workflow_count} />
            )}
            {repository.has_projects && (
              <ProfileItem label="Projects" value="Enabled" />
            )}
            {repository.has_packages && (
              <ProfileItem label="Packages" value="Yes" />
            )}
            {repository.release_count > 0 && (
              <ProfileItem label="Releases" value={repository.release_count} />
            )}
            {repository.has_release_assets && (
              <ProfileItem label="Has Release Assets" value="Yes" />
            )}
            {repository.branch_protections > 0 && (
              <ProfileItem label="Branch Protections" value={repository.branch_protections} />
            )}
            {repository.has_rulesets && (
              <ProfileItem label="Rulesets" value="Yes" />
            )}
            {repository.environment_count > 0 && (
              <ProfileItem label="Environments" value={repository.environment_count} />
            )}
            {repository.secret_count > 0 && (
              <ProfileItem label="Secrets" value={repository.secret_count} />
            )}
            {repository.webhook_count > 0 && (
              <ProfileItem label="Webhooks" value={repository.webhook_count} />
            )}
            {repository.has_code_scanning && (
              <ProfileItem label="Code Scanning" value="Enabled" />
            )}
            {repository.has_dependabot && (
              <ProfileItem label="Dependabot" value="Enabled" />
            )}
            {repository.has_secret_scanning && (
              <ProfileItem label="Secret Scanning" value="Enabled" />
            )}
            {repository.has_codeowners && (
              <ProfileItem label="CODEOWNERS" value="Yes" />
            )}
            {repository.has_self_hosted_runners && (
              <ProfileItem label="Self-Hosted Runners" value="Yes" />
            )}
            {repository.collaborator_count > 0 && (
              <ProfileItem label="Outside Collaborators" value={repository.collaborator_count} />
            )}
            {repository.installed_apps_count > 0 && (
              <ProfileItem label="GitHub Apps" value={repository.installed_apps_count} />
            )}
          </>
        )}
      </ProfileCard>
      </div>
    </div>
  );
}

