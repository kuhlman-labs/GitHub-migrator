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

      {/* GitHub Properties */}
      <ProfileCard title="GitHub Properties">
        {/* Always show */}
        <ProfileItem label="Visibility" value={repository.visibility} />
        {repository.is_archived && (
          <ProfileItem label="Archived" value="Yes" />
        )}
        {repository.is_fork && (
          <ProfileItem label="Fork" value="Yes" />
        )}
        
        {/* Show if has value */}
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
      </ProfileCard>
      </div>
    </div>
  );
}

