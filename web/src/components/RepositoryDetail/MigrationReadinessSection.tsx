import { useState } from 'react';
import { Repository } from '../../types';
import { api } from '../../services/api';
import { formatBytes } from '../../utils/format';
import { CollapsibleValidationSection } from './CollapsibleValidationSection';
import { MetadataBreakdownBar } from './MetadataBreakdownBar';

interface MigrationReadinessSectionProps {
  repository: Repository;
  onUpdate: () => void;
  onRevalidate: () => void;
}

interface ValidationIssue {
  sha?: string;
  path?: string;
  size_bytes?: number;
  size_mb?: number;
}

interface MetadataDetails {
  issues_estimate_bytes: number;
  prs_estimate_bytes: number;
  attachments_estimate_bytes: number;
  releases_bytes: number;
  overhead_bytes: number;
  total_bytes: number;
}

interface Recommendation {
  id: string;
  type: 'exclude_releases' | 'exclude_attachments';
  title: string;
  description: string;
  impact: string;
  priority: 'high' | 'medium' | 'low';
}

export function MigrationReadinessSection({ 
  repository, 
  onUpdate, 
  onRevalidate 
}: MigrationReadinessSectionProps) {
  // Exclusion flags state
  const [flags, setFlags] = useState({
    exclude_releases: repository.exclude_releases,
    exclude_attachments: repository.exclude_attachments,
    exclude_metadata: repository.exclude_metadata,
    exclude_git_data: repository.exclude_git_data,
    exclude_owner_projects: repository.exclude_owner_projects,
  });
  
  const [isSaving, setIsSaving] = useState(false);
  const [isRemediating, setIsRemediating] = useState(false);
  
  // Section expansion state
  const [expandedSections, setExpandedSections] = useState<Record<string, boolean>>({
    repoSize: repository.has_oversized_repository,
    metadata: repository.estimated_metadata_size ? repository.estimated_metadata_size > 35 * 1024 * 1024 * 1024 : false,
    gitLimits: repository.has_oversized_commits || repository.has_long_refs || repository.has_blocking_files,
    options: false,
  });
  
  // Parse JSON details
  const parseDetails = (details?: string): ValidationIssue[] | string[] => {
    if (!details) return [];
    try {
      return JSON.parse(details);
    } catch {
      return [];
    }
  };
  
  const parseMetadataDetails = (details?: string): MetadataDetails | null => {
    if (!details) return null;
    try {
      return JSON.parse(details);
    } catch {
      return null;
    }
  };
  
  const oversizedCommits = parseDetails(repository.oversized_commit_details) as ValidationIssue[];
  const longRefs = parseDetails(repository.long_ref_details) as string[];
  const blockingFiles = parseDetails(repository.blocking_file_details) as ValidationIssue[];
  const warningFiles = parseDetails(repository.large_file_warning_details) as ValidationIssue[];
  const metadataDetails = parseMetadataDetails(repository.metadata_size_details);
  
  // Determine validation statuses
  const hasOversizedRepo = repository.has_oversized_repository;
  const hasGitLimitIssues = repository.has_oversized_commits || repository.has_long_refs || repository.has_blocking_files;
  const hasBlockingIssues = hasOversizedRepo || hasGitLimitIssues;
  const hasMetadataWarning = repository.estimated_metadata_size && repository.estimated_metadata_size > 35 * 1024 * 1024 * 1024;
  const hasLargeFileWarning = repository.has_large_file_warnings;
  
  // Generate recommendations
  const generateRecommendations = (): Recommendation[] => {
    const recommendations: Recommendation[] = [];
    
    if (metadataDetails) {
      // Recommend excluding releases if they're large (>10 GB)
      if (metadataDetails.releases_bytes > 10 * 1024 * 1024 * 1024 && !flags.exclude_releases) {
        recommendations.push({
          id: 'exclude_releases',
          type: 'exclude_releases',
          title: 'Exclude Releases',
          description: 'Release assets make up most of your metadata size',
          impact: `Saves ~${formatBytes(metadataDetails.releases_bytes)}`,
          priority: 'high'
        });
      }
    }
    
    return recommendations.sort((a, b) => {
      const priorityWeight = { high: 1, medium: 2, low: 3 };
      return priorityWeight[a.priority] - priorityWeight[b.priority];
    });
  };
  
  const recommendations = generateRecommendations();
  
  // Check if there are unsaved changes
  const hasChanges = 
    flags.exclude_releases !== repository.exclude_releases ||
    flags.exclude_attachments !== repository.exclude_attachments ||
    flags.exclude_metadata !== repository.exclude_metadata ||
    flags.exclude_git_data !== repository.exclude_git_data ||
    flags.exclude_owner_projects !== repository.exclude_owner_projects;
  
  // Toggle section expansion
  const toggleSection = (section: string) => {
    setExpandedSections(prev => ({
      ...prev,
      [section]: !prev[section]
    }));
  };
  
  // Apply recommendations
  const applyRecommendations = () => {
    const newFlags = { ...flags };
    recommendations.forEach(rec => {
      if (rec.type === 'exclude_releases') {
        newFlags.exclude_releases = true;
      } else if (rec.type === 'exclude_attachments') {
        newFlags.exclude_attachments = true;
      }
    });
    setFlags(newFlags);
    // Auto-expand options section to show the changes
    setExpandedSections(prev => ({ ...prev, options: true }));
  };
  
  // Save exclusion flags
  const handleSave = async () => {
    setIsSaving(true);
    try {
      await api.updateRepository(repository.full_name, flags);
      alert('Migration options saved successfully!');
      onUpdate();
    } catch (error: any) {
      alert(`Failed to save migration options: ${error.response?.data?.error || error.message}`);
    } finally {
      setIsSaving(false);
    }
  };
  
  // Reset flags
  const handleReset = () => {
    setFlags({
      exclude_releases: repository.exclude_releases,
      exclude_attachments: repository.exclude_attachments,
      exclude_metadata: repository.exclude_metadata,
      exclude_git_data: repository.exclude_git_data,
      exclude_owner_projects: repository.exclude_owner_projects,
    });
  };
  
  // Mark as remediated
  const handleMarkRemediated = async () => {
    if (!confirm('Have you fixed all blocking issues in the source repository? This will trigger a full re-validation.')) {
      return;
    }
    
    setIsRemediating(true);
    try {
      await api.markRepositoryRemediated(repository.full_name);
      alert('Re-validation started. The repository will be re-analyzed for migration limits.');
      onRevalidate();
    } catch (error: any) {
      alert(`Failed to start re-validation: ${error.response?.data?.error || error.message}`);
    } finally {
      setIsRemediating(false);
    }
  };
  
  // Render the side-by-side layout
  const validationStatus = hasBlockingIssues ? 'blocking' : hasMetadataWarning || hasLargeFileWarning ? 'warning' : 'passed';
  
  // Render migration options section content
  function renderMigrationOptions() {
    return (
      <>
        <p className="text-gray-600 text-sm mb-3">
          Configure what data to include or exclude from the migration.
        </p>
        
        {/* Exclude Releases */}
        <div className="border border-gray-200 rounded-lg p-3">
          <label className="flex items-start cursor-pointer">
            <input
              type="checkbox"
              checked={flags.exclude_releases}
              onChange={(e) => setFlags({ ...flags, exclude_releases: e.target.checked })}
              className="mt-1 mr-3 h-4 w-4"
            />
            <div className="flex-1">
              <div className="font-medium text-gray-900">Exclude Releases</div>
              <div className="text-sm text-gray-600 mt-1">
                Skip migrating releases and their assets. This can significantly reduce metadata size for repositories with large release assets.
              </div>
              {metadataDetails && metadataDetails.releases_bytes > 0 && (
                <div className="text-sm text-blue-700 mt-1 font-medium">
                  💡 Estimated savings: ~{formatBytes(metadataDetails.releases_bytes)}
                </div>
              )}
            </div>
          </label>
        </div>
        
        {/* Save/Reset buttons */}
        {hasChanges && (
          <div className="pt-3 mt-3 border-t border-gray-200 flex gap-2">
            <button
              onClick={handleSave}
              disabled={isSaving}
              className="flex-1 bg-blue-600 hover:bg-blue-700 disabled:bg-blue-400 text-white px-4 py-2 rounded-lg text-sm font-medium transition-colors"
            >
              {isSaving ? 'Saving...' : 'Save Options'}
            </button>
            <button
              onClick={handleReset}
              disabled={isSaving}
              className="px-4 py-2 border border-gray-300 text-gray-700 rounded-lg text-sm font-medium hover:bg-gray-50 disabled:opacity-50"
            >
              Reset
            </button>
          </div>
        )}
      </>
    );
  }
  
  return (
    <div className="bg-white rounded-lg shadow">
      {/* Alert Banner (only shown when blocking issues exist) */}
      {hasBlockingIssues && (
        <div className="bg-red-600 text-white p-4 rounded-t-lg">
          <div className="flex items-center gap-3 mb-2">
            <svg className="w-6 h-6 flex-shrink-0" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" />
            </svg>
            <h3 className="text-lg font-semibold">MIGRATION BLOCKED - Action Required</h3>
          </div>
          <p className="text-red-50">
            This repository cannot be migrated due to {hasOversizedRepo ? 'repository size' : 'git'} limits. 
            Review validation details below and remediate before attempting migration.
          </p>
        </div>
      )}
      
      {/* Recommendations Banner */}
      {recommendations.length > 0 && !hasBlockingIssues && (
        <div className="bg-blue-50 border-b border-blue-200 p-4 rounded-t-lg">
          <div className="flex items-center justify-between">
            <div>
              <div className="flex items-center gap-2 mb-1">
                <svg className="w-5 h-5 text-blue-600" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" d="M12 18v-5.25m0 0a6.01 6.01 0 001.5-.189m-1.5.189a6.01 6.01 0 01-1.5-.189m3.75 7.478a12.06 12.06 0 01-4.5 0m3.75 2.383a14.406 14.406 0 01-3 0M14.25 18v-.192c0-.983.658-1.823 1.508-2.316a7.5 7.5 0 10-7.517 0c.85.493 1.509 1.333 1.509 2.316V18" />
                </svg>
                <h4 className="font-semibold text-blue-900">Recommended Actions</h4>
              </div>
              <ul className="text-sm text-blue-800 space-y-1">
                {recommendations.map(rec => (
                  <li key={rec.id}>
                    • <strong>{rec.title}:</strong> {rec.description} ({rec.impact})
                  </li>
                ))}
              </ul>
            </div>
            <button
              onClick={applyRecommendations}
              className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg font-medium transition-colors ml-4 flex-shrink-0"
            >
              Apply All
            </button>
          </div>
        </div>
      )}
      
      {/* Side-by-Side Layout */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 p-6">
        
        {/* LEFT COLUMN: Validation Status */}
        <div className="space-y-4">
          <div className="flex items-center gap-2 mb-4">
            {validationStatus === 'blocking' ? (
              <svg className="w-6 h-6 text-red-600" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636" />
              </svg>
            ) : validationStatus === 'warning' ? (
              <svg className="w-6 h-6 text-yellow-600" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" />
              </svg>
            ) : (
              <svg className="w-6 h-6 text-green-600" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
            )}
            <h3 className="text-lg font-semibold">Validation Status</h3>
          </div>
          
          {/* Status Summary */}
          <div className={`p-4 rounded-lg border-2 ${
            validationStatus === 'blocking' ? 'bg-red-50 border-red-200' :
            validationStatus === 'warning' ? 'bg-yellow-50 border-yellow-200' :
            'bg-green-50 border-green-200'
          }`}>
            <p className={`font-medium ${
              validationStatus === 'blocking' ? 'text-red-900' :
              validationStatus === 'warning' ? 'text-yellow-900' :
              'text-green-900'
            }`}>
              {validationStatus === 'blocking' ? 'Migration Blocked' :
               validationStatus === 'warning' ? 'Warnings Detected' :
               'All Checks Passed'}
            </p>
            <p className={`text-sm mt-1 ${
              validationStatus === 'blocking' ? 'text-red-700' :
              validationStatus === 'warning' ? 'text-yellow-700' :
              'text-green-700'
            }`}>
              {validationStatus === 'blocking' ? 'Repository has blocking issues that must be fixed before migration.' :
               validationStatus === 'warning' ? 'Repository can be migrated but has warnings to review.' :
               'Repository passes all GitHub migration limit validations.'}
            </p>
            
            {/* Validation Details Summary */}
            <div className="mt-3 pt-3 border-t border-gray-200">
              <ul className="space-y-1 text-xs">
                {/* Repository Size */}
                <li className="flex items-center gap-2">
                  {hasOversizedRepo ? (
                    <svg className="w-4 h-4 text-red-600 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                      <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.28 7.22a.75.75 0 00-1.06 1.06L8.94 10l-1.72 1.72a.75.75 0 101.06 1.06L10 11.06l1.72 1.72a.75.75 0 101.06-1.06L11.06 10l1.72-1.72a.75.75 0 00-1.06-1.06L10 8.94 8.28 7.22z" clipRule="evenodd" />
                    </svg>
                  ) : (
                    <svg className="w-4 h-4 text-green-600 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                      <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.857-9.809a.75.75 0 00-1.214-.882l-3.483 4.79-1.88-1.88a.75.75 0 10-1.06 1.061l2.5 2.5a.75.75 0 001.137-.089l4-5.5z" clipRule="evenodd" />
                    </svg>
                  )}
                  <span className={hasOversizedRepo ? 'text-red-700 font-medium' : 'text-gray-600'}>
                    Repository Size: {repository.total_size ? `${formatBytes(repository.total_size)} / 40 GB` : 'Not measured'}
                  </span>
                </li>
                
                {/* File Size Limits */}
                <li className="flex items-center gap-2">
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
                  <span className={repository.has_blocking_files ? 'text-red-700 font-medium' : repository.has_large_file_warnings ? 'text-yellow-700' : 'text-gray-600'}>
                    File Sizes: {repository.has_blocking_files ? 'Files >400 MB found' : repository.has_large_file_warnings ? 'Large files (100-400 MB)' : 'Within limits'}
                  </span>
                </li>
                
                {/* Metadata Size */}
                {repository.estimated_metadata_size && (
                  <li className="flex items-center gap-2">
                    {hasMetadataWarning ? (
                      <svg className="w-4 h-4 text-yellow-600 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                        <path fillRule="evenodd" d="M8.485 2.495c.673-1.167 2.357-1.167 3.03 0l6.28 10.875c.673 1.167-.17 2.625-1.516 2.625H3.72c-1.347 0-2.189-1.458-1.515-2.625L8.485 2.495zM10 5a.75.75 0 01.75.75v3.5a.75.75 0 01-1.5 0v-3.5A.75.75 0 0110 5zm0 9a1 1 0 100-2 1 1 0 000 2z" clipRule="evenodd" />
                      </svg>
                    ) : (
                      <svg className="w-4 h-4 text-green-600 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                        <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.857-9.809a.75.75 0 00-1.214-.882l-3.483 4.79-1.88-1.88a.75.75 0 10-1.06 1.061l2.5 2.5a.75.75 0 001.137-.089l4-5.5z" clipRule="evenodd" />
                      </svg>
                    )}
                    <span className={hasMetadataWarning ? 'text-yellow-700' : 'text-gray-600'}>
                      Metadata: {formatBytes(repository.estimated_metadata_size)} / 40 GB
                    </span>
                  </li>
                )}
                
                {/* Git Limits */}
                <li className="flex items-center gap-2">
                  {(repository.has_oversized_commits || repository.has_long_refs) ? (
                    <svg className="w-4 h-4 text-red-600 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                      <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.28 7.22a.75.75 0 00-1.06 1.06L8.94 10l-1.72 1.72a.75.75 0 101.06 1.06L10 11.06l1.72 1.72a.75.75 0 101.06-1.06L11.06 10l1.72-1.72a.75.75 0 00-1.06-1.06L10 8.94 8.28 7.22z" clipRule="evenodd" />
                    </svg>
                  ) : (
                    <svg className="w-4 h-4 text-green-600 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                      <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.857-9.809a.75.75 0 00-1.214-.882l-3.483 4.79-1.88-1.88a.75.75 0 10-1.06 1.061l2.5 2.5a.75.75 0 001.137-.089l4-5.5z" clipRule="evenodd" />
                    </svg>
                  )}
                  <span className={(repository.has_oversized_commits || repository.has_long_refs) ? 'text-red-700 font-medium' : 'text-gray-600'}>
                    Git Limits: {(repository.has_oversized_commits || repository.has_long_refs) ? 'Issues found' : 'All within limits'}
                  </span>
                </li>
              </ul>
            </div>
          </div>
          
          {/* Collapsible Validation Details */}
          {(hasBlockingIssues || hasMetadataWarning || hasLargeFileWarning) && (
            <button
              onClick={() => setExpandedSections(prev => ({ 
                ...prev, 
                repoSize: !prev.repoSize,
                metadata: !prev.metadata,
                gitLimits: !prev.gitLimits
              }))}
              className="text-sm text-blue-600 hover:text-blue-700 font-medium"
            >
              {(expandedSections.repoSize || expandedSections.metadata || expandedSections.gitLimits) ? '▼ Hide' : '▶ View'} Validation Details
            </button>
          )}
      
          {/* Repository Size Validation */}
      {(hasOversizedRepo || repository.total_size) && (
        <CollapsibleValidationSection
          id="repoSize"
          title="Repository Size Validation"
          status={hasOversizedRepo ? 'blocking' : 'passed'}
          expanded={expandedSections.repoSize}
          onToggle={() => toggleSection('repoSize')}
        >
          {hasOversizedRepo && repository.oversized_repository_details ? (
            <div>
              <div className="bg-red-100 p-4 rounded-lg mb-4">
                {(() => {
                  try {
                    const details = JSON.parse(repository.oversized_repository_details);
                    const sizeGB = details.size_gb || 0;
                    const limitGB = details.limit_gb || 40;
                    const percentage = (sizeGB / limitGB) * 100;
                    
                    return (
                      <>
                        <div className="mb-3">
                          <div className="flex justify-between text-sm mb-1">
                            <span className="font-medium text-red-900">Current Size: {sizeGB.toFixed(2)} GB</span>
                            <span className="text-red-700">Limit: {limitGB} GB</span>
                          </div>
                          <div className="w-full bg-red-200 rounded-full h-4 overflow-hidden">
                            <div 
                              className="bg-red-600 h-full transition-all"
                              style={{ width: `${Math.min(percentage, 100)}%` }}
                            />
                          </div>
                        </div>
                        <p className="text-sm text-red-800">
                          <strong>Exceeds limit by:</strong> {(sizeGB - limitGB).toFixed(2)} GB ({(percentage - 100).toFixed(1)}% over)
                        </p>
                      </>
                    );
                  } catch {
                    return <p className="text-red-800">Repository exceeds the 40 GB size limit.</p>;
                  }
                })()}
              </div>
              
              <div className="space-y-3">
                <h4 className="font-semibold text-gray-900">Remediation Steps:</h4>
                <ul className="list-disc list-inside space-y-2 text-sm text-gray-700">
                  <li>
                    <strong>Convert large files to Git LFS:</strong> Move large binary files to Git Large File Storage
                    <div className="ml-5 mt-1">
                      <a 
                        href="https://docs.github.com/en/repositories/working-with-files/managing-large-files/about-git-large-file-storage" 
                        className="text-blue-600 hover:underline text-xs"
                        target="_blank"
                        rel="noopener noreferrer"
                      >
                        Learn about Git LFS →
                      </a>
                    </div>
                  </li>
                  <li>
                    <strong>Remove large files from history:</strong> Use BFG Repo-Cleaner or git-filter-repo to clean up repository history
                    <div className="ml-5 mt-1">
                      <a 
                        href="https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/removing-sensitive-data-from-a-repository" 
                        className="text-blue-600 hover:underline text-xs"
                        target="_blank"
                        rel="noopener noreferrer"
                      >
                        Guide to removing files from history →
                      </a>
                    </div>
                  </li>
                  <li>
                    <strong>Split repository:</strong> Consider splitting into smaller repositories based on logical boundaries
                  </li>
                </ul>
                
                <div className="pt-4 border-t border-red-200 mt-4">
                  <button
                    onClick={handleMarkRemediated}
                    disabled={isRemediating}
                    className="bg-red-600 hover:bg-red-700 disabled:bg-red-400 text-white px-4 py-2 rounded-lg font-medium transition-colors"
                  >
                    {isRemediating ? 'Re-validating...' : 'Mark as Remediated'}
                  </button>
                  <p className="text-xs text-gray-600 mt-2">
                    After fixing these issues in the source repository, click to re-validate.
                  </p>
                </div>
              </div>
            </div>
          ) : (
            <div className="text-green-700">
              <p className="mb-2">✓ Repository size is within the 40 GB limit</p>
              {repository.total_size && (
                <p className="text-sm text-gray-600">
                  Current size: {formatBytes(repository.total_size)}
                </p>
              )}
            </div>
          )}
        </CollapsibleValidationSection>
      )}
      
      {/* Metadata Size Validation */}
      {(hasMetadataWarning || repository.estimated_metadata_size) && (
        <CollapsibleValidationSection
          id="metadata"
          title="Metadata Size Validation"
          status={hasMetadataWarning ? 'warning' : 'passed'}
          expanded={expandedSections.metadata}
          onToggle={() => toggleSection('metadata')}
        >
          {hasMetadataWarning && metadataDetails ? (
            <div className="space-y-4">
              <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4">
                <p className="text-yellow-800 mb-3">
                  This repository has a large amount of metadata (issues, PRs, releases) that is approaching GitHub's 40 GB metadata limit.
                </p>
                
                <MetadataBreakdownBar
                  releases={metadataDetails.releases_bytes}
                  issues={metadataDetails.issues_estimate_bytes}
                  prs={metadataDetails.prs_estimate_bytes}
                  attachments={metadataDetails.attachments_estimate_bytes}
                  total={metadataDetails.total_bytes}
                  limit={40 * 1024 * 1024 * 1024}
                />
              </div>
              
              <div>
                <h4 className="font-semibold text-gray-900 mb-2">💡 Solution: Use Exclusion Flags</h4>
                <p className="text-sm text-gray-700 mb-3">
                  You can reduce metadata size by excluding certain data types from the migration. 
                  Configure exclusion options in the "Migration Options" section below.
                </p>
                {!expandedSections.options && (
                  <button
                    onClick={() => toggleSection('options')}
                    className="text-blue-600 hover:text-blue-700 font-medium text-sm"
                  >
                    Open Migration Options →
                  </button>
                )}
              </div>
            </div>
          ) : repository.estimated_metadata_size && metadataDetails ? (
            <div>
              <p className="text-green-700 mb-3">✓ Metadata size is within acceptable limits</p>
              <MetadataBreakdownBar
                releases={metadataDetails.releases_bytes}
                issues={metadataDetails.issues_estimate_bytes}
                prs={metadataDetails.prs_estimate_bytes}
                attachments={metadataDetails.attachments_estimate_bytes}
                total={metadataDetails.total_bytes}
                limit={40 * 1024 * 1024 * 1024}
              />
            </div>
          ) : (
            <p className="text-gray-600">No metadata size information available</p>
          )}
        </CollapsibleValidationSection>
      )}
      
      {/* Git Limits Validation */}
      {(hasGitLimitIssues || hasLargeFileWarning) && (
        <CollapsibleValidationSection
          id="gitLimits"
          title="Git Limits Validation"
          status={hasGitLimitIssues ? 'blocking' : 'warning'}
          expanded={expandedSections.gitLimits}
          onToggle={() => toggleSection('gitLimits')}
        >
          <div className="space-y-4">
            {/* Oversized Commits */}
            {repository.has_oversized_commits && oversizedCommits.length > 0 && (
              <div className="bg-red-50 border border-red-200 rounded-lg p-4">
                <h4 className="font-medium text-red-800 mb-2">
                  🚫 Commits Exceeding 2 GB Limit
                </h4>
                <p className="text-sm text-red-700 mb-2">
                  GitHub limits single commits to 2 GB. These commits must be split before migration.
                </p>
                <ul className="list-disc list-inside space-y-1 text-sm">
                  {oversizedCommits.map((commit, idx) => (
                    <li key={idx} className="text-red-600">
                      <code className="bg-red-100 px-1 rounded">{commit.sha}</code>
                      {' - '}{commit.size_mb} MB
                    </li>
                  ))}
                </ul>
              </div>
            )}
            
            {/* Long Refs */}
            {repository.has_long_refs && longRefs.length > 0 && (
              <div className="bg-red-50 border border-red-200 rounded-lg p-4">
                <h4 className="font-medium text-red-800 mb-2">
                  🚫 Git References Exceeding 255 Byte Limit
                </h4>
                <p className="text-sm text-red-700 mb-2">
                  Git reference names (branches, tags) cannot exceed 255 bytes. Rename these before migration.
                </p>
                <ul className="list-disc list-inside space-y-1 text-sm">
                  {longRefs.slice(0, 10).map((ref, idx) => (
                    <li key={idx} className="text-red-600 break-all">
                      <code className="bg-red-100 px-1 rounded text-xs">{ref}</code>
                      {' '} ({ref.length} bytes)
                    </li>
                  ))}
                  {longRefs.length > 10 && (
                    <li className="text-red-600 italic">... and {longRefs.length - 10} more</li>
                  )}
                </ul>
              </div>
            )}
            
            {/* Blocking Files */}
            {repository.has_blocking_files && blockingFiles.length > 0 && (
              <div className="bg-red-50 border border-red-200 rounded-lg p-4">
                <h4 className="font-medium text-red-800 mb-2">
                  🚫 Files Exceeding 400 MB Migration Limit
                </h4>
                <p className="text-sm text-red-700 mb-2">
                  These files exceed GitHub's 400 MB migration limit. Use Git LFS or remove from history.
                </p>
                <ul className="list-disc list-inside space-y-1 text-sm">
                  {blockingFiles.map((file, idx) => (
                    <li key={idx} className="text-red-600">
                      <code className="bg-red-100 px-1 rounded">{file.path}</code>
                      {' - '}{file.size_mb} MB
                    </li>
                  ))}
                </ul>
              </div>
            )}
            
            {/* Large File Warnings */}
            {hasLargeFileWarning && warningFiles.length > 0 && (
              <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4">
                <h4 className="font-medium text-yellow-800 mb-2">
                  ⚠️ Large Files (100-400 MB)
                </h4>
                <p className="text-sm text-yellow-700 mb-2">
                  These files are allowed during migration but exceed GitHub's post-migration 100 MB limit.
                  Consider using Git LFS.
                </p>
                <ul className="list-disc list-inside space-y-1 text-sm">
                  {warningFiles.map((file, idx) => (
                    <li key={idx} className="text-yellow-600">
                      <code className="bg-yellow-100 px-1 rounded">{file.path}</code>
                      {' - '}{file.size_mb} MB
                    </li>
                  ))}
                </ul>
              </div>
            )}
            
            {/* Remediation button for blocking issues */}
            {hasGitLimitIssues && (
              <div className="pt-4 border-t border-gray-200">
                <button
                  onClick={handleMarkRemediated}
                  disabled={isRemediating}
                  className="bg-red-600 hover:bg-red-700 disabled:bg-red-400 text-white px-4 py-2 rounded-lg font-medium transition-colors"
                >
                  {isRemediating ? 'Re-validating...' : 'Mark as Remediated'}
                </button>
                <p className="text-xs text-gray-600 mt-2">
                  After fixing these issues in the source repository, click to re-validate.
                </p>
              </div>
            )}
            
            {!hasGitLimitIssues && !hasLargeFileWarning && (
              <p className="text-green-700">✓ All git limits validations passed</p>
            )}
          </div>
        </CollapsibleValidationSection>
      )}
        </div>
        
        {/* RIGHT COLUMN: Migration Configuration */}
        <div className="space-y-4">
          <div className="flex items-center gap-2 mb-4">
            <svg className="w-6 h-6 text-gray-700" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" d="M9.594 3.94c.09-.542.56-.94 1.11-.94h2.593c.55 0 1.02.398 1.11.94l.213 1.281c.063.374.313.686.645.87.074.04.147.083.22.127.324.196.72.257 1.075.124l1.217-.456a1.125 1.125 0 011.37.49l1.296 2.247a1.125 1.125 0 01-.26 1.431l-1.003.827c-.293.24-.438.613-.431.992a6.759 6.759 0 010 .255c-.007.378.138.75.43.99l1.005.828c.424.35.534.954.26 1.43l-1.298 2.247a1.125 1.125 0 01-1.369.491l-1.217-.456c-.355-.133-.75-.072-1.076.124a6.57 6.57 0 01-.22.128c-.331.183-.581.495-.644.869l-.213 1.28c-.09.543-.56.941-1.11.941h-2.594c-.55 0-1.02-.398-1.11-.94l-.213-1.281c-.062-.374-.312-.686-.644-.87a6.52 6.52 0 01-.22-.127c-.325-.196-.72-.257-1.076-.124l-1.217.456a1.125 1.125 0 01-1.369-.49l-1.297-2.247a1.125 1.125 0 01.26-1.431l1.004-.827c.292-.24.437-.613.43-.992a6.932 6.932 0 010-.255c.007-.378-.138-.75-.43-.99l-1.004-.828a1.125 1.125 0 01-.26-1.43l1.297-2.247a1.125 1.125 0 011.37-.491l1.216.456c.356.133.751.072 1.076-.124.072-.044.146-.087.22-.128.332-.183.582-.495.644-.869l.214-1.281z" />
              <path strokeLinecap="round" strokeLinejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
            </svg>
            <h3 className="text-lg font-semibold">Migration Configuration</h3>
          </div>
          
          {/* Migration Type Selector (placeholder for ELM) */}
          <div className="p-4 border border-gray-200 rounded-lg">
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Migration Type
            </label>
            <select 
              className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              disabled
            >
              <option value="gei">GitHub Enterprise Importer (GEI)</option>
              <option value="elm" disabled>Enterprise Live Migrator (ELM) - Coming Soon</option>
            </select>
            <p className="text-xs text-gray-500 mt-1">
              Select the migration tool to use. ELM support coming soon.
            </p>
          </div>
          
          {/* Migration Options */}
          <div className="p-4 border border-gray-200 rounded-lg">
            <h4 className="font-medium text-gray-900 mb-3">Migration Options</h4>
            {renderMigrationOptions()}
          </div>
          
          {/* Additional Resources */}
          <div className="p-4 bg-gray-50 border border-gray-200 rounded-lg">
            <h4 className="font-medium text-gray-800 mb-2 text-sm">Additional Resources</h4>
            <ul className="text-xs text-gray-600 space-y-1">
              <li>
                • <a 
                  href="https://docs.github.com/en/migrations/using-github-enterprise-importer/migrating-between-github-products/about-migrations-between-github-products#limitations-of-github" 
                  className="text-blue-600 hover:underline" 
                  target="_blank" 
                  rel="noopener noreferrer"
                >
                  GitHub Migration Limitations
                </a>
              </li>
              <li>
                • <a 
                  href="https://docs.github.com/en/repositories/working-with-files/managing-large-files" 
                  className="text-blue-600 hover:underline" 
                  target="_blank" 
                  rel="noopener noreferrer"
                >
                  Managing Large Files with Git LFS
                </a>
              </li>
              <li>
                • <a 
                  href="https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/removing-sensitive-data-from-a-repository" 
                  className="text-blue-600 hover:underline" 
                  target="_blank" 
                  rel="noopener noreferrer"
                >
                  Removing Files from Git History
                </a>
              </li>
            </ul>
          </div>
        </div>
      </div>
    </div>
  );
}

