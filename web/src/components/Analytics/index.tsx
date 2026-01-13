import { useState, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { BarChart, Bar, PieChart, Pie, Cell, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';
import { UnderlineNav, ProgressBar, ActionMenu, ActionList } from '@primer/react';
import { ChevronRightIcon, DownloadIcon, TriangleDownIcon } from '@primer/octicons-react';
import { BorderedButton } from '../common/buttons';
import { LoadingSpinner } from '../common/LoadingSpinner';
import { RefreshIndicator } from '../common/RefreshIndicator';
import { formatDuration } from '../../utils/format';
import { useAnalytics } from '../../hooks/useQueries';
import { FilterBar } from './FilterBar';
import { MigrationTrendChart } from './MigrationTrendChart';
import { ComplexityChart } from './ComplexityChart';
import { KPICard } from './KPICard';
import { getRepositoriesUrl } from '../../utils/filters';
import { api } from '../../services/api';
import { useToast } from '../../contexts/ToastContext';
import { useSourceContext } from '../../contexts/SourceContext';
import { handleApiError } from '../../utils/errorHandler';

const STATUS_COLORS: Record<string, string> = {
  pending: '#656D76',
  in_progress: '#0969DA',
  migration_complete: '#1A7F37',
  complete: '#1A7F37',
  failed: '#D1242F',
  dry_run_failed: '#D1242F',
  migration_failed: '#D1242F',
  rolled_back: '#FB8500',
  dry_run_complete: '#8250DF',
  wont_migrate: '#6B7280',
};

type AnalyticsTab = 'discovery' | 'migration';

export function Analytics() {
  const { showError } = useToast();
  const navigate = useNavigate();
  const [selectedOrganization, setSelectedOrganization] = useState('');
  const [selectedProject, setSelectedProject] = useState('');
  const [selectedBatch, setSelectedBatch] = useState('');
  const [activeTab, setActiveTab] = useState<AnalyticsTab>('discovery');

  // Derive source context for display
  // isAllSourcesMode from context correctly handles single-source setups (always false for single source)
  const { activeSource, isAllSourcesMode, hasMultipleSources } = useSourceContext();
  // For backwards compatibility with existing source-specific features
  const sourceType = activeSource?.type || 'github';

  const { data: analytics, isLoading, isFetching } = useAnalytics({
    organization: selectedOrganization || undefined,
    project: selectedProject || undefined,
    batch_id: selectedBatch || undefined,
    source_id: activeSource?.id,
  });

  // Calculate days until completion (must be before early returns to satisfy rules of hooks)
  const daysUntilCompletion = useMemo(() => {
    if (!analytics || !analytics.estimated_completion_date) return null;
    const estimatedDate = new Date(analytics.estimated_completion_date).getTime();
    const today = new Date().setHours(0, 0, 0, 0); // Use Date constructor instead of Date.now()
    return Math.ceil((estimatedDate - today) / (1000 * 60 * 60 * 24));
  }, [analytics]);

  if (isLoading) return <LoadingSpinner />;
  if (!analytics) return <div className="text-center py-12 text-gh-text-secondary">No analytics data available</div>;

  // Prepare chart data with GitHub colors
  const statusChartData = Object.entries(analytics.status_breakdown).map(([status, count]) => ({
    name: status.replace(/_/g, ' '),
    value: count,
    fill: STATUS_COLORS[status] || '#656D76',
  }));

  const completionRate = analytics.total_repositories > 0
    ? Math.round((analytics.migrated_count / analytics.total_repositories) * 100)
    : 0;

  const progressData = [
    { name: 'Migrated', value: analytics.migrated_count, fill: '#1A7F37' },
    { name: 'In Progress', value: analytics.in_progress_count, fill: '#0969DA' },
    { name: 'Failed', value: analytics.failed_count, fill: '#D1242F' },
    { name: 'Pending', value: analytics.pending_count, fill: '#656D76' },
  ].filter(item => item.value > 0);

  const sizeCategories: Record<string, string> = {
    small: 'Small (<100MB)',
    medium: 'Medium (100MB-1GB)',
    large: 'Large (1GB-5GB)',
    very_large: 'Very Large (>5GB)',
    unknown: 'Unknown',
  };

  const sizeColors: Record<string, string> = {
    small: '#10B981',      // Match 'simple' complexity - green
    medium: '#F59E0B',     // Match 'medium' complexity - orange
    large: '#F97316',      // Match 'complex' complexity - dark orange
    very_large: '#EF4444', // Match 'very_complex' complexity - red
    unknown: '#656D76',
  };

  // Calculate high complexity count
  const highComplexityCount = analytics.complexity_distribution
    ?.filter(d => d.category === 'complex' || d.category === 'very_complex')
    .reduce((sum, d) => sum + d.count, 0) || 0;

  // Export functions
  const handleExport = async (reportType: 'executive' | 'discovery', format: 'csv' | 'json') => {
    if (!analytics) {
      alert('No analytics data to export');
      return;
    }

    try {
      const filters = {
        organization: selectedOrganization || undefined,
        project: selectedProject || undefined,
        batch_id: selectedBatch || undefined,
        source_id: activeSource?.id,
      };

      let blob: Blob;
      let filename: string;
      const sourceSuffix = activeSource ? `_${activeSource.name.replace(/\s+/g, '_')}` : '';

      if (reportType === 'executive') {
        blob = await api.exportExecutiveReport(format, filters);
        filename = `executive-migration-report${sourceSuffix}.${format}`;
      } else {
        blob = await api.exportDetailedDiscoveryReport(format, filters);
        filename = `detailed-discovery-report${sourceSuffix}.${format}`;
      }

      const url = window.URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = filename;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      window.URL.revokeObjectURL(url);
    } catch (error) {
      handleApiError(error, showError, 'Failed to export report');
    }
  };

  return (
    <div className="relative">
      <RefreshIndicator isRefreshing={isFetching && !isLoading} />
      <div className="flex items-start justify-between mb-8">
        <div>
          <h1 className="text-2xl font-semibold" style={{ color: 'var(--fgColor-default)' }}>Analytics Dashboard</h1>
          <p className="text-sm mt-1" style={{ color: 'var(--fgColor-muted)' }}>
            Migration metrics and insights for reporting and planning
          </p>
        </div>
        
        {/* Export Button with Dropdown */}
        <ActionMenu>
          <ActionMenu.Anchor>
            <BorderedButton
              disabled={!analytics}
              leadingVisual={DownloadIcon}
              trailingAction={TriangleDownIcon}
            >
              Export
            </BorderedButton>
          </ActionMenu.Anchor>
          <ActionMenu.Overlay>
            <ActionList>
              <ActionList.Group title="Executive Report">
                <ActionList.Item onSelect={() => handleExport('executive', 'csv')}>
                  Export as CSV
                </ActionList.Item>
                <ActionList.Item onSelect={() => handleExport('executive', 'json')}>
                  Export as JSON
                </ActionList.Item>
              </ActionList.Group>
              <ActionList.Divider />
              <ActionList.Group title="Discovery Report">
                <ActionList.Item onSelect={() => handleExport('discovery', 'csv')}>
                  Export as CSV
                </ActionList.Item>
                <ActionList.Item onSelect={() => handleExport('discovery', 'json')}>
                  Export as JSON
                </ActionList.Item>
              </ActionList.Group>
            </ActionList>
          </ActionMenu.Overlay>
        </ActionMenu>
      </div>

      {/* Filter Bar */}
      <FilterBar
        selectedOrganization={selectedOrganization}
        selectedProject={selectedProject}
        selectedBatch={selectedBatch}
        onOrganizationChange={setSelectedOrganization}
        onProjectChange={setSelectedProject}
        onBatchChange={setSelectedBatch}
        sourceType={sourceType}
        isAllSourcesMode={isAllSourcesMode}
        sourceId={activeSource?.id}
      />

      {/* Tabs Navigation */}
      <div className="mb-8">
        <UnderlineNav aria-label="Analytics sections">
          <UnderlineNav.Item
            aria-current={activeTab === 'discovery' ? 'page' : undefined}
            onSelect={() => setActiveTab('discovery')}
          >
            Discovery Analytics
          </UnderlineNav.Item>
          <UnderlineNav.Item
            aria-current={activeTab === 'migration' ? 'page' : undefined}
            onSelect={() => setActiveTab('migration')}
          >
            Migration Analytics
          </UnderlineNav.Item>
        </UnderlineNav>
      </div>

      {/* SECTION 1: DISCOVERY ANALYTICS */}
      {activeTab === 'discovery' && (
      <section className="mb-12">
        <div className="border-l-4 pl-4 mb-6" style={{ borderColor: 'var(--accent-emphasis)' }}>
          <h2 className="text-2xl font-light" style={{ color: 'var(--fgColor-default)' }}>Discovery Analytics</h2>
          <p className="text-sm mt-1" style={{ color: 'var(--fgColor-muted)' }}>
            Source environment overview to drive batch planning decisions
          </p>
        </div>

        {/* Discovery Summary Cards */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
          <KPICard
            title="Total Repositories"
            value={analytics.total_repositories}
            color="blue"
            tooltip={
              isAllSourcesMode && hasMultipleSources
                ? "Total number of repositories discovered across all sources"
                : sourceType === 'azuredevops'
                  ? "Total number of repositories discovered in Azure DevOps"
                  : "Total number of repositories discovered in GitHub"
            }
          />
          <KPICard
            title={
              isAllSourcesMode && hasMultipleSources 
                ? 'Source Groups' 
                : sourceType === 'azuredevops' 
                  ? 'Projects' 
                  : 'Organizations'
            }
            value={
              isAllSourcesMode && hasMultipleSources
                ? analytics.organization_stats?.length || 0
                : sourceType === 'azuredevops'
                  ? analytics.project_stats?.length || 0
                  : analytics.organization_stats?.length || 0
            }
            color="purple"
            subtitle={
              isAllSourcesMode 
                ? 'Across all sources' 
                : activeSource?.name || 'Source groups'
            }
            tooltip={
              isAllSourcesMode && hasMultipleSources
                ? "Number of source groups (organizations, projects, etc.) with repositories"
                : sourceType === 'azuredevops'
                  ? "Number of Azure DevOps projects with repositories"
                  : "Number of GitHub organizations with repositories"
            }
          />
          <KPICard
            title="High Complexity"
            value={highComplexityCount}
            color="yellow"
            subtitle={`${analytics.total_repositories > 0 ? Math.round((highComplexityCount / analytics.total_repositories) * 100) : 0}% of total`}
            onClick={() => navigate(getRepositoriesUrl({ complexity: ['complex', 'very_complex'] }))}
            tooltip="Repositories marked as complex or very complex requiring special attention"
          />
          <KPICard
            title="Features Detected"
            value={analytics.feature_stats ? Object.entries(analytics.feature_stats).filter(([key, value]) => key !== 'total_repositories' && typeof value === 'number' && value > 0).length : 0}
            color="green"
            tooltip="Number of different features detected across all repositories (LFS, Actions, Wikis, etc.)"
          />
        </div>

        {/* Complexity and Size Distribution Charts */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
          <ComplexityChart 
            data={analytics.complexity_distribution || []} 
            source={isAllSourcesMode ? 'all' : sourceType} 
          />
          
          {/* Size Distribution */}
          {analytics.size_distribution && analytics.size_distribution.length > 0 && (
            <div className="rounded-lg shadow-sm p-6" style={{ backgroundColor: 'var(--bgColor-default)' }}>
              <h3 className="text-lg font-medium mb-4" style={{ color: 'var(--fgColor-default)' }}>Repository Size Distribution</h3>
              <p className="text-sm mb-4" style={{ color: 'var(--fgColor-muted)' }}>
                Distribution of repositories by disk size, helping identify storage requirements and migration capacity planning needs.
              </p>
              <ResponsiveContainer width="100%" height={250}>
                <PieChart>
                  <Pie
                    data={analytics.size_distribution.map(item => ({
                      name: sizeCategories[item.category] || item.category,
                      value: item.count,
                      fill: sizeColors[item.category] || '#9CA3AF',
                      category: item.category,
                    }))}
                    cx="50%"
                    cy="50%"
                    labelLine={false}
                    label={false}
                    outerRadius={80}
                    dataKey="value"
                    // eslint-disable-next-line @typescript-eslint/no-explicit-any
                    onClick={(data: any) => {
                      if (data && data.payload && data.payload.category) {
                        navigate(getRepositoriesUrl({ size_category: [data.payload.category] }));
                      }
                    }}
                    cursor="pointer"
                  >
                    {analytics.size_distribution.map((entry, index) => (
                      <Cell key={`cell-${index}`} fill={sizeColors[entry.category] || '#9CA3AF'} />
                    ))}
                  </Pie>
                  <Tooltip 
                    // eslint-disable-next-line @typescript-eslint/no-explicit-any
                    formatter={(value: number, _name: string, props: any) => [
                      `${value} repositories`,
                      props.payload?.name || ''
                    ]}
                    contentStyle={{
                      backgroundColor: 'rgba(27, 31, 36, 0.95)',
                      border: '1px solid rgba(255, 255, 255, 0.1)',
                      borderRadius: '6px',
                      color: '#ffffff',
                      padding: '8px 12px'
                    }}
                    labelStyle={{ color: '#ffffff', fontWeight: 600 }}
                    itemStyle={{ color: '#ffffff' }}
                  />
                </PieChart>
              </ResponsiveContainer>
              
              {/* Legend */}
              <div className="mt-4 grid grid-cols-2 gap-2">
                {analytics.size_distribution.map((item) => (
                  <button
                    key={item.category}
                    onClick={() => navigate(getRepositoriesUrl({ size_category: [item.category] }))}
                    className="flex items-center gap-2 p-2 rounded hover:bg-gh-info-bg transition-colors cursor-pointer text-left"
                  >
                    <div 
                      className="w-4 h-4 rounded flex-shrink-0" 
                      style={{ backgroundColor: sizeColors[item.category] || '#9CA3AF' }}
                    />
                    <span className="text-sm truncate" style={{ color: 'var(--fgColor-default)' }}>
                      {sizeCategories[item.category] || item.category}: {item.count}
                    </span>
                  </button>
                ))}
              </div>
            </div>
          )}
        </div>

        {/* Source Group Breakdown and Feature Stats */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
          {/* Source Group Breakdown */}
          {(() => {
            // When viewing all sources, always use organization_stats (includes ADO orgs, not projects)
            // Only use project_stats when a specific ADO source is selected
            const stats = !isAllSourcesMode && sourceType === 'azuredevops' && analytics.project_stats 
              ? analytics.project_stats 
              : analytics.organization_stats;
            
            if (!stats || stats.length === 0) return null;

            // Determine terminology based on source context
            const groupLabel = isAllSourcesMode && hasMultipleSources 
              ? 'Source Group' 
              : sourceType === 'azuredevops' 
                ? 'Project' 
                : 'Organization';
            
            const sectionTitle = isAllSourcesMode && hasMultipleSources 
              ? 'Source Group Breakdown' 
              : sourceType === 'azuredevops' 
                ? 'Project Breakdown' 
                : 'Organization Breakdown';
            
            const sectionDescription = isAllSourcesMode && hasMultipleSources
              ? 'Repository count and distribution across source groups, useful for workload allocation and team coordination.'
              : sourceType === 'azuredevops'
                ? 'Repository count and distribution across Azure DevOps projects, useful for workload allocation and team coordination.'
                : 'Repository count and distribution across GitHub organizations, useful for workload allocation and team coordination.';
            
            return (
              <div className="rounded-lg shadow-sm p-6" style={{ backgroundColor: 'var(--bgColor-default)' }}>
                <h3 className="text-lg font-medium mb-4" style={{ color: 'var(--fgColor-default)' }}>
                  {sectionTitle}
                </h3>
                <p className="text-sm mb-4" style={{ color: 'var(--fgColor-muted)' }}>
                  {sectionDescription}
                </p>
                <div className="overflow-x-auto">
                  <table className="min-w-full divide-y" style={{ borderColor: 'var(--borderColor-muted)' }}>
                    <thead style={{ backgroundColor: 'var(--bgColor-muted)' }}>
                      <tr>
                        <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider" style={{ color: 'var(--fgColor-muted)' }}>
                          {groupLabel}
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider" style={{ color: 'var(--fgColor-muted)' }}>
                          Repositories
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider" style={{ color: 'var(--fgColor-muted)' }}>
                          Percentage
                        </th>
                      </tr>
                    </thead>
                    <tbody className="divide-y" style={{ backgroundColor: 'var(--bgColor-default)', borderColor: 'var(--borderColor-muted)' }}>
                      {stats
                        .sort((a, b) => b.total_repos - a.total_repos)
                        .map((org) => {
                          const percentage = analytics.total_repositories > 0
                            ? ((org.total_repos / analytics.total_repositories) * 100).toFixed(1)
                            : '0.0';
                          return (
                            <tr key={org.organization} className="hover:opacity-80 transition-opacity">
                              <td className="px-6 py-4 whitespace-nowrap text-sm font-medium" style={{ color: 'var(--fgColor-default)' }}>
                                {org.organization}
                              </td>
                              <td className="px-6 py-4 whitespace-nowrap text-sm" style={{ color: 'var(--fgColor-default)' }}>
                                {org.total_repos}
                              </td>
                              <td className="px-6 py-4 whitespace-nowrap text-sm" style={{ color: 'var(--fgColor-default)' }}>
                                {percentage}%
                              </td>
                            </tr>
                          );
                        })}
                    </tbody>
                  </table>
                </div>
              </div>
            );
          })()}

          {/* Feature Stats */}
          {analytics.feature_stats && (() => {
            const featureStats = analytics.feature_stats;
            
            // Separate features by source type
            const adoFeatures = [
              { label: "Azure Boards", count: featureStats.ado_has_boards, filter: { ado_has_boards: true } },
              { label: "Azure Pipelines", count: featureStats.ado_has_pipelines, filter: { ado_has_pipelines: true } },
              { label: "YAML Pipelines", count: featureStats.ado_has_yaml_pipelines, filter: { ado_yaml_pipeline_count: '> 0' } },
              { label: "Classic Pipelines", count: featureStats.ado_has_classic_pipelines, filter: { ado_classic_pipeline_count: '> 0' } },
              { label: "GHAS (Azure DevOps)", count: featureStats.ado_has_ghas, filter: { ado_has_ghas: true } },
              { label: "TFVC Repositories", count: featureStats.ado_tfvc_count, filter: { ado_is_git: false } },
              { label: "Pull Requests", count: featureStats.ado_has_pull_requests, filter: { ado_pull_request_count: '> 0' } },
              { label: "Work Items", count: featureStats.ado_has_work_items, filter: { ado_work_item_count: '> 0' } },
              { label: "Branch Policies", count: featureStats.ado_has_branch_policies, filter: { ado_branch_policy_count: '> 0' } },
              { label: "Wikis (ADO)", count: featureStats.ado_has_wiki, filter: { ado_has_wiki: true } },
              { label: "Test Plans", count: featureStats.ado_has_test_plans, filter: { ado_test_plan_count: '> 0' } },
              { label: "Package Feeds", count: featureStats.ado_has_package_feeds, filter: { ado_package_feed_count: '> 0' } },
              { label: "Service Hooks", count: featureStats.ado_has_service_hooks, filter: { ado_service_hook_count: '> 0' } },
            ];
            
            const githubFeatures = [
              { label: "Archived", count: featureStats.is_archived, filter: { is_archived: true } },
              { label: "Forked Repositories", count: featureStats.is_fork, filter: { is_fork: true } },
              { label: "LFS", count: featureStats.has_lfs, filter: { has_lfs: true } },
              { label: "Submodules", count: featureStats.has_submodules, filter: { has_submodules: true } },
              { label: "Large Files (>100MB)", count: featureStats.has_large_files, filter: { has_large_files: true } },
              { label: "GitHub Actions", count: featureStats.has_actions, filter: { has_actions: true } },
              { label: "Wikis", count: featureStats.has_wiki, filter: { has_wiki: true } },
              { label: "Pages", count: featureStats.has_pages, filter: { has_pages: true } },
              { label: "Discussions", count: featureStats.has_discussions, filter: { has_discussions: true } },
              { label: "Projects", count: featureStats.has_projects, filter: { has_projects: true } },
              { label: "Packages", count: featureStats.has_packages, filter: { has_packages: true } },
              { label: "Branch Protections", count: featureStats.has_branch_protections, filter: { has_branch_protections: true } },
              { label: "Rulesets", count: featureStats.has_rulesets, filter: { has_rulesets: true } },
              { label: "Code Scanning", count: featureStats.has_code_scanning, filter: { has_code_scanning: true } },
              { label: "Dependabot", count: featureStats.has_dependabot, filter: { has_dependabot: true } },
              { label: "Secret Scanning", count: featureStats.has_secret_scanning, filter: { has_secret_scanning: true } },
              { label: "CODEOWNERS", count: featureStats.has_codeowners, filter: { has_codeowners: true } },
              { label: "Self-Hosted Runners", count: featureStats.has_self_hosted_runners, filter: { has_self_hosted_runners: true } },
              { label: "Release Assets", count: featureStats.has_release_assets, filter: { has_release_assets: true } },
              { label: "Webhooks", count: featureStats.has_webhooks, filter: { has_webhooks: true } },
              { label: "Environments", count: featureStats.has_environments, filter: { has_environments: true } },
              { label: "Secrets", count: featureStats.has_secrets, filter: { has_secrets: true } },
              { label: "Variables", count: featureStats.has_variables, filter: { has_variables: true } },
            ];
            
            // Select features based on source type - show combined when viewing all sources
            const features = isAllSourcesMode && hasMultipleSources
              ? [...githubFeatures, ...adoFeatures].filter(feature => feature.count && feature.count > 0)
              : (sourceType === 'azuredevops' ? adoFeatures : githubFeatures)
                  .filter(feature => feature.count && feature.count > 0);

            const featureDescription = isAllSourcesMode && hasMultipleSources
              ? 'Features detected across repositories that may require special migration handling, including CI/CD workflows, security configurations, and advanced settings.'
              : sourceType === 'azuredevops'
                ? 'Azure DevOps features detected including pipelines, boards, wikis, and other project artifacts that may require special migration handling.'
                : 'GitHub features detected including Actions workflows, security configurations, branch protections, and other settings that may require special migration handling.';

            return features.length > 0 ? (
              <div className="rounded-lg shadow-sm p-6" style={{ backgroundColor: 'var(--bgColor-default)' }}>
                <h3 className="text-lg font-medium mb-4" style={{ color: 'var(--fgColor-default)' }}>Feature Usage Statistics</h3>
                <p className="text-sm mb-4" style={{ color: 'var(--fgColor-muted)' }}>{featureDescription}</p>
                <div className="space-y-1">
                  {features.map(feature => (
                    <FeatureStat 
                      key={feature.label}
                      label={feature.label} 
                      count={feature.count} 
                      total={featureStats.total_repositories}
                      onClick={() => navigate(getRepositoriesUrl(feature.filter))}
                    />
                  ))}
                </div>
              </div>
            ) : null;
          })()}
        </div>
      </section>
      )}

      {/* SECTION 2: MIGRATION ANALYTICS */}
      {activeTab === 'migration' && (
      <section>
        <div className="border-l-4 pl-4 mb-6" style={{ borderColor: 'var(--success-emphasis)' }}>
          <h2 className="text-2xl font-light" style={{ color: 'var(--fgColor-default)' }}>Migration Analytics</h2>
          <p className="text-sm mt-1" style={{ color: 'var(--fgColor-muted)' }}>
            Migration progress and performance for executive reporting
          </p>
        </div>

        {/* KPI Cards */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
          <KPICard
            title="Completion Rate"
            value={`${completionRate}%`}
            subtitle={`${analytics.migrated_count} of ${analytics.total_repositories}`}
            color="green"
            tooltip="Percentage of repositories successfully migrated"
          />
          <KPICard
            title="Migration Velocity"
            value={analytics.migration_velocity ? `${analytics.migration_velocity.repos_per_week.toFixed(1)}` : '0'}
            subtitle="repos/week"
            color="blue"
            tooltip="Average number of repositories migrated per week over the last 30 days"
          />
          <KPICard
            title="Success Rate"
            value={analytics.success_rate ? `${analytics.success_rate.toFixed(1)}%` : '0%'}
            subtitle="of attempted migrations"
            color="purple"
            tooltip="Percentage of successful migrations vs. failed migrations"
          />
          <KPICard
            title="Est. Completion"
            value={analytics.estimated_completion_date ? new Date(analytics.estimated_completion_date).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' }) : 'TBD'}
            subtitle={daysUntilCompletion !== null ? `${daysUntilCompletion} days` : ''}
            color="yellow"
            tooltip="Estimated completion date based on current velocity"
          />
        </div>

        {/* Migration Trend and Status Charts */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
          <MigrationTrendChart data={analytics.migration_time_series || []} />

          {/* Status Breakdown Pie Chart */}
          <div className="rounded-lg shadow-sm p-6" style={{ backgroundColor: 'var(--bgColor-default)' }}>
            <h2 className="text-lg font-medium mb-4" style={{ color: 'var(--fgColor-default)' }}>Current Status Distribution</h2>
            <p className="text-sm mb-4" style={{ color: 'var(--fgColor-muted)' }}>Real-time snapshot of repository migration states, showing pending, in-progress, completed, and failed migrations.</p>
            <ResponsiveContainer width="100%" height={250}>
              <PieChart>
                <Pie
                  data={progressData}
                  cx="50%"
                  cy="50%"
                  labelLine={false}
                  label={false}
                  outerRadius={80}
                  dataKey="value"
                  // eslint-disable-next-line @typescript-eslint/no-explicit-any
                  onClick={(data: any) => {
                    if (data && data.payload && data.payload.name) {
                      const statusMap: Record<string, string[]> = {
                        'Migrated': ['complete', 'migration_complete'],
                        'In Progress': ['pre_migration', 'archive_generating', 'queued_for_migration', 'migrating_content', 'post_migration'],
                        'Failed': ['dry_run_failed', 'migration_failed', 'rolled_back'],
                        'Pending': ['pending', 'dry_run_queued', 'dry_run_in_progress', 'dry_run_complete'],
                      };
                      const statuses = statusMap[data.payload.name];
                      if (statuses) {
                        navigate(getRepositoriesUrl({ status: statuses }));
                      }
                    }
                  }}
                  cursor="pointer"
                >
                  {progressData.map((entry, index) => (
                    <Cell key={`cell-${index}`} fill={entry.fill} />
                  ))}
                </Pie>
                <Tooltip 
                  formatter={(value: number, name: string) => [`${value} repositories`, name]}
                  contentStyle={{
                    backgroundColor: 'rgba(27, 31, 36, 0.95)',
                    border: '1px solid rgba(255, 255, 255, 0.1)',
                    borderRadius: '6px',
                    color: '#ffffff',
                    padding: '8px 12px'
                  }}
                  labelStyle={{ 
                    color: '#ffffff',
                    fontWeight: 600,
                    marginBottom: '4px'
                  }}
                  itemStyle={{
                    color: '#ffffff',
                    padding: '4px 0'
                  }}
                />
              </PieChart>
            </ResponsiveContainer>
            
            {/* Legend */}
            <div className="mt-4 grid grid-cols-2 gap-2">
              {progressData.map((item) => {
                const statusMap: Record<string, string[]> = {
                  'Migrated': ['complete', 'migration_complete'],
                  'In Progress': ['pre_migration', 'archive_generating', 'queued_for_migration', 'migrating_content', 'post_migration'],
                  'Failed': ['dry_run_failed', 'migration_failed', 'rolled_back'],
                  'Pending': ['pending', 'dry_run_queued', 'dry_run_in_progress', 'dry_run_complete'],
                };
                const statuses = statusMap[item.name];
                const percentage = analytics.total_repositories > 0 
                  ? ((item.value / analytics.total_repositories) * 100).toFixed(0)
                  : '0';
                
                return (
                  <button
                    key={item.name}
                    onClick={() => statuses && navigate(getRepositoriesUrl({ status: statuses }))}
                    className="flex items-center gap-2 p-2 rounded hover:bg-gh-info-bg transition-colors cursor-pointer text-left"
                  >
                    <div 
                      className="w-4 h-4 rounded flex-shrink-0" 
                      style={{ backgroundColor: item.fill }}
                    />
                    <span className="text-sm style={{ color: 'var(--fgColor-default)' }} truncate">
                      {item.name}: {item.value} ({percentage}%)
                    </span>
                  </button>
                );
              })}
            </div>
          </div>
        </div>

        {/* Migration Progress by Organization */}
        {analytics.migration_completion_stats && analytics.migration_completion_stats.length > 0 && (() => {
          // Determine terminology based on source context
          const progressGroupLabel = isAllSourcesMode && hasMultipleSources 
            ? 'Source Group' 
            : sourceType === 'azuredevops' 
              ? 'Project' 
              : 'Organization';
          
          const progressSectionTitle = isAllSourcesMode && hasMultipleSources 
            ? 'Migration Progress by Source Group' 
            : sourceType === 'azuredevops' 
              ? 'Migration Progress by Project' 
              : 'Migration Progress by Organization';
          
          const progressDescription = isAllSourcesMode && hasMultipleSources
            ? 'Detailed migration status breakdown by source group, showing completion rates and identifying areas requiring attention.'
            : sourceType === 'azuredevops'
              ? 'Detailed migration status breakdown by Azure DevOps project, showing completion rates and identifying areas requiring attention.'
              : 'Detailed migration status breakdown by GitHub organization, showing completion rates and identifying areas requiring attention.';
          
          return (
          <div className="rounded-lg shadow-sm p-6 mb-6" style={{ backgroundColor: 'var(--bgColor-default)' }}>
            <h3 className="text-lg font-medium mb-4" style={{ color: 'var(--fgColor-default)' }}>
              {progressSectionTitle}
            </h3>
            <p className="text-sm mb-4" style={{ color: 'var(--fgColor-muted)' }}>
              {progressDescription}
            </p>
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y" style={{ borderColor: 'var(--borderColor-muted)' }}>
                <thead style={{ backgroundColor: 'var(--bgColor-muted)' }}>
                  <tr>
                    <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider" style={{ color: 'var(--fgColor-muted)' }}>
                      {progressGroupLabel}
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider" style={{ color: 'var(--fgColor-muted)' }}>
                      Total
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider" style={{ color: 'var(--fgColor-muted)' }}>
                      Completed
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider" style={{ color: 'var(--fgColor-muted)' }}>
                      In Progress
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider" style={{ color: 'var(--fgColor-muted)' }}>
                      Pending
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider" style={{ color: 'var(--fgColor-muted)' }}>
                      Failed
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider" style={{ color: 'var(--fgColor-muted)' }}>
                      Progress
                    </th>
                  </tr>
                </thead>
                <tbody className="divide-y" style={{ backgroundColor: 'var(--bgColor-default)', borderColor: 'var(--borderColor-muted)' }}>
                  {analytics.migration_completion_stats.map((org) => {
                    const completionPercentage = org.total_repos > 0 
                      ? Math.round((org.completed_count / org.total_repos) * 100)
                      : 0;
                    
                    return (
                      <tr key={org.organization} className="hover:opacity-80 transition-opacity">
                        <td className="px-6 py-4 whitespace-nowrap text-sm font-medium" style={{ color: 'var(--fgColor-default)' }}>
                          {org.organization}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm" style={{ color: 'var(--fgColor-muted)' }}>
                          {org.total_repos}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm font-medium" style={{ color: 'var(--fgColor-success)' }}>
                          {org.completed_count}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm" style={{ color: 'var(--fgColor-accent)' }}>
                          {org.in_progress_count}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm" style={{ color: 'var(--fgColor-muted)' }}>
                          {org.pending_count}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm" style={{ color: 'var(--fgColor-danger)' }}>
                          {org.failed_count}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap">
                          <div className="flex items-center gap-2">
                            <div className="flex-1 min-w-[100px]">
                              <ProgressBar 
                                progress={completionPercentage} 
                                aria-label={`${org.organization} ${completionPercentage}% complete`}
                                bg="success.emphasis"
                                barSize="small"
                              />
                            </div>
                            <span className="text-sm font-medium min-w-[3rem] text-right" style={{ color: 'var(--fgColor-default)' }}>
                              {completionPercentage}%
                            </span>
                          </div>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          </div>
          );
        })()}

        {/* Performance Metrics Card */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
          <div className="rounded-lg shadow-sm p-6" style={{ backgroundColor: 'var(--bgColor-default)' }}>
            <h3 className="text-lg font-medium mb-4" style={{ color: 'var(--fgColor-default)' }}>Performance Metrics</h3>
            <p className="text-sm mb-4" style={{ color: 'var(--fgColor-muted)' }}>Key migration performance indicators including average time per repository and daily throughput over the last 30 days.</p>
            <div className="space-y-4">
              <div className="flex justify-between items-center">
                <span className="text-sm" style={{ color: 'var(--fgColor-default)' }}>Average Migration Time</span>
                <span className="text-lg font-medium" style={{ color: 'var(--fgColor-default)' }}>
                  {analytics.average_migration_time && analytics.average_migration_time > 0 
                    ? formatDuration(analytics.average_migration_time) 
                    : 'N/A'}
                </span>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-sm" style={{ color: 'var(--fgColor-default)' }}>Median Migration Time</span>
                <span className="text-lg font-medium" style={{ color: 'var(--fgColor-default)' }}>
                  {analytics.median_migration_time && analytics.median_migration_time > 0 
                    ? formatDuration(analytics.median_migration_time) 
                    : 'N/A'}
                </span>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-sm" style={{ color: 'var(--fgColor-default)' }}>Total Migrated</span>
                <span className="text-lg font-medium" style={{ color: 'var(--fgColor-success)' }}>
                  {analytics.migrated_count}
                </span>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-sm" style={{ color: 'var(--fgColor-default)' }}>Failed Migrations</span>
                <span className="text-lg font-medium" style={{ color: 'var(--fgColor-danger)' }}>
                  {analytics.failed_count}
                </span>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-sm" style={{ color: 'var(--fgColor-default)' }}>Daily Average (30 days)</span>
                <span className="text-lg font-medium" style={{ color: 'var(--fgColor-accent)' }}>
                  {analytics.migration_velocity ? analytics.migration_velocity.repos_per_day.toFixed(1) : '0'}
                </span>
              </div>
            </div>
          </div>

          {/* Status Breakdown Bar Chart */}
          <div className="rounded-lg shadow-sm p-6" style={{ backgroundColor: 'var(--bgColor-default)' }}>
            <h2 className="text-lg font-medium mb-4" style={{ color: 'var(--fgColor-default)' }}>Status Breakdown</h2>
            <p className="text-sm mb-4" style={{ color: 'var(--fgColor-muted)' }}>Visual comparison of repository counts across all migration statuses, including dry-run and excluded repositories.</p>
            <ResponsiveContainer width="100%" height={250}>
              <BarChart data={statusChartData}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="name" angle={-45} textAnchor="end" height={100} tick={{ fontSize: 11 }} />
                <YAxis />
                <Tooltip 
                  contentStyle={{
                    backgroundColor: 'rgba(27, 31, 36, 0.95)',
                    border: '1px solid rgba(255, 255, 255, 0.1)',
                    borderRadius: '6px',
                    color: '#ffffff',
                    padding: '8px 12px'
                  }}
                  labelStyle={{ color: '#ffffff', fontWeight: 600 }}
                  itemStyle={{ color: '#ffffff' }}
                  cursor={{ fill: 'rgba(127, 127, 127, 0.1)' }}
                />
                <Bar dataKey="value" fill="#3B82F6" radius={[4, 4, 0, 0]}>
                  {statusChartData.map((entry, index) => (
                    <Cell key={`cell-${index}`} fill={entry.fill} />
                  ))}
                </Bar>
              </BarChart>
            </ResponsiveContainer>
          </div>
        </div>

        {/* Detailed Status Table */}
        <div className="rounded-lg shadow-sm p-6" style={{ backgroundColor: 'var(--bgColor-default)' }}>
          <h2 className="text-lg font-medium mb-4" style={{ color: 'var(--fgColor-default)' }}>Detailed Status Breakdown</h2>
          <p className="text-sm mb-4" style={{ color: 'var(--fgColor-muted)' }}>Comprehensive status listing with exact counts and percentages for reporting and tracking purposes.</p>
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y" style={{ borderColor: 'var(--borderColor-muted)' }}>
              <thead style={{ backgroundColor: 'var(--bgColor-muted)' }}>
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider" style={{ color: 'var(--fgColor-muted)' }}>
                    Status
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider" style={{ color: 'var(--fgColor-muted)' }}>
                    Count
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider" style={{ color: 'var(--fgColor-muted)' }}>
                    Percentage
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y" style={{ backgroundColor: 'var(--bgColor-default)', borderColor: 'var(--borderColor-muted)' }}>
                {(() => {
                  // Calculate total from all status counts (including wont_migrate)
                  const totalAllStatuses = Object.values(analytics.status_breakdown).reduce((sum, count) => sum + count, 0);
                  
                  return Object.entries(analytics.status_breakdown)
                    .sort((a, b) => b[1] - a[1])
                    .map(([status, count]) => {
                      const percentage = totalAllStatuses > 0
                        ? ((count / totalAllStatuses) * 100).toFixed(1)
                        : '0.0';
                      return (
                        <tr key={status}>
                          <td className="px-6 py-4 whitespace-nowrap">
                            <div className="flex items-center">
                              <div
                                className="w-3 h-3 rounded-full mr-3"
                                style={{ backgroundColor: STATUS_COLORS[status] || '#9CA3AF' }}
                              ></div>
                              <span className="text-sm" style={{ color: 'var(--fgColor-default)' }}>{status.replace(/_/g, ' ')}</span>
                            </div>
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-sm" style={{ color: 'var(--fgColor-default)' }}>
                            {count}
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-sm" style={{ color: 'var(--fgColor-default)' }}>
                            {percentage}%
                          </td>
                        </tr>
                      );
                    });
                })()}
              </tbody>
            </table>
          </div>
        </div>
      </section>
      )}
    </div>
  );
}

function FeatureStat({ label, count, total, onClick }: { label: string; count: number; total: number; onClick?: () => void }) {
  const percentage = total > 0 ? ((count / total) * 100).toFixed(1) : '0.0';
  
  const content = (
    <>
      <span className="text-sm style={{ color: 'var(--fgColor-default)' }} flex-1 text-left">{label}</span>
      <div className="flex items-center gap-3 ml-auto">
        <span className="text-sm font-medium style={{ color: 'var(--fgColor-default)' }} min-w-[40px] text-right">{count}</span>
        <span className="text-sm style={{ color: 'var(--fgColor-muted)' }} min-w-[55px] text-left">({percentage}%)</span>
      </div>
    </>
  );

  if (onClick) {
    return (
      <button
        onClick={onClick}
        className="flex items-center justify-between w-full px-3 py-2 -mx-3 rounded hover:bg-gh-info-bg transition-colors cursor-pointer group text-left"
      >
        {content}
        <ChevronRightIcon 
          size={16}
          className="text-gh-text-secondary opacity-0 group-hover:opacity-100 transition-opacity ml-2 flex-shrink-0" 
        />
      </button>
    );
  }

  return (
    <div className="flex items-center justify-between py-2 px-3 -mx-3 text-left">
      {content}
    </div>
  );
}

