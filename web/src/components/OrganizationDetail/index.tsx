import { useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { api } from '../../services/api';
import type { Repository } from '../../types';
import { LoadingSpinner } from '../common/LoadingSpinner';
import { StatusBadge } from '../common/StatusBadge';
import { Badge } from '../common/Badge';
import { formatBytes } from '../../utils/format';

export function OrganizationDetail() {
  const { orgName } = useParams<{ orgName: string }>();
  const [repositories, setRepositories] = useState<Repository[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState<string>('all');
  const [searchTerm, setSearchTerm] = useState('');

  useEffect(() => {
    loadRepositories();
    const interval = setInterval(loadRepositories, 10000);
    return () => clearInterval(interval);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [orgName, filter]);

  const loadRepositories = async () => {
    setLoading(true);
    try {
      const data = await api.listRepositories({ 
        status: filter === 'all' ? undefined : filter 
      });
      
      // Filter repositories for this organization
      const orgRepos = data.filter(repo => {
        const org = repo.full_name.split('/')[0];
        return org === orgName;
      });
      
      setRepositories(orgRepos);
    } catch (error) {
      console.error('Failed to load repositories:', error);
    } finally {
      setLoading(false);
    }
  };

  const filteredRepos = repositories.filter(repo =>
    repo.full_name.toLowerCase().includes(searchTerm.toLowerCase())
  );

  const statuses = ['all', 'pending', 'in_progress', 'migration_complete', 'complete', 'failed'];

  return (
    <div className="max-w-7xl mx-auto">
      <div className="mb-6">
        <Link to="/" className="text-blue-600 hover:underline text-sm">
          ← Back to Organizations
        </Link>
      </div>

      <div className="flex justify-between items-center mb-8">
        <h1 className="text-3xl font-light text-gray-900">{orgName}</h1>
        <div className="flex gap-4">
          <input
            type="text"
            placeholder="Search repositories..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
          <select
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            className="px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          >
            {statuses.map((status) => (
              <option key={status} value={status}>
                {status === 'all' ? 'All Status' : status.charAt(0).toUpperCase() + status.slice(1).replace(/_/g, ' ')}
              </option>
            ))}
          </select>
        </div>
      </div>

      <div className="mb-4 text-sm text-gray-600">
        Showing {filteredRepos.length} of {repositories.length} repositories
      </div>

      {loading ? (
        <LoadingSpinner />
      ) : filteredRepos.length === 0 ? (
        <div className="text-center py-12 text-gray-500">
          No repositories found
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {filteredRepos.map((repo) => (
            <RepositoryCard key={repo.id} repository={repo} />
          ))}
        </div>
      )}
    </div>
  );
}

function RepositoryCard({ repository }: { repository: Repository }) {
  return (
    <Link
      to={`/repository/${encodeURIComponent(repository.full_name)}`}
      className="bg-white rounded-lg shadow-sm hover:shadow-md transition-shadow p-6 block"
    >
      <h3 className="text-lg font-medium text-gray-900 mb-2 truncate">
        {repository.full_name}
      </h3>
      <div className="mb-4">
        <StatusBadge status={repository.status} />
      </div>
      <div className="space-y-2 text-sm text-gray-600">
        <div>Size: {formatBytes(repository.total_size)}</div>
        <div>Branches: {repository.branch_count}</div>
        <div className="flex gap-2 flex-wrap mt-2">
          {repository.has_lfs && <Badge color="blue">LFS</Badge>}
          {repository.has_submodules && <Badge color="purple">Submodules</Badge>}
          {repository.has_actions && <Badge color="green">Actions</Badge>}
          {repository.has_wiki && <Badge color="yellow">Wiki</Badge>}
        </div>
      </div>
    </Link>
  );
}

