import Papa from 'papaparse';
import ExcelJS from 'exceljs';
import type { Repository } from '../types';
import { formatBytes } from './format';

export interface ExportRow {
  // Repository Information
  repository: string;
  organization: string;
  project?: string;
  source: string;
  status: string;
  batch?: string;
  
  // Size Information
  size_bytes: number;
  size_human: string;
  
  // Commit Information
  commit_count: number;
  commits_last_12_weeks: number;
  
  // File Characteristics
  has_lfs: boolean;
  has_submodules: boolean;
  has_large_files: boolean;
  large_file_count: number;
  largest_file_size: number;
  has_blocking_files: boolean;
  
  // Complexity
  complexity_score: number;
  
  // Git Metadata
  default_branch: string;
  branch_count: number;
  last_commit_date: string;
  visibility: string;
  is_archived: boolean;
  is_fork: boolean;
  
  // Platform-specific features (GitHub)
  workflow_count?: number;
  environment_count?: number;
  secret_count?: number;
  has_actions?: boolean;
  has_packages?: boolean;
  has_projects?: boolean;
  branch_protections?: number;
  has_rulesets?: boolean;
  contributor_count?: number;
  issue_count?: number;
  pull_request_count?: number;
  has_self_hosted_runners?: boolean;
  
  // Platform-specific features (Azure DevOps)
  is_git?: boolean;
  pipeline_count?: number;
  yaml_pipelines?: number;
  classic_pipelines?: number;
  has_boards?: boolean;
  has_wiki?: boolean;
  ado_pull_requests?: number;
  work_items?: number;
  branch_policies?: number;
  test_plans?: number;
  package_feeds?: number;
  service_hooks?: number;
}

/**
 * Convert a repository to an export row matching discovery data format
 */
export function repositoryToExportRow(repo: Repository): ExportRow {
  // Extract organization from full_name or use ado_project
  const organization = repo.ado_project || repo.full_name.split('/')[0];
  
  // Format dates
  const lastCommitDate = repo.last_commit_date ? new Date(repo.last_commit_date).toISOString().split('T')[0] : '';
  const defaultBranch = repo.default_branch || '';
  
  // Base export row with common fields
  const exportRow: ExportRow = {
    // Repository Information
    repository: repo.full_name,
    organization,
    source: repo.source,
    status: repo.status,
    
    // Size Information
    size_bytes: repo.total_size,
    size_human: formatBytes(repo.total_size),
    
    // Commit Information
    commit_count: repo.commit_count,
    commits_last_12_weeks: repo.commits_last_12_weeks || 0,
    
    // File Characteristics
    has_lfs: repo.has_lfs,
    has_submodules: repo.has_submodules,
    has_large_files: repo.has_large_files,
    large_file_count: repo.large_file_count || 0,
    largest_file_size: repo.largest_file_size || 0,
    has_blocking_files: repo.has_blocking_files || false,
    
    // Complexity
    complexity_score: repo.complexity_score || 0,
    
    // Git Metadata
    default_branch: defaultBranch,
    branch_count: repo.branch_count,
    last_commit_date: lastCommitDate,
    visibility: repo.visibility,
    is_archived: repo.is_archived,
    is_fork: repo.is_fork,
  };
  
  // Add Azure DevOps specific fields
  if (repo.source === 'azuredevops') {
    exportRow.project = repo.ado_project || '';
    exportRow.is_git = repo.ado_is_git;
    exportRow.pipeline_count = repo.ado_pipeline_count || 0;
    exportRow.yaml_pipelines = repo.ado_yaml_pipeline_count || 0;
    exportRow.classic_pipelines = repo.ado_classic_pipeline_count || 0;
    exportRow.has_boards = repo.ado_has_boards;
    exportRow.has_wiki = repo.ado_has_wiki;
    exportRow.ado_pull_requests = repo.ado_pull_request_count || 0;
    exportRow.work_items = repo.ado_work_item_count || 0;
    exportRow.branch_policies = repo.ado_branch_policy_count || 0;
    exportRow.test_plans = repo.ado_test_plan_count || 0;
    exportRow.package_feeds = repo.ado_package_feed_count || 0;
    exportRow.service_hooks = repo.ado_service_hook_count || 0;
  } else {
    // Add GitHub specific fields
    exportRow.workflow_count = repo.workflow_count || 0;
    exportRow.environment_count = repo.environment_count || 0;
    exportRow.secret_count = repo.secret_count || 0;
    exportRow.has_actions = repo.has_actions;
    exportRow.has_packages = repo.has_packages;
    exportRow.has_projects = repo.has_projects;
    exportRow.branch_protections = repo.branch_protections || 0;
    exportRow.has_rulesets = repo.has_rulesets;
    exportRow.contributor_count = repo.contributor_count || 0;
    exportRow.issue_count = repo.issue_count || 0;
    exportRow.pull_request_count = repo.pull_request_count || 0;
    exportRow.has_self_hosted_runners = repo.has_self_hosted_runners;
  }
  
  return exportRow;
}

/**
 * Export repositories to CSV format
 */
export function exportToCSV(repositories: Repository[], filename: string = 'repositories.csv'): void {
  const rows = repositories.map(repositoryToExportRow);
  const csv = Papa.unparse(rows);

  // Create blob and download
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' });
  downloadBlob(blob, filename);
}

/**
 * Export repositories to Excel format
 */
export async function exportToExcel(repositories: Repository[], filename: string = 'repositories.xlsx'): Promise<void> {
  const rows = repositories.map(repositoryToExportRow);

  // Create workbook and worksheet
  const workbook = new ExcelJS.Workbook();
  const worksheet = workbook.addWorksheet('Repositories');

  // Add header row
  const headers = Object.keys(rows[0] || {});
  worksheet.addRow(headers);

  // Style header row
  worksheet.getRow(1).font = { bold: true };
  worksheet.getRow(1).fill = {
    type: 'pattern',
    pattern: 'solid',
    fgColor: { argb: 'FFE0E0E0' }
  };

  // Add data rows
  rows.forEach(row => {
    worksheet.addRow(Object.values(row));
  });

  // Auto-fit column widths based on content
  worksheet.columns.forEach((column, index) => {
    let maxLength = 0;
    const headerLength = headers[index]?.length || 10;
    
    column?.eachCell?.({ includeEmpty: false }, (cell) => {
      const cellValue = cell.value?.toString() || '';
      maxLength = Math.max(maxLength, cellValue.length);
    });
    
    column.width = Math.max(Math.min(maxLength + 2, 50), headerLength + 2, 10);
  });

  // Generate Excel file
  const buffer = await workbook.xlsx.writeBuffer();
  const blob = new Blob([buffer], { type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' });
  downloadBlob(blob, filename);
}

/**
 * Export repositories to JSON format
 */
export function exportToJSON(repositories: Repository[], filename: string = 'repositories.json'): void {
  const rows = repositories.map(repositoryToExportRow);
  const json = JSON.stringify(rows, null, 2);

  // Create blob and download
  const blob = new Blob([json], { type: 'application/json;charset=utf-8;' });
  downloadBlob(blob, filename);
}

/**
 * Helper function to download a blob as a file
 */
function downloadBlob(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
}

/**
 * Get appropriate filename with timestamp
 */
export function getTimestampedFilename(baseName: string, extension: string): string {
  const timestamp = new Date().toISOString().replace(/[:.]/g, '-').slice(0, -5);
  return `${baseName}_${timestamp}.${extension}`;
}

