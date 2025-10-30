import { useState, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { BarChart, Bar, PieChart, Pie, Cell, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';
import { LoadingSpinner } from '../common/LoadingSpinner';
import { RefreshIndicator } from '../common/RefreshIndicator';
import { formatDuration } from '../../utils/format';
import { useAnalytics } from '../../hooks/useQueries';
import { FilterBar } from './FilterBar';
import { MigrationTrendChart } from './MigrationTrendChart';
import { ComplexityChart } from './ComplexityChart';
import { KPICard } from './KPICard';
import { getRepositoriesUrl } from '../../utils/filters';

const STATUS_COLORS: Record<string, string> = {
  pending: '#656D76',
  in_progress: '#0969DA',
  migration_complete: '#1A7F37',
  complete: '#1A7F37',
  failed: '#D1242F',
  dry_run_complete: '#8250DF',
  wont_migrate: '#6B7280',
};

export function Analytics() {
  const navigate = useNavigate();
  const [selectedOrganization, setSelectedOrganization] = useState('');
  const [selectedBatch, setSelectedBatch] = useState('');

  const { data: analytics, isLoading, isFetching } = useAnalytics({
    organization: selectedOrganization || undefined,
    batch_id: selectedBatch || undefined,
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

  return (
    <div className="max-w-7xl mx-auto relative">
      <RefreshIndicator isRefreshing={isFetching && !isLoading} />
      <h1 className="text-2xl font-semibold text-gh-text-primary mb-8">Analytics Dashboard</h1>

      {/* Filter Bar */}
      <FilterBar
        selectedOrganization={selectedOrganization}
        selectedBatch={selectedBatch}
        onOrganizationChange={setSelectedOrganization}
        onBatchChange={setSelectedBatch}
      />

      {/* SECTION 1: DISCOVERY ANALYTICS */}
      <section className="mb-12">
        <div className="border-l-4 border-gh-blue pl-4 mb-6">
          <h2 className="text-xl font-semibold text-gh-text-primary">Discovery Analytics</h2>
          <p className="text-sm text-gh-text-secondary mt-1">
            Source environment overview to drive batch planning decisions
          </p>
        </div>

        {/* Discovery Summary Cards */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
          <StatCard
            title="Total Repositories"
            value={analytics.total_repositories}
            color="blue"
          />
          <StatCard
            title="Organizations"
            value={analytics.organization_stats?.length || 0}
            color="blue"
          />
          <StatCard
            title="High Complexity"
            value={highComplexityCount}
            color="yellow"
            subtitle={`${analytics.total_repositories > 0 ? Math.round((highComplexityCount / analytics.total_repositories) * 100) : 0}% of total`}
            onClick={() => navigate(getRepositoriesUrl({ complexity: ['complex', 'very_complex'] }))}
          />
          <StatCard
            title="Features Detected"
            value={analytics.feature_stats ? Object.values(analytics.feature_stats).filter(v => typeof v === 'number' && v > 0).length : 0}
            color="blue"
          />
        </div>

        {/* Complexity and Size Distribution Charts */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
          <ComplexityChart data={analytics.complexity_distribution || []} />
          
          {/* Size Distribution */}
          {analytics.size_distribution && analytics.size_distribution.length > 0 && (
            <div className="bg-white rounded-lg shadow-sm p-6">
              <h3 className="text-lg font-medium text-gray-900 mb-4">Repository Size Distribution</h3>
              <p className="text-sm text-gray-600 mb-4">Click on the chart to view repositories by size category</p>
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
                    onClick={(data: any) => {
                      if (data && data.category) {
                        navigate(getRepositoriesUrl({ size_category: [data.category] }));
                      }
                    }}
                    cursor="pointer"
                  >
                    {analytics.size_distribution.map((entry, index) => (
                      <Cell key={`cell-${index}`} fill={sizeColors[entry.category] || '#9CA3AF'} />
                    ))}
                  </Pie>
                  <Tooltip 
                    formatter={(value: number, _name: string, props: any) => [
                      `${value} repositories`,
                      props.payload.name
                    ]}
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
                    <span className="text-sm text-gray-700 truncate">
                      {sizeCategories[item.category] || item.category}: {item.count}
                    </span>
                  </button>
                ))}
              </div>
            </div>
          )}
        </div>

        {/* Organization Breakdown and Feature Stats */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
          {/* Organization Breakdown */}
          {analytics.organization_stats && analytics.organization_stats.length > 0 && (
            <div className="bg-white rounded-lg shadow-sm p-6">
              <h3 className="text-lg font-medium text-gray-900 mb-4">Organization Breakdown</h3>
              <div className="overflow-x-auto">
                <table className="min-w-full divide-y divide-gray-200">
                  <thead className="bg-gray-50">
                    <tr>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                        Organization
                      </th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                        Repositories
                      </th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                        Percentage
                      </th>
                    </tr>
                  </thead>
                  <tbody className="bg-white divide-y divide-gray-200">
                    {analytics.organization_stats
                      .sort((a, b) => b.total_repos - a.total_repos)
                      .map((org) => {
                        const percentage = analytics.total_repositories > 0
                          ? ((org.total_repos / analytics.total_repositories) * 100).toFixed(1)
                          : '0.0';
                        return (
                          <tr key={org.organization} className="hover:bg-gray-50">
                            <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                              {org.organization}
                            </td>
                            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                              {org.total_repos}
                            </td>
                            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                              {percentage}%
                            </td>
                          </tr>
                        );
                      })}
                  </tbody>
                </table>
              </div>
            </div>
          )}

          {/* Feature Stats */}
          {analytics.feature_stats && (
            <div className="bg-white rounded-lg shadow-sm p-6">
              <h3 className="text-lg font-medium text-gray-900 mb-4">Feature Usage Statistics</h3>
              <p className="text-sm text-gray-600 mb-4">Click on any feature to view repositories with that feature</p>
              <div className="space-y-1">
                <FeatureStat 
                  label="Archived" 
                  count={analytics.feature_stats.is_archived} 
                  total={analytics.feature_stats.total_repositories}
                  onClick={() => navigate(getRepositoriesUrl({ is_archived: true }))}
                />
                <FeatureStat 
                  label="Forked Repositories" 
                  count={analytics.feature_stats.is_fork} 
                  total={analytics.feature_stats.total_repositories}
                  onClick={() => navigate(getRepositoriesUrl({ is_fork: true }))}
                />
                <FeatureStat 
                  label="LFS" 
                  count={analytics.feature_stats.has_lfs} 
                  total={analytics.feature_stats.total_repositories}
                  onClick={() => navigate(getRepositoriesUrl({ has_lfs: true }))}
                />
                <FeatureStat 
                  label="Submodules" 
                  count={analytics.feature_stats.has_submodules} 
                  total={analytics.feature_stats.total_repositories}
                  onClick={() => navigate(getRepositoriesUrl({ has_submodules: true }))}
                />
                <FeatureStat 
                  label="Large Files (>100MB)" 
                  count={analytics.feature_stats.has_large_files} 
                  total={analytics.feature_stats.total_repositories}
                  onClick={() => navigate(getRepositoriesUrl({ has_large_files: true }))}
                />
                <FeatureStat 
                  label="GitHub Actions" 
                  count={analytics.feature_stats.has_actions} 
                  total={analytics.feature_stats.total_repositories}
                  onClick={() => navigate(getRepositoriesUrl({ has_actions: true }))}
                />
                <FeatureStat 
                  label="Wikis" 
                  count={analytics.feature_stats.has_wiki} 
                  total={analytics.feature_stats.total_repositories}
                  onClick={() => navigate(getRepositoriesUrl({ has_wiki: true }))}
                />
                <FeatureStat 
                  label="Pages" 
                  count={analytics.feature_stats.has_pages} 
                  total={analytics.feature_stats.total_repositories}
                  onClick={() => navigate(getRepositoriesUrl({ has_pages: true }))}
                />
                <FeatureStat 
                  label="Discussions" 
                  count={analytics.feature_stats.has_discussions} 
                  total={analytics.feature_stats.total_repositories}
                  onClick={() => navigate(getRepositoriesUrl({ has_discussions: true }))}
                />
                <FeatureStat 
                  label="Projects" 
                  count={analytics.feature_stats.has_projects} 
                  total={analytics.feature_stats.total_repositories}
                  onClick={() => navigate(getRepositoriesUrl({ has_projects: true }))}
                />
                <FeatureStat 
                  label="Packages" 
                  count={analytics.feature_stats.has_packages} 
                  total={analytics.feature_stats.total_repositories}
                  onClick={() => navigate(getRepositoriesUrl({ has_packages: true }))}
                />
                <FeatureStat 
                  label="Branch Protections" 
                  count={analytics.feature_stats.has_branch_protections} 
                  total={analytics.feature_stats.total_repositories}
                  onClick={() => navigate(getRepositoriesUrl({ has_branch_protections: true }))}
                />
                <FeatureStat 
                  label="Rulesets" 
                  count={analytics.feature_stats.has_rulesets} 
                  total={analytics.feature_stats.total_repositories}
                  onClick={() => navigate(getRepositoriesUrl({ has_rulesets: true }))}
                />
                <FeatureStat 
                  label="Code Scanning" 
                  count={analytics.feature_stats.has_code_scanning} 
                  total={analytics.feature_stats.total_repositories}
                  onClick={() => navigate(getRepositoriesUrl({ has_code_scanning: true }))}
                />
                <FeatureStat 
                  label="Dependabot" 
                  count={analytics.feature_stats.has_dependabot} 
                  total={analytics.feature_stats.total_repositories}
                  onClick={() => navigate(getRepositoriesUrl({ has_dependabot: true }))}
                />
                <FeatureStat 
                  label="Secret Scanning" 
                  count={analytics.feature_stats.has_secret_scanning} 
                  total={analytics.feature_stats.total_repositories}
                  onClick={() => navigate(getRepositoriesUrl({ has_secret_scanning: true }))}
                />
                <FeatureStat 
                  label="CODEOWNERS" 
                  count={analytics.feature_stats.has_codeowners} 
                  total={analytics.feature_stats.total_repositories}
                  onClick={() => navigate(getRepositoriesUrl({ has_codeowners: true }))}
                />
                <FeatureStat 
                  label="Self-Hosted Runners" 
                  count={analytics.feature_stats.has_self_hosted_runners} 
                  total={analytics.feature_stats.total_repositories}
                  onClick={() => navigate(getRepositoriesUrl({ has_self_hosted_runners: true }))}
                />
                <FeatureStat 
                  label="Release Assets" 
                  count={analytics.feature_stats.has_release_assets} 
                  total={analytics.feature_stats.total_repositories}
                  onClick={() => navigate(getRepositoriesUrl({ has_release_assets: true }))}
                />
              </div>
            </div>
          )}
        </div>
      </section>

      {/* SECTION 2: MIGRATION ANALYTICS */}
      <section>
        <div className="border-l-4 border-green-500 pl-4 mb-6">
          <h2 className="text-2xl font-light text-gray-900">Migration Analytics</h2>
          <p className="text-sm text-gray-600 mt-1">
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
          <div className="bg-white rounded-lg shadow-sm p-6">
            <h2 className="text-lg font-medium text-gray-900 mb-4">Current Status Distribution</h2>
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
                  onClick={(data: any) => {
                    if (data && data.name) {
                      const statusMap: Record<string, string> = {
                        'Migrated': 'migration_complete',
                        'In Progress': 'in_progress',
                        'Failed': 'failed',
                        'Pending': 'pending',
                      };
                      const status = statusMap[data.name];
                      if (status) {
                        navigate(getRepositoriesUrl({ status: status }));
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
                  formatter={(value: number, _name: string, props: any) => [
                    `${value} repositories`,
                    props.payload.name
                  ]}
                />
              </PieChart>
            </ResponsiveContainer>
            
            {/* Legend */}
            <div className="mt-4 grid grid-cols-2 gap-2">
              {progressData.map((item) => {
                const statusMap: Record<string, string> = {
                  'Migrated': 'migration_complete',
                  'In Progress': 'in_progress',
                  'Failed': 'failed',
                  'Pending': 'pending',
                };
                const status = statusMap[item.name];
                const percentage = analytics.total_repositories > 0 
                  ? ((item.value / analytics.total_repositories) * 100).toFixed(0)
                  : '0';
                
                return (
                  <button
                    key={item.name}
                    onClick={() => status && navigate(getRepositoriesUrl({ status: status }))}
                    className="flex items-center gap-2 p-2 rounded hover:bg-gh-info-bg transition-colors cursor-pointer text-left"
                  >
                    <div 
                      className="w-4 h-4 rounded flex-shrink-0" 
                      style={{ backgroundColor: item.fill }}
                    />
                    <span className="text-sm text-gray-700 truncate">
                      {item.name}: {item.value} ({percentage}%)
                    </span>
                  </button>
                );
              })}
            </div>
          </div>
        </div>

        {/* Migration Progress by Organization */}
        {analytics.migration_completion_stats && analytics.migration_completion_stats.length > 0 && (
          <div className="bg-white rounded-lg shadow-sm p-6 mb-6">
            <h3 className="text-lg font-medium text-gray-900 mb-4">Migration Progress by Organization</h3>
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-gray-200">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Organization
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Total
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Completed
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      In Progress
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Pending
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Failed
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Progress
                    </th>
                  </tr>
                </thead>
                <tbody className="bg-white divide-y divide-gray-200">
                  {analytics.migration_completion_stats.map((org) => {
                    const completionPercentage = org.total_repos > 0 
                      ? Math.round((org.completed_count / org.total_repos) * 100)
                      : 0;
                    
                    return (
                      <tr key={org.organization} className="hover:bg-gray-50">
                        <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                          {org.organization}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                          {org.total_repos}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-green-600 font-medium">
                          {org.completed_count}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-blue-600">
                          {org.in_progress_count}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600">
                          {org.pending_count}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-red-600">
                          {org.failed_count}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap">
                          <div className="flex items-center gap-2">
                            <div className="flex-1 bg-gray-200 rounded-full h-2 min-w-[100px]">
                              <div
                                className="bg-gh-success h-2 rounded-full"
                                style={{ width: `${completionPercentage}%` }}
                              />
                            </div>
                            <span className="text-sm font-medium text-gray-700 min-w-[3rem] text-right">
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
        )}

        {/* Performance Metrics Card */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
          <div className="bg-white rounded-lg shadow-sm p-6">
            <h3 className="text-lg font-medium text-gray-900 mb-4">Performance Metrics</h3>
            <div className="space-y-4">
              <div className="flex justify-between items-center">
                <span className="text-sm text-gray-700">Average Migration Time</span>
                <span className="text-lg font-medium text-gray-900">
                  {analytics.average_migration_time && analytics.average_migration_time > 0 
                    ? formatDuration(analytics.average_migration_time) 
                    : 'N/A'}
                </span>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-sm text-gray-700">Total Migrated</span>
                <span className="text-lg font-medium text-green-600">
                  {analytics.migrated_count}
                </span>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-sm text-gray-700">Failed Migrations</span>
                <span className="text-lg font-medium text-red-600">
                  {analytics.failed_count}
                </span>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-sm text-gray-700">Daily Average (30 days)</span>
                <span className="text-lg font-medium text-blue-600">
                  {analytics.migration_velocity ? analytics.migration_velocity.repos_per_day.toFixed(1) : '0'}
                </span>
              </div>
            </div>
          </div>

          {/* Status Breakdown Bar Chart */}
          <div className="bg-white rounded-lg shadow-sm p-6">
            <h2 className="text-lg font-medium text-gray-900 mb-4">Status Breakdown</h2>
            <ResponsiveContainer width="100%" height={250}>
              <BarChart data={statusChartData}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="name" angle={-45} textAnchor="end" height={100} tick={{ fontSize: 11 }} />
                <YAxis />
                <Tooltip />
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
        <div className="bg-white rounded-lg shadow-sm p-6">
          <h2 className="text-lg font-medium text-gray-900 mb-4">Detailed Status Breakdown</h2>
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Status
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Count
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Percentage
                  </th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
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
                              <span className="text-sm text-gray-900">{status.replace(/_/g, ' ')}</span>
                            </div>
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                            {count}
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
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
    </div>
  );
}

function FeatureStat({ label, count, total, onClick }: { label: string; count: number; total: number; onClick?: () => void }) {
  const percentage = total > 0 ? ((count / total) * 100).toFixed(1) : '0.0';
  
  const content = (
    <>
      <span className="text-sm text-gray-700 flex-1">{label}</span>
      <div className="flex items-center gap-2 ml-auto">
        <span className="text-sm font-medium text-gray-900 min-w-[60px] text-right">{count}</span>
        <span className="text-xs text-gray-500 min-w-[60px]">({percentage}%)</span>
      </div>
    </>
  );

  if (onClick) {
    return (
      <button
        onClick={onClick}
        className="flex items-center justify-between w-full px-3 py-2 -mx-3 rounded hover:bg-gh-info-bg transition-colors cursor-pointer group"
      >
        {content}
        <svg 
          className="w-4 h-4 text-gh-text-secondary opacity-0 group-hover:opacity-100 transition-opacity ml-2 flex-shrink-0" 
          fill="none" 
          stroke="currentColor" 
          viewBox="0 0 24 24"
        >
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
        </svg>
      </button>
    );
  }

  return (
    <div className="flex items-center justify-between py-2 px-3 -mx-3">
      {content}
    </div>
  );
}

interface StatCardProps {
  title: string;
  value: number;
  color: 'blue' | 'green' | 'red' | 'yellow';
  subtitle?: string;
  onClick?: () => void;
}

function StatCard({ title, value, color, subtitle, onClick }: StatCardProps) {
  const colorClasses = {
    blue: 'bg-blue-50 text-blue-600',
    green: 'bg-green-50 text-green-600',
    red: 'bg-red-50 text-red-600',
    yellow: 'bg-yellow-50 text-yellow-600',
  };

  const isClickable = !!onClick;
  const baseClasses = "bg-white rounded-lg shadow-sm p-6 transition-all";
  const clickableClasses = isClickable 
    ? "cursor-pointer hover:shadow-lg hover:border-2 hover:border-blue-500" 
    : "";

  const handleClick = () => {
    if (onClick) {
      onClick();
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (onClick && (e.key === 'Enter' || e.key === ' ')) {
      e.preventDefault();
      onClick();
    }
  };

  return (
    <div 
      className={`${baseClasses} ${clickableClasses}`}
      onClick={handleClick}
      onKeyDown={handleKeyDown}
      role={isClickable ? 'button' : undefined}
      tabIndex={isClickable ? 0 : undefined}
      aria-label={isClickable ? `View repositories: ${title}` : undefined}
    >
      <h3 className="text-sm font-medium text-gray-600 mb-2">{title}</h3>
      <div className={`text-3xl font-light mb-1 ${colorClasses[color]}`}>
        {value}
      </div>
      {subtitle && <div className="text-sm text-gray-500">{subtitle}</div>}
    </div>
  );
}
