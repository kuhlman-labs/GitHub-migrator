import type { Repository } from '../../types';
import { formatBytes } from '../../utils/format';

interface RepositoryListItemProps {
  repository: Repository;
  selected: boolean;
  onToggle: (repoId: number) => void;
}

export function RepositoryListItem({ repository, selected, onToggle }: RepositoryListItemProps) {
  const getComplexityIndicator = () => {
    // Calculate complexity score matching backend logic:
    // Size tier * 3 + has_lfs (2) + has_submodules (2) + has_large_files (4) + has_packages (3) + branch_protections > 0 (1)
    const MB100 = 100 * 1024 * 1024;
    const GB1 = 1024 * 1024 * 1024;
    const GB5 = 5 * 1024 * 1024 * 1024;
    
    let sizeTier = 0;
    if (repository.total_size >= GB5) sizeTier = 3;
    else if (repository.total_size >= GB1) sizeTier = 2;
    else if (repository.total_size >= MB100) sizeTier = 1;
    
    let score = sizeTier * 3;
    if (repository.has_lfs) score += 2;
    if (repository.has_submodules) score += 2;
    if (repository.has_large_files) score += 4;
    if (repository.has_packages) score += 3; // Packages don't migrate with GEI
    if (repository.branch_protections > 0) score += 1;

    if (score <= 3) return { 
      label: 'Simple', 
      bgColor: 'bg-green-100', 
      textColor: 'text-green-800',
      borderColor: 'border-green-200'
    };
    if (score <= 6) return { 
      label: 'Medium', 
      bgColor: 'bg-yellow-100', 
      textColor: 'text-yellow-800',
      borderColor: 'border-yellow-200'
    };
    if (score <= 9) return { 
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
            {repository.full_name.split('/')[1] || repository.full_name}
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
        {(repository.is_archived || repository.is_fork || repository.has_packages || repository.has_lfs || repository.has_actions || repository.has_submodules || repository.has_large_files || repository.has_wiki) && (
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
          </div>
        )}
      </div>
    </label>
  );
}

