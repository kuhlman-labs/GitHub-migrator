import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { TechnicalProfileTab } from './TechnicalProfileTab';
import type { Repository } from '../../types';

const createMockRepository = (overrides: Partial<Repository> = {}): Repository => ({
  id: 1,
  full_name: 'org/repo',
  name: 'repo',
  org_name: 'org',
  source: 'github',
  status: 'pending',
  default_branch: 'main',
  source_url: 'https://github.com/org/repo',
  commit_count: 1500,
  branch_count: 12,
  tag_count: 25,
  pr_count: 100,
  issue_count: 50,
  visibility: 'private',
  is_archived: false,
  is_fork: false,
  has_lfs: true,
  has_submodules: true,
  has_large_files: false,
  total_size: 1024000000, // ~1 GB
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
  complexity_score: 25,
  complexity_level: 'medium',
  ...overrides,
});

describe('TechnicalProfileTab', () => {
  describe('Git Properties', () => {
    it('should render Git Properties section', () => {
      const repo = createMockRepository();
      render(<TechnicalProfileTab repository={repo} />);

      expect(screen.getByText('Git Properties')).toBeInTheDocument();
    });

    it('should display default branch', () => {
      const repo = createMockRepository({ default_branch: 'develop' });
      render(<TechnicalProfileTab repository={repo} />);

      // The label includes a colon, so use regex or getByText with colon
      expect(screen.getByText(/Default Branch/)).toBeInTheDocument();
      expect(screen.getByText('develop')).toBeInTheDocument();
    });

    it('should display commit SHA when available', () => {
      const repo = createMockRepository({ last_commit_sha: 'abc123456789def' });
      render(<TechnicalProfileTab repository={repo} />);

      expect(screen.getByText(/Last Commit SHA/)).toBeInTheDocument();
      expect(screen.getByText('abc12345')).toBeInTheDocument();
    });

    it('should display Unknown when no commit SHA', () => {
      const repo = createMockRepository({ last_commit_sha: undefined });
      render(<TechnicalProfileTab repository={repo} />);

      expect(screen.getByText('Unknown')).toBeInTheDocument();
    });

    it('should display Total Size label', () => {
      const repo = createMockRepository();
      render(<TechnicalProfileTab repository={repo} />);

      expect(screen.getByText(/Total Size/)).toBeInTheDocument();
    });

    it('should display branch count', () => {
      const repo = createMockRepository({ branch_count: 15 });
      render(<TechnicalProfileTab repository={repo} />);

      expect(screen.getByText(/Branches/)).toBeInTheDocument();
      expect(screen.getByText('15')).toBeInTheDocument();
    });

    it('should display formatted commit count', () => {
      const repo = createMockRepository({ commit_count: 5000 });
      render(<TechnicalProfileTab repository={repo} />);

      expect(screen.getByText(/^Commits:/)).toBeInTheDocument();
      expect(screen.getByText('5,000')).toBeInTheDocument();
    });

    it('should display LFS status as Yes when enabled', () => {
      const repo = createMockRepository({ has_lfs: true });
      render(<TechnicalProfileTab repository={repo} />);

      expect(screen.getByText(/Has LFS/)).toBeInTheDocument();
    });

    it('should display largest file when available', () => {
      const repo = createMockRepository({ 
        largest_file: 'data/big-file.bin'
      });
      render(<TechnicalProfileTab repository={repo} />);

      expect(screen.getByText(/Largest File/)).toBeInTheDocument();
      expect(screen.getByText('data/big-file.bin')).toBeInTheDocument();
    });
  });

  describe('GitHub Properties section', () => {
    it('should render GitHub Properties section for GitHub repos', () => {
      const repo = createMockRepository({ source: 'github' });
      render(<TechnicalProfileTab repository={repo} />);

      expect(screen.getByText('GitHub Properties')).toBeInTheDocument();
    });

    it('should display visibility when available', () => {
      const repo = createMockRepository({ visibility: 'public' });
      render(<TechnicalProfileTab repository={repo} />);

      expect(screen.getByText(/Visibility/)).toBeInTheDocument();
      expect(screen.getByText('public')).toBeInTheDocument();
    });

    it('should display Archived status when true', () => {
      const repo = createMockRepository({ is_archived: true });
      render(<TechnicalProfileTab repository={repo} />);

      expect(screen.getByText(/Archived/)).toBeInTheDocument();
    });

    it('should display Fork status when true', () => {
      const repo = createMockRepository({ is_fork: true });
      render(<TechnicalProfileTab repository={repo} />);

      expect(screen.getByText(/Fork/)).toBeInTheDocument();
    });

    it('should display contributor count when > 0', () => {
      const repo = createMockRepository({ contributor_count: 33 });
      render(<TechnicalProfileTab repository={repo} />);

      expect(screen.getByText(/Contributors/)).toBeInTheDocument();
      expect(screen.getByText('33')).toBeInTheDocument();
    });

    it('should display issue counts when > 0', () => {
      const repo = createMockRepository({ issue_count: 100, open_issue_count: 15 });
      render(<TechnicalProfileTab repository={repo} />);

      expect(screen.getByText(/Issues/)).toBeInTheDocument();
      expect(screen.getByText('15 open / 100 total')).toBeInTheDocument();
    });

    it('should display pull request counts when > 0', () => {
      const repo = createMockRepository({ pull_request_count: 50, open_pr_count: 5 });
      render(<TechnicalProfileTab repository={repo} />);

      expect(screen.getByText(/5 open \/ 50 total/)).toBeInTheDocument();
    });

    it('should display wiki status when enabled', () => {
      const repo = createMockRepository({ has_wiki: true });
      render(<TechnicalProfileTab repository={repo} />);

      expect(screen.getByText(/Wikis/)).toBeInTheDocument();
    });

    it('should display discussions status when enabled', () => {
      const repo = createMockRepository({ has_discussions: true });
      render(<TechnicalProfileTab repository={repo} />);

      expect(screen.getByText(/Discussions/)).toBeInTheDocument();
    });

    it('should display workflow count when > 0', () => {
      const repo = createMockRepository({ workflow_count: 10 });
      render(<TechnicalProfileTab repository={repo} />);

      expect(screen.getByText(/Workflows/)).toBeInTheDocument();
      expect(screen.getByText('10')).toBeInTheDocument();
    });
  });

  describe('Azure DevOps Properties', () => {
    it('should render Azure DevOps Properties section for ADO repos', () => {
      const repo = createMockRepository({ 
        source: 'azuredevops',
        ado_project: 'MyProject'
      });
      render(<TechnicalProfileTab repository={repo} />);

      expect(screen.getByText('Azure DevOps Properties')).toBeInTheDocument();
    });

    it('should display ADO project', () => {
      const repo = createMockRepository({ 
        source: 'azuredevops',
        ado_project: 'MyProject'
      });
      render(<TechnicalProfileTab repository={repo} />);

      expect(screen.getByText(/ADO Project/)).toBeInTheDocument();
      expect(screen.getByText('MyProject')).toBeInTheDocument();
    });

    it('should display Git repository type for ADO when ado_is_git is true', () => {
      const repo = createMockRepository({ 
        source: 'azuredevops',
        ado_project: 'MyProject',
        ado_is_git: true
      });
      render(<TechnicalProfileTab repository={repo} />);

      expect(screen.getByText(/Repository Type/)).toBeInTheDocument();
      expect(screen.getByText('Git')).toBeInTheDocument();
    });

    it('should display TFVC warning for non-Git repos', () => {
      const repo = createMockRepository({ 
        source: 'azuredevops',
        ado_project: 'MyProject',
        ado_is_git: false
      });
      render(<TechnicalProfileTab repository={repo} />);

      expect(screen.getByText('TFVC (Requires Conversion)')).toBeInTheDocument();
    });

    it('should display ADO pipeline info when has pipelines', () => {
      const repo = createMockRepository({ 
        source: 'azuredevops',
        ado_project: 'MyProject',
        ado_has_pipelines: true,
        ado_pipeline_count: 10
      });
      render(<TechnicalProfileTab repository={repo} />);

      expect(screen.getByText(/Azure Pipelines/)).toBeInTheDocument();
      expect(screen.getByText(/Total Pipelines/)).toBeInTheDocument();
    });

    it('should display Azure Boards when enabled', () => {
      const repo = createMockRepository({ 
        source: 'azuredevops',
        ado_project: 'MyProject',
        ado_has_boards: true
      });
      render(<TechnicalProfileTab repository={repo} />);

      expect(screen.getByText(/Azure Boards/)).toBeInTheDocument();
    });
  });
});
