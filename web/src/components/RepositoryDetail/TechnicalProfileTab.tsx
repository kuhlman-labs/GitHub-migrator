import { useState } from 'react';
import type { Repository } from '../../types';
import { ProfileCard } from '../common/ProfileCard';
import { ProfileItem } from '../common/ProfileItem';
import { formatBytes } from '../../utils/format';

interface TechnicalProfileTabProps {
  repository: Repository;
}

export function TechnicalProfileTab({ repository }: TechnicalProfileTabProps) {
  const [expandedValidation, setExpandedValidation] = useState(false);
  
  // Determine validation status
  const hasBlockingIssues = repository.has_oversized_repository || 
    repository.has_oversized_commits || 
    repository.has_long_refs || 
    repository.has_blocking_files;
  const hasWarnings = (repository.estimated_metadata_size && repository.estimated_metadata_size > 35 * 1024 * 1024 * 1024) || 
    repository.has_large_file_warnings;

  return (
    <div className="space-y-6">
      {/* Validation Status Summary */}
      <div className="bg-white rounded-lg shadow-sm p-6">
        <h3 className="text-lg font-semibold mb-4 flex items-center gap-2">
          {hasBlockingIssues ? (
            <>
              <svg className="w-5 h-5 text-red-600" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636" />
              </svg>
              <span className="text-red-900">Validation Status</span>
            </>
          ) : hasWarnings ? (
            <>
              <svg className="w-5 h-5 text-yellow-600" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" />
              </svg>
              <span className="text-yellow-900">Validation Status</span>
            </>
          ) : (
            <>
              <svg className="w-5 h-5 text-green-600" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <span className="text-green-900">Validation Status</span>
            </>
          )}
        </h3>

        <div className={`p-4 rounded-lg border-2 mb-4 ${
          hasBlockingIssues ? 'bg-red-50 border-red-200' :
          hasWarnings ? 'bg-yellow-50 border-yellow-200' :
          'bg-green-50 border-green-200'
        }`}>
          <p className={`font-medium ${
            hasBlockingIssues ? 'text-red-900' :
            hasWarnings ? 'text-yellow-900' :
            'text-green-900'
          }`}>
            {hasBlockingIssues ? 'Migration Blocked' :
             hasWarnings ? 'Warnings Detected' :
             'All Checks Passed'}
          </p>
          <p className={`text-sm mt-1 ${
            hasBlockingIssues ? 'text-red-700' :
            hasWarnings ? 'text-yellow-700' :
            'text-green-700'
          }`}>
            {hasBlockingIssues ? 'Repository has blocking issues that must be fixed before migration.' :
             hasWarnings ? 'Repository can be migrated but has warnings to review.' :
             'Repository passes all GitHub migration limit validations.'}
          </p>
        </div>

        {/* Validation Checks */}
        <div className="space-y-2">
          <h4 className="text-sm font-semibold text-gray-700 mb-3">Validation Checks:</h4>
          
          {/* Repository Size */}
          <div className="flex items-center gap-2 py-2">
            {repository.has_oversized_repository ? (
              <svg className="w-4 h-4 text-red-600 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.28 7.22a.75.75 0 00-1.06 1.06L8.94 10l-1.72 1.72a.75.75 0 101.06 1.06L10 11.06l1.72 1.72a.75.75 0 101.06-1.06L11.06 10l1.72-1.72a.75.75 0 00-1.06-1.06L10 8.94 8.28 7.22z" clipRule="evenodd" />
              </svg>
            ) : (
              <svg className="w-4 h-4 text-green-600 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.857-9.809a.75.75 0 00-1.214-.882l-3.483 4.79-1.88-1.88a.75.75 0 10-1.06 1.061l2.5 2.5a.75.75 0 001.137-.089l4-5.5z" clipRule="evenodd" />
              </svg>
            )}
            <span className={`text-sm ${repository.has_oversized_repository ? 'text-red-700 font-medium' : 'text-gray-600'}`}>
              Repository Size: {repository.total_size ? `${formatBytes(repository.total_size)} / 40 GB` : 'Not measured'}
            </span>
          </div>

          {/* File Sizes */}
          <div className="flex items-center gap-2 py-2">
            {repository.has_blocking_files ? (
              <svg className="w-4 h-4 text-red-600 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.28 7.22a.75.75 0 00-1.06 1.06L8.94 10l-1.72 1.72a.75.75 0 101.06 1.06L10 11.06l1.72 1.72a.75.75 0 101.06-1.06L11.06 10l1.72-1.72a.75.75 0 00-1.06-1.06L10 8.94 8.28 7.22z" clipRule="evenodd" />
              </svg>
            ) : repository.has_large_file_warnings ? (
              <svg className="w-4 h-4 text-yellow-600 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M8.485 2.495c.673-1.167 2.357-1.167 3.03 0l6.28 10.875c.673 1.167-.17 2.625-1.516 2.625H3.72c-1.347 0-2.189-1.458-1.515-2.625L8.485 2.495zM10 5a.75.75 0 01.75.75v3.5a.75.75 0 01-1.5 0v-3.5A.75.75 0 0110 5zm0 9a1 1 0 100-2 1 1 0 000 2z" clipRule="evenodd" />
              </svg>
            ) : (
              <svg className="w-4 h-4 text-green-600 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.857-9.809a.75.75 0 00-1.214-.882l-3.483 4.79-1.88-1.88a.75.75 0 10-1.06 1.061l2.5 2.5a.75.75 0 001.137-.089l4-5.5z" clipRule="evenodd" />
              </svg>
            )}
            <span className={`text-sm ${
              repository.has_blocking_files ? 'text-red-700 font-medium' : 
              repository.has_large_file_warnings ? 'text-yellow-700' : 
              'text-gray-600'
            }`}>
              File Sizes: {repository.has_blocking_files ? 'Files >400 MB found' : repository.has_large_file_warnings ? 'Large files (100-400 MB)' : 'Within limits'}
            </span>
          </div>

          {/* Metadata Size - Only show if we have actual calculated metadata */}
          {repository.estimated_metadata_size && repository.estimated_metadata_size > 1024 * 1024 * 1024 && (
            <div className="flex items-center gap-2 py-2">
              {repository.estimated_metadata_size > 35 * 1024 * 1024 * 1024 ? (
                <svg className="w-4 h-4 text-yellow-600 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M8.485 2.495c.673-1.167 2.357-1.167 3.03 0l6.28 10.875c.673 1.167-.17 2.625-1.516 2.625H3.72c-1.347 0-2.189-1.458-1.515-2.625L8.485 2.495zM10 5a.75.75 0 01.75.75v3.5a.75.75 0 01-1.5 0v-3.5A.75.75 0 0110 5zm0 9a1 1 0 100-2 1 1 0 000 2z" clipRule="evenodd" />
                </svg>
              ) : (
                <svg className="w-4 h-4 text-green-600 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.857-9.809a.75.75 0 00-1.214-.882l-3.483 4.79-1.88-1.88a.75.75 0 10-1.06 1.061l2.5 2.5a.75.75 0 001.137-.089l4-5.5z" clipRule="evenodd" />
                </svg>
              )}
              <span className={`text-sm ${repository.estimated_metadata_size > 35 * 1024 * 1024 * 1024 ? 'text-yellow-700' : 'text-gray-600'}`}>
                Metadata (est.): {formatBytes(repository.estimated_metadata_size)} / 40 GB
              </span>
            </div>
          )}

          {/* Git Limits */}
          <div className="flex items-center gap-2 py-2">
            {(repository.has_oversized_commits || repository.has_long_refs) ? (
              <svg className="w-4 h-4 text-red-600 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.28 7.22a.75.75 0 00-1.06 1.06L8.94 10l-1.72 1.72a.75.75 0 101.06 1.06L10 11.06l1.72 1.72a.75.75 0 101.06-1.06L11.06 10l1.72-1.72a.75.75 0 00-1.06-1.06L10 8.94 8.28 7.22z" clipRule="evenodd" />
              </svg>
            ) : (
              <svg className="w-4 h-4 text-green-600 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.857-9.809a.75.75 0 00-1.214-.882l-3.483 4.79-1.88-1.88a.75.75 0 10-1.06 1.061l2.5 2.5a.75.75 0 001.137-.089l4-5.5z" clipRule="evenodd" />
              </svg>
            )}
            <span className={`text-sm ${(repository.has_oversized_commits || repository.has_long_refs) ? 'text-red-700 font-medium' : 'text-gray-600'}`}>
              Git Limits: {(repository.has_oversized_commits || repository.has_long_refs) ? 'Issues found' : 'All within limits'}
            </span>
          </div>
        </div>
      </div>

      {/* Detailed Validation Issues - Expandable (only if there are issues) */}
      {(hasBlockingIssues || hasWarnings) && (
        <div className="bg-white rounded-lg shadow-sm border border-gray-200">
          <button
            onClick={() => setExpandedValidation(!expandedValidation)}
            className="w-full px-6 py-4 flex items-center justify-between hover:bg-gray-50 transition-colors"
          >
            <div className="flex items-center gap-3">
              {hasBlockingIssues ? (
                <svg className="w-6 h-6 text-red-600" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636" />
                </svg>
              ) : (
                <svg className="w-6 h-6 text-yellow-600" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" />
                </svg>
              )}
              <div className="text-left">
                <h3 className="font-semibold text-gray-900">
                  {hasBlockingIssues ? 'Validation Issues (Blocking)' : 'Validation Warnings'}
                </h3>
                <p className="text-sm text-gray-600">
                  {hasBlockingIssues 
                    ? 'Repository has issues that must be resolved before migration' 
                    : 'Repository can migrate but has warnings to review'}
                </p>
              </div>
            </div>
            <svg 
              className={`w-5 h-5 text-gray-400 transition-transform ${expandedValidation ? 'rotate-180' : ''}`}
              fill="none" 
              viewBox="0 0 24 24" 
              strokeWidth={2} 
              stroke="currentColor"
            >
              <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
            </svg>
          </button>
          
          {expandedValidation && (
            <div className="px-6 pb-4 border-t border-gray-200 pt-4">
              <ul className="space-y-2 text-sm">
                {repository.has_oversized_repository && (
                  <li className="flex items-start gap-2 text-red-700">
                    <span className="text-red-600">✗</span>
                    <span>Repository size exceeds 40 GB limit ({formatBytes(repository.total_size)})</span>
                  </li>
                )}
                {repository.has_blocking_files && (
                  <li className="flex items-start gap-2 text-red-700">
                    <span className="text-red-600">✗</span>
                    <span>Files larger than 400 MB detected</span>
                  </li>
                )}
                {repository.has_oversized_commits && (
                  <li className="flex items-start gap-2 text-red-700">
                    <span className="text-red-600">✗</span>
                    <span>Commits larger than 2 GB detected</span>
                  </li>
                )}
                {repository.has_long_refs && (
                  <li className="flex items-start gap-2 text-red-700">
                    <span className="text-red-600">✗</span>
                    <span>Git references longer than 255 bytes detected</span>
                  </li>
                )}
                {repository.estimated_metadata_size && repository.estimated_metadata_size > 35 * 1024 * 1024 * 1024 && (
                  <li className="flex items-start gap-2 text-yellow-700">
                    <span className="text-yellow-600">⚠</span>
                    <span>Metadata size approaching 40 GB limit (est. {formatBytes(repository.estimated_metadata_size)})</span>
                  </li>
                )}
                {repository.has_large_file_warnings && (
                  <li className="flex items-start gap-2 text-yellow-700">
                    <span className="text-yellow-600">⚠</span>
                    <span>Large files (100-400 MB) detected - consider Git LFS</span>
                  </li>
                )}
              </ul>
            </div>
          )}
        </div>
      )}

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

