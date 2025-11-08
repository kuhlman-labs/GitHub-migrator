import type { RepositoryFilters } from '../types';

/**
 * Convert a RepositoryFilters object to URL search parameters
 */
export function filtersToSearchParams(filters: RepositoryFilters): URLSearchParams {
  const params = new URLSearchParams();

  // Handle each filter type
  if (filters.status) params.set('status', filters.status);
  if (filters.batch_id) params.set('batch_id', filters.batch_id.toString());
  if (filters.source) params.set('source', filters.source);
  
  // Handle organization (can be string or array)
  if (filters.organization) {
    if (Array.isArray(filters.organization)) {
      params.set('organization', filters.organization.join(','));
    } else {
      params.set('organization', filters.organization);
    }
  }

  // Handle size filters
  if (filters.min_size !== undefined) params.set('min_size', filters.min_size.toString());
  if (filters.max_size !== undefined) params.set('max_size', filters.max_size.toString());

  // Handle boolean feature filters
  if (filters.has_lfs !== undefined) params.set('has_lfs', filters.has_lfs.toString());
  if (filters.has_submodules !== undefined) params.set('has_submodules', filters.has_submodules.toString());
  if (filters.has_large_files !== undefined) params.set('has_large_files', filters.has_large_files.toString());
  if (filters.has_actions !== undefined) params.set('has_actions', filters.has_actions.toString());
  if (filters.has_wiki !== undefined) params.set('has_wiki', filters.has_wiki.toString());
  if (filters.has_pages !== undefined) params.set('has_pages', filters.has_pages.toString());
  if (filters.has_discussions !== undefined) params.set('has_discussions', filters.has_discussions.toString());
  if (filters.has_projects !== undefined) params.set('has_projects', filters.has_projects.toString());
  if (filters.has_packages !== undefined) params.set('has_packages', filters.has_packages.toString());
  if (filters.has_branch_protections !== undefined) params.set('has_branch_protections', filters.has_branch_protections.toString());
  if (filters.has_rulesets !== undefined) params.set('has_rulesets', filters.has_rulesets.toString());
  if (filters.is_archived !== undefined) params.set('is_archived', filters.is_archived.toString());
  if (filters.is_fork !== undefined) params.set('is_fork', filters.is_fork.toString());
  if (filters.has_code_scanning !== undefined) params.set('has_code_scanning', filters.has_code_scanning.toString());
  if (filters.has_dependabot !== undefined) params.set('has_dependabot', filters.has_dependabot.toString());
  if (filters.has_secret_scanning !== undefined) params.set('has_secret_scanning', filters.has_secret_scanning.toString());
  if (filters.has_codeowners !== undefined) params.set('has_codeowners', filters.has_codeowners.toString());
  if (filters.has_self_hosted_runners !== undefined) params.set('has_self_hosted_runners', filters.has_self_hosted_runners.toString());
  if (filters.has_release_assets !== undefined) params.set('has_release_assets', filters.has_release_assets.toString());
  if (filters.has_webhooks !== undefined) params.set('has_webhooks', filters.has_webhooks.toString());
  if (filters.visibility) params.set('visibility', filters.visibility);

  // Handle complexity and size category (arrays)
  if (filters.complexity) {
    if (Array.isArray(filters.complexity)) {
      params.set('complexity', filters.complexity.join(','));
    } else {
      params.set('complexity', filters.complexity);
    }
  }
  if (filters.size_category) {
    if (Array.isArray(filters.size_category)) {
      params.set('size_category', filters.size_category.join(','));
    } else {
      params.set('size_category', filters.size_category);
    }
  }

  // Handle search and sorting
  if (filters.search) params.set('search', filters.search);
  if (filters.sort_by) params.set('sort_by', filters.sort_by);
  if (filters.available_for_batch !== undefined) params.set('available_for_batch', filters.available_for_batch.toString());

  return params;
}

/**
 * Parse URL search parameters into a RepositoryFilters object
 */
