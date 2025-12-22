import { describe, it, expect } from 'vitest';
import {
  filtersToSearchParams,
  searchParamsToFilters,
  getRepositoriesUrl,
} from './filters';
import type { RepositoryFilters } from '../types';

describe('filtersToSearchParams', () => {
  it('should return empty params for empty filters', () => {
    const params = filtersToSearchParams({});
    expect(params.toString()).toBe('');
  });

  it('should convert status string to search params', () => {
    const params = filtersToSearchParams({ status: 'pending' });
    expect(params.get('status')).toBe('pending');
  });

  it('should convert status array to comma-separated string', () => {
    const params = filtersToSearchParams({ status: ['pending', 'complete'] });
    expect(params.get('status')).toBe('pending,complete');
  });

  it('should convert batch_id to string', () => {
    const params = filtersToSearchParams({ batch_id: 123 });
    expect(params.get('batch_id')).toBe('123');
  });

  it('should convert source filter', () => {
    const params = filtersToSearchParams({ source: 'github' });
    expect(params.get('source')).toBe('github');
  });

  it('should convert organization string', () => {
    const params = filtersToSearchParams({ organization: 'my-org' });
    expect(params.get('organization')).toBe('my-org');
  });

  it('should convert organization array', () => {
    const params = filtersToSearchParams({ organization: ['org1', 'org2'] });
    expect(params.get('organization')).toBe('org1,org2');
  });

  it('should convert size filters', () => {
    const params = filtersToSearchParams({ min_size: 1000, max_size: 5000 });
    expect(params.get('min_size')).toBe('1000');
    expect(params.get('max_size')).toBe('5000');
  });

  it('should convert boolean filters', () => {
    const params = filtersToSearchParams({
      has_lfs: true,
      has_submodules: false,
      is_archived: true,
    });
    expect(params.get('has_lfs')).toBe('true');
    expect(params.get('has_submodules')).toBe('false');
    expect(params.get('is_archived')).toBe('true');
  });

  it('should convert complexity array', () => {
    const params = filtersToSearchParams({ complexity: ['simple', 'moderate'] });
    expect(params.get('complexity')).toBe('simple,moderate');
  });

  it('should convert complexity string', () => {
    const params = filtersToSearchParams({ complexity: 'complex' });
    expect(params.get('complexity')).toBe('complex');
  });

  it('should convert size_category array', () => {
    const params = filtersToSearchParams({ size_category: ['small', 'medium'] });
    expect(params.get('size_category')).toBe('small,medium');
  });

  it('should convert search and sort_by', () => {
    const params = filtersToSearchParams({ search: 'test', sort_by: 'name' });
    expect(params.get('search')).toBe('test');
    expect(params.get('sort_by')).toBe('name');
  });

  it('should convert visibility', () => {
    const params = filtersToSearchParams({ visibility: 'private' });
    expect(params.get('visibility')).toBe('private');
  });

  it('should convert team filter', () => {
    const params = filtersToSearchParams({ team: 'org/team-slug' });
    expect(params.get('team')).toBe('org/team-slug');
  });

  it('should convert team array', () => {
    const params = filtersToSearchParams({ team: ['org/team1', 'org/team2'] });
    expect(params.get('team')).toBe('org/team1,org/team2');
  });

  it('should convert ADO filters', () => {
    const params = filtersToSearchParams({
      ado_organization: 'ado-org',
      ado_is_git: true,
      ado_has_boards: false,
    });
    expect(params.get('ado_organization')).toBe('ado-org');
    expect(params.get('ado_is_git')).toBe('true');
    expect(params.get('ado_has_boards')).toBe('false');
  });

  it('should convert ado_organization array', () => {
    const params = filtersToSearchParams({ ado_organization: ['ado-org1', 'ado-org2'] });
    expect(params.get('ado_organization')).toBe('ado-org1,ado-org2');
  });

  it('should convert project string', () => {
    const params = filtersToSearchParams({ project: 'my-project' });
    expect(params.get('project')).toBe('my-project');
  });

  it('should convert project array', () => {
    const params = filtersToSearchParams({ project: ['project1', 'project2'] });
    expect(params.get('project')).toBe('project1,project2');
  });
});

