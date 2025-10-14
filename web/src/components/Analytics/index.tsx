import { BarChart, Bar, PieChart, Pie, Cell, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';
import { LoadingSpinner } from '../common/LoadingSpinner';
import { RefreshIndicator } from '../common/RefreshIndicator';
import { formatDuration } from '../../utils/format';
import { useAnalytics } from '../../hooks/useQueries';

const STATUS_COLORS: Record<string, string> = {
  pending: '#9CA3AF',
  in_progress: '#3B82F6',
  migration_complete: '#10B981',
  complete: '#059669',
  failed: '#EF4444',
  dry_run_complete: '#8B5CF6',
};

export function Analytics() {
  const { data: analytics, isLoading, isFetching } = useAnalytics();

  if (isLoading) return <LoadingSpinner />;
  if (!analytics) return <div className="text-center py-12 text-gray-500">No analytics data available</div>;

  // Prepare chart data
  const statusChartData = Object.entries(analytics.status_breakdown).map(([status, count]) => ({
    name: status.replace(/_/g, ' '),
    value: count,
    fill: STATUS_COLORS[status] || '#9CA3AF',
  }));

  const completionRate = analytics.total_repositories > 0
    ? Math.round((analytics.migrated_count / analytics.total_repositories) * 100)
    : 0;

  const progressData = [
    { name: 'Migrated', value: analytics.migrated_count, fill: '#10B981' },
    { name: 'In Progress', value: analytics.in_progress_count, fill: '#3B82F6' },
    { name: 'Failed', value: analytics.failed_count, fill: '#EF4444' },
    { name: 'Pending', value: analytics.pending_count, fill: '#9CA3AF' },
  ].filter(item => item.value > 0);

  const sizeCategories: Record<string, string> = {
    small: 'Small (<100MB)',
    medium: 'Medium (100MB-1GB)',
    large: 'Large (1GB-5GB)',
    very_large: 'Very Large (>5GB)',
    unknown: 'Unknown',
  };

  const sizeColors: Record<string, string> = {
    small: '#10B981',
    medium: '#3B82F6',
    large: '#F59E0B',
    very_large: '#EF4444',
    unknown: '#9CA3AF',
  };

  return (
    <div className="max-w-7xl mx-auto relative">
      <RefreshIndicator isRefreshing={isFetching && !isLoading} />
      <h1 className="text-3xl font-light text-gray-900 mb-8">Migration Analytics</h1>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
        <StatCard
          title="Total Repositories"
          value={analytics.total_repositories}
          color="blue"
        />
        <StatCard
          title="Migrated"
          value={analytics.migrated_count}
          color="green"
          subtitle={`${completionRate}% complete`}
        />
        <StatCard
          title="In Progress"
          value={analytics.in_progress_count}
          color="blue"
        />
        <StatCard
          title="Failed"
          value={analytics.failed_count}
          color="red"
        />
      </div>

      {/* Average Migration Time */}
      {analytics.average_migration_time && analytics.average_migration_time > 0 && (
        <div className="bg-white rounded-lg shadow-sm p-6 mb-8">
          <h2 className="text-lg font-medium text-gray-900 mb-2">Average Migration Time</h2>
          <div className="text-3xl font-light text-blue-600">
            {formatDuration(analytics.average_migration_time)}
          </div>
        </div>
      )}

      {/* Charts */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Status Breakdown Bar Chart */}
        <div className="bg-white rounded-lg shadow-sm p-6">
          <h2 className="text-lg font-medium text-gray-900 mb-4">Status Breakdown</h2>
          <ResponsiveContainer width="100%" height={300}>
            <BarChart data={statusChartData}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="name" angle={-45} textAnchor="end" height={100} />
              <YAxis />
              <Tooltip />
              <Bar dataKey="value" fill="#3B82F6">
                {statusChartData.map((entry, index) => (
                  <Cell key={`cell-${index}`} fill={entry.fill} />
                ))}
              </Bar>
            </BarChart>
          </ResponsiveContainer>
        </div>

        {/* Progress Pie Chart */}
        <div className="bg-white rounded-lg shadow-sm p-6">
          <h2 className="text-lg font-medium text-gray-900 mb-4">Migration Progress</h2>
          <ResponsiveContainer width="100%" height={300}>
            <PieChart>
              <Pie
                data={progressData}
                cx="50%"
                cy="50%"
                labelLine={false}
                label={({ name, value, percent }) => `${name}: ${value} (${(percent * 100).toFixed(0)}%)`}
                outerRadius={80}
                dataKey="value"
              >
                {progressData.map((entry, index) => (
                  <Cell key={`cell-${index}`} fill={entry.fill} />
                ))}
              </Pie>
              <Tooltip />
            </PieChart>
          </ResponsiveContainer>
        </div>
      </div>

      {/* Detailed Status Table */}
      <div className="bg-white rounded-lg shadow-sm p-6 mt-6">
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
              {Object.entries(analytics.status_breakdown)
                .sort((a, b) => b[1] - a[1])
                .map(([status, count]) => {
                  const percentage = analytics.total_repositories > 0
                    ? ((count / analytics.total_repositories) * 100).toFixed(1)
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
                })}
            </tbody>
          </table>
        </div>
      </div>

      {/* Discovery Statistics Section */}
      {analytics.organization_stats && analytics.organization_stats.length > 0 && (
        <div className="mt-8">
          <h2 className="text-2xl font-light text-gray-900 mb-6">Discovery Statistics</h2>
          
          {/* Organization Breakdown */}
          <div className="bg-white rounded-lg shadow-sm p-6 mb-6">
            <h3 className="text-lg font-medium text-gray-900 mb-4">Organization Breakdown</h3>
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-gray-200">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Organization
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Total Repositories
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

          {/* Size Distribution and Feature Stats */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
            {/* Size Distribution */}
            {analytics.size_distribution && analytics.size_distribution.length > 0 && (
              <div className="bg-white rounded-lg shadow-sm p-6">
                <h3 className="text-lg font-medium text-gray-900 mb-4">Repository Size Distribution</h3>
                <ResponsiveContainer width="100%" height={300}>
                  <PieChart>
                    <Pie
                      data={analytics.size_distribution.map(item => ({
                        name: sizeCategories[item.category] || item.category,
                        value: item.count,
                        fill: sizeColors[item.category] || '#9CA3AF',
                      }))}
                      cx="50%"
                      cy="50%"
                      labelLine={false}
                      label={({ name, value, percent }) => `${name}: ${value} (${(percent * 100).toFixed(0)}%)`}
                      outerRadius={80}
                      dataKey="value"
                    >
                      {analytics.size_distribution.map((entry, index) => (
                        <Cell key={`cell-${index}`} fill={sizeColors[entry.category] || '#9CA3AF'} />
                      ))}
                    </Pie>
                    <Tooltip />
                  </PieChart>
                </ResponsiveContainer>
              </div>
            )}

            {/* Feature Stats */}
            {analytics.feature_stats && (
              <div className="bg-white rounded-lg shadow-sm p-6">
                <h3 className="text-lg font-medium text-gray-900 mb-4">Feature Usage Statistics</h3>
                <div className="space-y-3">
                  <FeatureStat label="Archived" count={analytics.feature_stats.is_archived} total={analytics.feature_stats.total_repositories} />
                  <FeatureStat label="LFS" count={analytics.feature_stats.has_lfs} total={analytics.feature_stats.total_repositories} />
                  <FeatureStat label="Submodules" count={analytics.feature_stats.has_submodules} total={analytics.feature_stats.total_repositories} />
                  <FeatureStat label="Large Files (>100MB)" count={analytics.feature_stats.has_large_files} total={analytics.feature_stats.total_repositories} />
                  <FeatureStat label="GitHub Actions" count={analytics.feature_stats.has_actions} total={analytics.feature_stats.total_repositories} />
                  <FeatureStat label="Wikis" count={analytics.feature_stats.has_wiki} total={analytics.feature_stats.total_repositories} />
                  <FeatureStat label="Pages" count={analytics.feature_stats.has_pages} total={analytics.feature_stats.total_repositories} />
                  <FeatureStat label="Discussions" count={analytics.feature_stats.has_discussions} total={analytics.feature_stats.total_repositories} />
                  <FeatureStat label="Projects" count={analytics.feature_stats.has_projects} total={analytics.feature_stats.total_repositories} />
                  <FeatureStat label="Branch Protections" count={analytics.feature_stats.has_branch_protections} total={analytics.feature_stats.total_repositories} />
                </div>
              </div>
            )}
          </div>

          {/* Migration Completion by Organization */}
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
                        Total Repos
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
                                  className="bg-green-600 h-2 rounded-full"
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
        </div>
      )}
    </div>
  );
}

function FeatureStat({ label, count, total }: { label: string; count: number; total: number }) {
  const percentage = total > 0 ? ((count / total) * 100).toFixed(1) : '0.0';
  
  return (
    <div className="flex items-center justify-between">
      <span className="text-sm text-gray-700">{label}</span>
      <div className="flex items-center gap-2">
        <span className="text-sm font-medium text-gray-900">{count}</span>
        <span className="text-xs text-gray-500">({percentage}%)</span>
      </div>
    </div>
  );
}

interface StatCardProps {
  title: string;
  value: number;
  color: 'blue' | 'green' | 'red' | 'yellow';
  subtitle?: string;
}

function StatCard({ title, value, color, subtitle }: StatCardProps) {
  const colorClasses = {
    blue: 'bg-blue-50 text-blue-600',
    green: 'bg-green-50 text-green-600',
    red: 'bg-red-50 text-red-600',
    yellow: 'bg-yellow-50 text-yellow-600',
  };

  return (
    <div className="bg-white rounded-lg shadow-sm p-6">
      <h3 className="text-sm font-medium text-gray-600 mb-2">{title}</h3>
      <div className={`text-3xl font-light mb-1 ${colorClasses[color]}`}>
        {value}
      </div>
      {subtitle && <div className="text-sm text-gray-500">{subtitle}</div>}
    </div>
  );
}