export function searchParamsToFilters(searchParams: URLSearchParams): RepositoryFilters {
  const filters: RepositoryFilters = {};

  // Simple string filters
  const status = searchParams.get('status');
  if (status) filters.status = status;

  const source = searchParams.get('source');
  if (source) filters.source = source;

  const search = searchParams.get('search');
  if (search) filters.search = search;

  const sortBy = searchParams.get('sort_by');
  if (sortBy) filters.sort_by = sortBy as 'name' | 'size' | 'org' | 'updated';

  // Number filters
  const batchId = searchParams.get('batch_id');
  if (batchId) filters.batch_id = parseInt(batchId, 10);

  const minSize = searchParams.get('min_size');
  if (minSize) filters.min_size = parseInt(minSize, 10);

  const maxSize = searchParams.get('max_size');
  if (maxSize) filters.max_size = parseInt(maxSize, 10);

  // Boolean filters
  const hasLfs = searchParams.get('has_lfs');
  if (hasLfs) filters.has_lfs = hasLfs === 'true';

  const hasSubmodules = searchParams.get('has_submodules');
  if (hasSubmodules) filters.has_submodules = hasSubmodules === 'true';

  const hasLargeFiles = searchParams.get('has_large_files');
  if (hasLargeFiles) filters.has_large_files = hasLargeFiles === 'true';

  const hasActions = searchParams.get('has_actions');
  if (hasActions) filters.has_actions = hasActions === 'true';

  const hasWiki = searchParams.get('has_wiki');
  if (hasWiki) filters.has_wiki = hasWiki === 'true';

  const hasPages = searchParams.get('has_pages');
  if (hasPages) filters.has_pages = hasPages === 'true';

  const hasDiscussions = searchParams.get('has_discussions');
  if (hasDiscussions) filters.has_discussions = hasDiscussions === 'true';

  const hasProjects = searchParams.get('has_projects');
  if (hasProjects) filters.has_projects = hasProjects === 'true';

  const hasPackages = searchParams.get('has_packages');
  if (hasPackages) filters.has_packages = hasPackages === 'true';

  const hasBranchProtections = searchParams.get('has_branch_protections');
  if (hasBranchProtections) filters.has_branch_protections = hasBranchProtections === 'true';

  const hasRulesets = searchParams.get('has_rulesets');
  if (hasRulesets) filters.has_rulesets = hasRulesets === 'true';

  const isArchived = searchParams.get('is_archived');
  if (isArchived) filters.is_archived = isArchived === 'true';

  const isFork = searchParams.get('is_fork');
  if (isFork) filters.is_fork = isFork === 'true';

  const hasCodeScanning = searchParams.get('has_code_scanning');
  if (hasCodeScanning) filters.has_code_scanning = hasCodeScanning === 'true';

  const hasDependabot = searchParams.get('has_dependabot');
  if (hasDependabot) filters.has_dependabot = hasDependabot === 'true';

  const hasSecretScanning = searchParams.get('has_secret_scanning');
  if (hasSecretScanning) filters.has_secret_scanning = hasSecretScanning === 'true';

  const hasCodeowners = searchParams.get('has_codeowners');
  if (hasCodeowners) filters.has_codeowners = hasCodeowners === 'true';

  const hasSelfHostedRunners = searchParams.get('has_self_hosted_runners');
  if (hasSelfHostedRunners) filters.has_self_hosted_runners = hasSelfHostedRunners === 'true';

  const hasReleaseAssets = searchParams.get('has_release_assets');
  if (hasReleaseAssets) filters.has_release_assets = hasReleaseAssets === 'true';

  const hasWebhooks = searchParams.get('has_webhooks');
  if (hasWebhooks) filters.has_webhooks = hasWebhooks === 'true';

  const availableForBatch = searchParams.get('available_for_batch');
  if (availableForBatch) filters.available_for_batch = availableForBatch === 'true';

  // Visibility filter
  const visibility = searchParams.get('visibility');
  if (visibility) filters.visibility = visibility as 'public' | 'private' | 'internal';

  // Array filters
  const organization = searchParams.get('organization');
  if (organization) {
    filters.organization = organization.includes(',') ? organization.split(',') : organization;
  }

  const complexity = searchParams.get('complexity');
  if (complexity) {
    filters.complexity = complexity.includes(',') ? complexity.split(',') : [complexity];
  }

  const sizeCategory = searchParams.get('size_category');
  if (sizeCategory) {
    filters.size_category = sizeCategory.includes(',') ? sizeCategory.split(',') : [sizeCategory];
  }

  return filters;
}

/**
 * Generate a navigation URL with filters
 */
export function getRepositoriesUrl(filters: Partial<RepositoryFilters>): string {
  const params = filtersToSearchParams(filters as RepositoryFilters);
  const queryString = params.toString();
  return `/repositories${queryString ? `?${queryString}` : ''}`;
}