describe('searchParamsToFilters', () => {
  it('should return empty filters for empty params', () => {
    const params = new URLSearchParams();
    const filters = searchParamsToFilters(params);
    expect(filters).toEqual({});
  });

  it('should parse status string', () => {
    const params = new URLSearchParams('status=pending');
    const filters = searchParamsToFilters(params);
    expect(filters.status).toBe('pending');
  });

  it('should parse status array from comma-separated string', () => {
    const params = new URLSearchParams('status=pending,complete');
    const filters = searchParamsToFilters(params);
    expect(filters.status).toEqual(['pending', 'complete']);
  });

  it('should parse source', () => {
    const params = new URLSearchParams('source=github');
    const filters = searchParamsToFilters(params);
    expect(filters.source).toBe('github');
  });

  it('should parse batch_id as number', () => {
    const params = new URLSearchParams('batch_id=123');
    const filters = searchParamsToFilters(params);
    expect(filters.batch_id).toBe(123);
  });

  it('should parse size filters as numbers', () => {
    const params = new URLSearchParams('min_size=1000&max_size=5000');
    const filters = searchParamsToFilters(params);
    expect(filters.min_size).toBe(1000);
    expect(filters.max_size).toBe(5000);
  });

  it('should parse boolean filters', () => {
    const params = new URLSearchParams('has_lfs=true&has_submodules=false&is_archived=true');
    const filters = searchParamsToFilters(params);
    expect(filters.has_lfs).toBe(true);
    expect(filters.has_submodules).toBe(false);
    expect(filters.is_archived).toBe(true);
  });

  it('should parse organization string', () => {
    const params = new URLSearchParams('organization=my-org');
    const filters = searchParamsToFilters(params);
    expect(filters.organization).toBe('my-org');
  });

  it('should parse organization array', () => {
    const params = new URLSearchParams('organization=org1,org2');
    const filters = searchParamsToFilters(params);
    expect(filters.organization).toEqual(['org1', 'org2']);
  });

  it('should parse complexity as array', () => {
    const params = new URLSearchParams('complexity=simple');
    const filters = searchParamsToFilters(params);
    expect(filters.complexity).toEqual(['simple']);
  });

  it('should parse complexity array from comma-separated', () => {
    const params = new URLSearchParams('complexity=simple,moderate');
    const filters = searchParamsToFilters(params);
    expect(filters.complexity).toEqual(['simple', 'moderate']);
  });

  it('should parse size_category as array', () => {
    const params = new URLSearchParams('size_category=small,medium');
    const filters = searchParamsToFilters(params);
    expect(filters.size_category).toEqual(['small', 'medium']);
  });

  it('should parse search and sort_by', () => {
    const params = new URLSearchParams('search=test&sort_by=name');
    const filters = searchParamsToFilters(params);
    expect(filters.search).toBe('test');
    expect(filters.sort_by).toBe('name');
  });

  it('should parse visibility', () => {
    const params = new URLSearchParams('visibility=private');
    const filters = searchParamsToFilters(params);
    expect(filters.visibility).toBe('private');
  });

  it('should parse team filter', () => {
    const params = new URLSearchParams('team=org/team-slug');
    const filters = searchParamsToFilters(params);
    expect(filters.team).toBe('org/team-slug');
  });

  it('should parse team array', () => {
    const params = new URLSearchParams('team=org/team1,org/team2');
    const filters = searchParamsToFilters(params);
    expect(filters.team).toEqual(['org/team1', 'org/team2']);
  });

  it('should parse ADO filters', () => {
    const params = new URLSearchParams('ado_organization=ado-org&ado_is_git=true&ado_has_boards=false');
    const filters = searchParamsToFilters(params);
    expect(filters.ado_organization).toBe('ado-org');
    expect(filters.ado_is_git).toBe(true);
    expect(filters.ado_has_boards).toBe(false);
  });
});

describe('getRepositoriesUrl', () => {
  it('should return base path for empty filters', () => {
    const url = getRepositoriesUrl({});
    expect(url).toBe('/repositories');
  });

  it('should append query string for filters', () => {
    const url = getRepositoriesUrl({ status: 'pending' });
    expect(url).toBe('/repositories?status=pending');
  });

  it('should handle multiple filters', () => {
    const url = getRepositoriesUrl({ status: 'pending', source: 'github' });
    expect(url).toContain('/repositories?');
    expect(url).toContain('status=pending');
    expect(url).toContain('source=github');
  });
});

describe('roundtrip conversion', () => {
  it('should preserve filters through conversion', () => {
    const originalFilters: RepositoryFilters = {
      status: ['pending', 'complete'],
      source: 'github',
      organization: 'my-org',
      has_lfs: true,
      min_size: 1000,
      complexity: ['simple', 'moderate'],
      search: 'test',
    };

    const params = filtersToSearchParams(originalFilters);
    const parsed = searchParamsToFilters(params);

    expect(parsed.status).toEqual(originalFilters.status);
    expect(parsed.source).toBe(originalFilters.source);
    expect(parsed.organization).toBe(originalFilters.organization);
    expect(parsed.has_lfs).toBe(originalFilters.has_lfs);
    expect(parsed.min_size).toBe(originalFilters.min_size);
    expect(parsed.complexity).toEqual(originalFilters.complexity);
    expect(parsed.search).toBe(originalFilters.search);
  });
});

