import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import {
  repositoryToExportRow,
  getTimestampedFilename,
  downloadBlobAsFile,
  exportDependenciesToCSV,
  exportDependenciesToJSON,
  exportToCSV,
  exportToJSON,
} from './export';
import type { Repository } from '../types';

// Mock DOM APIs for download functions
const mockCreateObjectURL = vi.fn(() => 'blob:http://test/123');
const mockRevokeObjectURL = vi.fn();
const mockAppendChild = vi.fn();
const mockRemoveChild = vi.fn();
const mockClick = vi.fn();

describe('export utilities', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    
    // Setup DOM mocks
    Object.defineProperty(window, 'URL', {
      value: {
        createObjectURL: mockCreateObjectURL,
        revokeObjectURL: mockRevokeObjectURL,
      },
      writable: true,
    });

    vi.spyOn(document.body, 'appendChild').mockImplementation(mockAppendChild);
    vi.spyOn(document.body, 'removeChild').mockImplementation(mockRemoveChild);
    vi.spyOn(document, 'createElement').mockImplementation((tag: string) => {
      if (tag === 'a') {
        return {
          href: '',
          download: '',
          click: mockClick,
        } as unknown as HTMLAnchorElement;
      }
      return document.createElement(tag);
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('repositoryToExportRow', () => {
    const baseGitHubRepo: Repository = {
      id: 1,
      full_name: 'my-org/my-repo',
      name: 'my-repo',
      source: 'github',
      status: 'pending',
      total_size: 1024000,
      commit_count: 100,
      commits_last_12_weeks: 25,
      has_lfs: false,
      has_submodules: true,
      has_large_files: false,
      large_file_count: 0,
      largest_file_size: 0,
      has_blocking_files: false,
      complexity_score: 3,
      default_branch: 'main',
      branch_count: 5,
      last_commit_date: '2024-01-15T10:30:00Z',
      visibility: 'private',
      is_archived: false,
      is_fork: false,
      created_at: '2023-01-01T00:00:00Z',
      updated_at: '2024-01-15T10:30:00Z',
      workflow_count: 3,
      environment_count: 2,
      secret_count: 5,
      has_actions: true,
      has_packages: false,
      has_projects: true,
      branch_protections: 2,
      has_rulesets: true,
      contributor_count: 10,
      issue_count: 25,
      pull_request_count: 15,
      has_self_hosted_runners: false,
    };

    it('should convert GitHub repository to export row', () => {
      const row = repositoryToExportRow(baseGitHubRepo);

      expect(row.repository).toBe('my-org/my-repo');
      expect(row.organization).toBe('my-org');
      expect(row.source).toBe('github');
      expect(row.status).toBe('pending');
      expect(row.size_bytes).toBe(1024000);
      expect(row.size_human).toBe('1000 KB');
      expect(row.commit_count).toBe(100);
      expect(row.commits_last_12_weeks).toBe(25);
      expect(row.has_lfs).toBe(false);
      expect(row.has_submodules).toBe(true);
      expect(row.complexity_score).toBe(3);
      expect(row.default_branch).toBe('main');
      expect(row.branch_count).toBe(5);
      expect(row.last_commit_date).toBe('2024-01-15');
      expect(row.visibility).toBe('private');
      expect(row.is_archived).toBe(false);
      expect(row.is_fork).toBe(false);
    });

    it('should include GitHub-specific fields', () => {
      const row = repositoryToExportRow(baseGitHubRepo);

      expect(row.workflow_count).toBe(3);
      expect(row.environment_count).toBe(2);
      expect(row.secret_count).toBe(5);
      expect(row.has_actions).toBe(true);
      expect(row.has_packages).toBe(false);
      expect(row.has_projects).toBe(true);
      expect(row.branch_protections).toBe(2);
      expect(row.has_rulesets).toBe(true);
      expect(row.contributor_count).toBe(10);
      expect(row.issue_count).toBe(25);
      expect(row.pull_request_count).toBe(15);
      expect(row.has_self_hosted_runners).toBe(false);
    });

    it('should convert Azure DevOps repository to export row', () => {
      const adoRepo: Repository = {
        ...baseGitHubRepo,
        source: 'azuredevops',
        ado_project: 'MyProject',
        ado_is_git: true,
        ado_pipeline_count: 5,
        ado_yaml_pipeline_count: 3,
        ado_classic_pipeline_count: 2,
        ado_has_boards: true,
        ado_has_wiki: false,
        ado_pull_request_count: 8,
        ado_work_item_count: 50,
        ado_branch_policy_count: 3,
        ado_test_plan_count: 1,
        ado_package_feed_count: 2,
        ado_service_hook_count: 4,
      };

      const row = repositoryToExportRow(adoRepo);

      expect(row.organization).toBe('MyProject');
      expect(row.project).toBe('MyProject');
      expect(row.is_git).toBe(true);
      expect(row.pipeline_count).toBe(5);
      expect(row.yaml_pipelines).toBe(3);
      expect(row.classic_pipelines).toBe(2);
      expect(row.has_boards).toBe(true);
      expect(row.has_wiki).toBe(false);
      expect(row.ado_pull_requests).toBe(8);
      expect(row.work_items).toBe(50);
      expect(row.branch_policies).toBe(3);
      expect(row.test_plans).toBe(1);
      expect(row.package_feeds).toBe(2);
      expect(row.service_hooks).toBe(4);
    });

    it('should handle missing optional fields with defaults', () => {
      const minimalRepo: Repository = {
        id: 1,
        full_name: 'org/repo',
        name: 'repo',
        source: 'github',
        status: 'pending',
        total_size: 0,
        commit_count: 0,
        has_lfs: false,
        has_submodules: false,
        has_large_files: false,
        branch_count: 1,
        visibility: 'public',
        is_archived: false,
        is_fork: false,
        created_at: '2023-01-01T00:00:00Z',
        updated_at: '2023-01-01T00:00:00Z',
      };

      const row = repositoryToExportRow(minimalRepo);

      expect(row.commits_last_12_weeks).toBe(0);
      expect(row.large_file_count).toBe(0);
      expect(row.largest_file_size).toBe(0);
      expect(row.has_blocking_files).toBe(false);
      expect(row.complexity_score).toBe(0);
      expect(row.default_branch).toBe('');
      expect(row.last_commit_date).toBe('');
    });
  });

  describe('getTimestampedFilename', () => {
    beforeEach(() => {
      vi.useFakeTimers();
      vi.setSystemTime(new Date('2024-03-15T14:30:45.123Z'));
    });

    afterEach(() => {
      vi.useRealTimers();
    });

    it('should create filename with timestamp', () => {
      const filename = getTimestampedFilename('repositories', 'csv');
      expect(filename).toBe('repositories_2024-03-15T14-30-45.csv');
    });

    it('should work with different base names', () => {
      const filename = getTimestampedFilename('export', 'xlsx');
      expect(filename).toBe('export_2024-03-15T14-30-45.xlsx');
    });

    it('should work with different extensions', () => {
      const filename = getTimestampedFilename('data', 'json');
      expect(filename).toBe('data_2024-03-15T14-30-45.json');
    });
  });

  describe('downloadBlobAsFile', () => {
    it('should create blob URL and trigger download', () => {
      const blob = new Blob(['test content'], { type: 'text/plain' });
      
      downloadBlobAsFile(blob, 'test.txt');

      expect(mockCreateObjectURL).toHaveBeenCalledWith(blob);
      expect(mockAppendChild).toHaveBeenCalled();
      expect(mockClick).toHaveBeenCalled();
      expect(mockRemoveChild).toHaveBeenCalled();
      expect(mockRevokeObjectURL).toHaveBeenCalledWith('blob:http://test/123');
    });
  });

  describe('exportDependenciesToCSV', () => {
    const mockDependencies = [
      {
        repository: 'org/repo1',
        dependency_full_name: 'org/dep1',
        direction: 'depends_on' as const,
        dependency_type: 'internal',
        dependency_url: 'https://github.com/org/dep1',
      },
      {
        repository: 'org/repo2',
        dependency_full_name: 'org/dep2',
        direction: 'depended_by' as const,
        dependency_type: 'external',
        dependency_url: 'https://github.com/org/dep2',
      },
    ];

    it('should create CSV with headers and data', () => {
      let capturedBlob: Blob | null = null;
      mockCreateObjectURL.mockImplementation((blob: Blob) => {
        capturedBlob = blob;
        return 'blob:http://test/123';
      });

      exportDependenciesToCSV(mockDependencies, 'deps.csv');

      expect(capturedBlob).toBeInstanceOf(Blob);
      expect(mockClick).toHaveBeenCalled();
    });

    it('should use default filename if not provided', () => {
      exportDependenciesToCSV(mockDependencies);
      expect(mockClick).toHaveBeenCalled();
    });
  });

  describe('exportDependenciesToJSON', () => {
    const mockDependencies = [
      {
        repository: 'org/repo1',
        dependency_full_name: 'org/dep1',
        direction: 'depends_on' as const,
        dependency_type: 'internal',
        dependency_url: 'https://github.com/org/dep1',
      },
    ];

    it('should create JSON blob and trigger download', () => {
      exportDependenciesToJSON(mockDependencies, 'deps.json');
      
      expect(mockCreateObjectURL).toHaveBeenCalled();
      expect(mockClick).toHaveBeenCalled();
    });

    it('should use default filename if not provided', () => {
      exportDependenciesToJSON(mockDependencies);
      expect(mockClick).toHaveBeenCalled();
    });
  });

  describe('exportToCSV', () => {
    const mockRepos: Repository[] = [
      {
        id: 1,
        full_name: 'org/repo1',
        name: 'repo1',
        source: 'github',
        status: 'pending',
        total_size: 1024,
        commit_count: 10,
        has_lfs: false,
        has_submodules: false,
        has_large_files: false,
        branch_count: 1,
        visibility: 'public',
        is_archived: false,
        is_fork: false,
        created_at: '2023-01-01T00:00:00Z',
        updated_at: '2023-01-01T00:00:00Z',
      },
    ];

    it('should export repositories to CSV', () => {
      exportToCSV(mockRepos, 'test.csv');
      
      expect(mockCreateObjectURL).toHaveBeenCalled();
      expect(mockClick).toHaveBeenCalled();
    });

    it('should use default filename if not provided', () => {
      exportToCSV(mockRepos);
      expect(mockClick).toHaveBeenCalled();
    });
  });

  describe('exportToJSON', () => {
    const mockRepos: Repository[] = [
      {
        id: 1,
        full_name: 'org/repo2',
        name: 'repo2',
        source: 'github',
        status: 'complete',
        total_size: 2048,
        commit_count: 20,
        has_lfs: true,
        has_submodules: false,
        has_large_files: false,
        branch_count: 3,
        visibility: 'private',
        is_archived: false,
        is_fork: false,
        created_at: '2023-02-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      },
    ];

    it('should export repositories to JSON', () => {
      exportToJSON(mockRepos, 'test.json');
      
      expect(mockCreateObjectURL).toHaveBeenCalled();
      expect(mockClick).toHaveBeenCalled();
    });

    it('should use default filename if not provided', () => {
      exportToJSON(mockRepos);
      expect(mockClick).toHaveBeenCalled();
    });
  });
});

