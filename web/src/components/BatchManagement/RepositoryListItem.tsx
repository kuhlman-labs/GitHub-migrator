import type { Repository } from '../../types';
import { formatBytes } from '../../utils/format';

interface RepositoryListItemProps {
  repository: Repository;
  selected: boolean;
  onToggle: (repoId: number) => void;
}

export function RepositoryListItem({ repository, selected, onToggle }: RepositoryListItemProps) {
  const getComplexityIndicator = () => {
    // Use backend-calculated score if available (uses proper quantile-based activity scoring)
    // Only fallback to frontend calculation if backend score is missing
    let score: number;
    
    if (repository.complexity_score !== undefined && repository.complexity_score !== null) {
      score = repository.complexity_score;
    } else {
      // Fallback: Calculate complexity score matching GitHub-specific backend logic
      // Note: This uses static thresholds for activity, not quantiles like the backend
      const MB100 = 100 * 1024 * 1024;
      const GB1 = 1024 * 1024 * 1024;
      const GB5 = 5 * 1024 * 1024 * 1024;
      
      // Size tier scoring (0-9 points)
      let sizeTier = 0;
      if (repository.total_size >= GB5) sizeTier = 3;
      else if (repository.total_size >= GB1) sizeTier = 2;
      else if (repository.total_size >= MB100) sizeTier = 1;
      
      score = sizeTier * 3;
    
    // High impact features (3-4 points each)
    if (repository.has_large_files) score += 4;
    if (repository.environment_count > 0) score += 3;
    if (repository.secret_count > 0) score += 3;
    if (repository.has_packages) score += 3;
    if (repository.has_self_hosted_runners) score += 3;
    
    // Moderate impact features (2 points each)
    if (repository.variable_count > 0) score += 2;
    if (repository.has_discussions) score += 2;
    if (repository.release_count > 0) score += 2;
    if (repository.has_lfs) score += 2;
    if (repository.has_submodules) score += 2;
    if (repository.installed_apps_count > 0) score += 2;
    if (repository.has_projects) score += 2;
    
    // Low impact features (1 point each)
    if (repository.has_code_scanning || repository.has_dependabot || repository.has_secret_scanning) score += 1;
    if (repository.webhook_count > 0) score += 1;
    if (repository.branch_protections > 0) score += 1;
    if (repository.has_rulesets) score += 1;
    if (repository.visibility === 'public') score += 1;
    if (repository.visibility === 'internal') score += 1;
    if (repository.has_codeowners) score += 1;
    
    // Activity-based scoring (0-4 points) - approximated with static thresholds
    // Backend uses quantiles for more accurate per-customer calculation
    // High-activity repos require significantly more coordination and planning
    const activityScore = 
      (repository.branch_count > 50 ? 0.5 : repository.branch_count > 10 ? 0.25 : 0) +
      (repository.commit_count > 1000 ? 0.5 : repository.commit_count > 100 ? 0.25 : 0) +
      (repository.issue_count > 100 ? 0.5 : repository.issue_count > 10 ? 0.25 : 0) +
      (repository.pull_request_count > 50 ? 0.5 : repository.pull_request_count > 10 ? 0.25 : 0);
    
      if (activityScore >= 1.5) score += 4; // High activity - many users, extensive coordination
      else if (activityScore >= 0.5) score += 2; // Moderate activity - some coordination needed
    }

    if (score <= 5) return { 
      label: 'Simple', 
      bgColor: 'bg-green-100', 
      textColor: 'text-green-800',
      borderColor: 'border-green-200'
    };
    if (score <= 10) return { 
      label: 'Medium', 
      bgColor: 'bg-yellow-100', 
      textColor: 'text-yellow-800',
      borderColor: 'border-yellow-200'
    };
    if (score <= 17) return { 
      label: 'Complex', 
      bgColor: 'bg-orange-100',
      textColor: 'text-orange-800',
      borderColor: 'border-orange-200'
    };
    return { 
      label: 'Very Complex', 
      bgColor: 'bg-red-100', 
      textColor: 'text-red-800',
      borderColor: 'border-red-200'
    };
  };

  const complexity = getComplexityIndicator();

  return (
    <label
      className={`group flex items-start gap-3 p-3 border rounded-lg cursor-pointer transition-all ${
        selected 
          ? 'border-blue-500 bg-blue-50 shadow-sm' 
          : 'border-gray-200 bg-white hover:border-gray-300 hover:shadow-md'
      }`}
    >
      <input
        type="checkbox"
        checked={selected}
        onChange={() => onToggle(repository.id)}
        className="mt-0.5 rounded border-gray-300 text-blue-600 focus:ring-blue-500 focus:ring-offset-0"
      />
      
      <div className="flex-1 min-w-0">
        {/* Header with repo name and complexity badge */}
        <div className="flex items-center gap-2 mb-2">
          <span className="font-semibold text-gray-900 truncate text-sm">
            {repository.ado_project 
              ? repository.full_name // For ADO, full_name is just the repo name
              : repository.full_name.split('/')[1] || repository.full_name // For GitHub, extract repo name from org/repo
            }
          </span>
          <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-semibold ${complexity.bgColor} ${complexity.textColor} border ${complexity.borderColor}`}>
            {complexity.label}
          </span>
        </div>
        
        {/* Metadata row */}
        <div className="flex items-center gap-3 text-xs text-gray-600 mb-2">
          <span className="flex items-center gap-1">
            <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4" />
            </svg>
            {formatBytes(repository.total_size || 0)}
          </span>
          <span className="flex items-center gap-1">
            <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z" />
            </svg>
            {repository.branch_count} branches
          </span>
          {repository.commit_count > 0 && (
            <span className="flex items-center gap-1">
              <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
              </svg>
              {repository.commit_count.toLocaleString()} commits
            </span>
          )}
        </div>
        
        {/* Feature tags */}
        {(repository.is_archived || repository.is_fork || repository.has_packages || repository.has_lfs || repository.has_actions || repository.has_submodules || repository.has_large_files || repository.has_wiki || repository.has_code_scanning || repository.has_dependabot || repository.has_secret_scanning || repository.has_self_hosted_runners || repository.visibility === 'public' || repository.visibility === 'internal' || repository.has_codeowners) && (
          <div className="flex items-center gap-1.5 flex-wrap">
            {repository.is_archived && (
              <span className="inline-flex items-center gap-1 px-2 py-0.5 bg-gray-100 text-gray-600 rounded text-xs font-medium border border-gray-300">
                <svg className="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
                  <path d="M4 3a2 2 0 100 4h12a2 2 0 100-4H4z" />
                  <path fillRule="evenodd" d="M3 8h14v7a2 2 0 01-2 2H5a2 2 0 01-2-2V8zm5 3a1 1 0 011-1h2a1 1 0 110 2H9a1 1 0 01-1-1z" clipRule="evenodd" />
                </svg>
                Archived
              </span>
            )}
            {repository.is_fork && (
              <span className="inline-flex items-center gap-1 px-2 py-0.5 bg-indigo-50 text-indigo-700 rounded text-xs font-medium border border-indigo-200">
                <svg className="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M7.707 3.293a1 1 0 010 1.414L5.414 7H11a7 7 0 017 7v2a1 1 0 11-2 0v-2a5 5 0 00-5-5H5.414l2.293 2.293a1 1 0 11-1.414 1.414l-4-4a1 1 0 010-1.414l4-4a1 1 0 011.414 0z" clipRule="evenodd" />
                </svg>
                Fork
              </span>
            )}
            {repository.has_packages && (
              <span className="inline-flex items-center gap-1 px-2 py-0.5 bg-amber-50 text-amber-700 rounded text-xs font-medium border border-amber-200">
                <svg className="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
                  <path d="M3 1a1 1 0 000 2h1.22l.305 1.222a.997.997 0 00.01.042l1.358 5.43-.893.892C3.74 11.846 4.632 14 6.414 14H15a1 1 0 000-2H6.414l1-1H14a1 1 0 00.894-.553l3-6A1 1 0 0017 3H6.28l-.31-1.243A1 1 0 005 1H3zM16 16.5a1.5 1.5 0 11-3 0 1.5 1.5 0 013 0zM6.5 18a1.5 1.5 0 100-3 1.5 1.5 0 000 3z" />
                </svg>
                Packages
              </span>
            )}
            {repository.has_lfs && (
              <span className="inline-flex items-center gap-1 px-2 py-0.5 bg-purple-50 text-purple-700 rounded text-xs font-medium border border-purple-200">
                <svg className="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M3 3a1 1 0 011-1h12a1 1 0 011 1v3a1 1 0 01-.293.707L12 11.414V15a1 1 0 01-.293.707l-2 2A1 1 0 018 17v-5.586L3.293 6.707A1 1 0 013 6V3z" clipRule="evenodd" />
                </svg>
                LFS
              </span>
            )}
            {repository.has_actions && (
              <span className="inline-flex items-center gap-1 px-2 py-0.5 bg-blue-50 text-blue-700 rounded text-xs font-medium border border-blue-200">
                <svg className="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M11.3 1.046A1 1 0 0112 2v5h4a1 1 0 01.82 1.573l-7 10A1 1 0 018 18v-5H4a1 1 0 01-.82-1.573l7-10a1 1 0 011.12-.38z" clipRule="evenodd" />
                </svg>
                Actions
              </span>
            )}
            {repository.has_submodules && (
              <span className="inline-flex items-center gap-1 px-2 py-0.5 bg-orange-50 text-orange-700 rounded text-xs font-medium border border-orange-200">
                <svg className="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
                  <path d="M7 3a1 1 0 000 2h6a1 1 0 100-2H7zM4 7a1 1 0 011-1h10a1 1 0 110 2H5a1 1 0 01-1-1zM2 11a2 2 0 012-2h12a2 2 0 012 2v4a2 2 0 01-2 2H4a2 2 0 01-2-2v-4z" />
                </svg>
                Submodules
              </span>
            )}
            {repository.has_large_files && (
              <span className="inline-flex items-center gap-1 px-2 py-0.5 bg-red-50 text-red-700 rounded text-xs font-medium border border-red-200">
                <svg className="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
                </svg>
                Large Files
              </span>
            )}
            {repository.has_wiki && (
              <span className="inline-flex items-center gap-1 px-2 py-0.5 bg-gray-50 text-gray-700 rounded text-xs font-medium border border-gray-200">
                <svg className="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
                  <path d="M9 4.804A7.968 7.968 0 005.5 4c-1.255 0-2.443.29-3.5.804v10A7.969 7.969 0 015.5 14c1.669 0 3.218.51 4.5 1.385A7.962 7.962 0 0114.5 14c1.255 0 2.443.29 3.5.804v-10A7.968 7.968 0 0014.5 4c-1.255 0-2.443.29-3.5.804V12a1 1 0 11-2 0V4.804z" />
                </svg>
                Wiki
              </span>
            )}
            {(repository.has_code_scanning || repository.has_dependabot || repository.has_secret_scanning) && (
              <span className="inline-flex items-center gap-1 px-2 py-0.5 bg-green-50 text-green-700 rounded text-xs font-medium border border-green-200">
                <svg className="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M2.166 4.999A11.954 11.954 0 0010 1.944 11.954 11.954 0 0017.834 5c.11.65.166 1.32.166 2.001 0 5.225-3.34 9.67-8 11.317C5.34 16.67 2 12.225 2 7c0-.682.057-1.35.166-2.001zm11.541 3.708a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
                </svg>
                Security
              </span>
            )}
            {repository.has_self_hosted_runners && (
              <span className="inline-flex items-center gap-1 px-2 py-0.5 bg-purple-50 text-purple-700 rounded text-xs font-medium border border-purple-200">
                <svg className="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M3 5a2 2 0 012-2h10a2 2 0 012 2v8a2 2 0 01-2 2h-2.22l.123.489.804.804A1 1 0 0113 18H7a1 1 0 01-.707-1.707l.804-.804L7.22 15H5a2 2 0 01-2-2V5zm5.771 7H5V5h10v7H8.771z" clipRule="evenodd" />
                </svg>
                Self-Hosted
              </span>
            )}
            {repository.visibility === 'public' && (
              <span className="inline-flex items-center gap-1 px-2 py-0.5 bg-blue-50 text-blue-700 rounded text-xs font-medium border border-blue-200">
                <svg className="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM4.332 8.027a6.012 6.012 0 011.912-2.706C6.512 5.73 6.974 6 7.5 6A1.5 1.5 0 019 7.5V8a2 2 0 004 0 2 2 0 011.523-1.943A5.977 5.977 0 0116 10c0 .34-.028.675-.083 1H15a2 2 0 00-2 2v2.197A5.973 5.973 0 0110 16v-2a2 2 0 00-2-2 2 2 0 01-2-2 2 2 0 00-1.668-1.973z" clipRule="evenodd" />
                </svg>
                Public
              </span>
            )}
            {repository.visibility === 'internal' && (
              <span className="inline-flex items-center gap-1 px-2 py-0.5 bg-yellow-50 text-yellow-700 rounded text-xs font-medium border border-yellow-200">
                <svg className="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M18 8a6 6 0 01-7.743 5.743L10 14l-1 1-1 1H6v2H2v-4l4.257-4.257A6 6 0 1118 8zm-6-4a1 1 0 100 2 2 2 0 012 2 1 1 0 102 0 4 4 0 00-4-4z" clipRule="evenodd" />
                </svg>
                Internal
              </span>
            )}
            {repository.has_codeowners && (
              <span className="inline-flex items-center gap-1 px-2 py-0.5 bg-blue-50 text-blue-700 rounded text-xs font-medium border border-blue-200">
                <svg className="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
                  <path d="M9 6a3 3 0 11-6 0 3 3 0 016 0zM17 6a3 3 0 11-6 0 3 3 0 016 0zM12.93 17c.046-.327.07-.66.07-1a6.97 6.97 0 00-1.5-4.33A5 5 0 0119 16v1h-6.07zM6 11a5 5 0 015 5v1H1v-1a5 5 0 015-5z" />
                </svg>
                CODEOWNERS
              </span>
            )}
          </div>
        )}
      </div>
    </label>
  );
}

