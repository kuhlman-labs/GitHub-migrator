import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { api } from '../../services/api';
import type { Repository } from '../../types';
import { LoadingSpinner } from '../common/LoadingSpinner';
import { StatusBadge } from '../common/StatusBadge';
import { Badge } from '../common/Badge';
import { formatBytes } from '../../utils/format';

export function Dashboard() {
  const [repositories, setRepositories] = useState<Repository[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState<string>('all');
  const [searchTerm, setSearchTerm] = useState('');

  useEffect(() => {
    loadRepositories();
    // Poll for updates every 10 seconds
    const interval = setInterval(loadRepositories, 10000);
    return () => clearInterval(interval);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [filter]);

  const loadRepositories = async () => {
    setLoading(true);
    try {
      const data = await api.listRepositories({ 
        status: filter === 'all' ? undefined : filter 
      });
      setRepositories(data);
    } catch (error) {
      console.error('Failed to load repositories:', error);
    } finally {
      setLoading(false);
    }
  };

  const filteredRepos = repositories.filter(repo =>
    repo.full_name.toLowerCase().includes(searchTerm.toLowerCase())
  );

  return (
    <div>
      <div className="flex justify-between items-center mb-8">
        <h1 className="text-3xl font-light text-gray-900">Repository Dashboard</h1>
        <div className="flex gap-4">
          <input
            type="text"
            placeholder="Search repositories..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
          <StatusFilter value={filter} onChange={setFilter} />
        </div>
      </div>

      <div className="mb-4 text-sm text-gray-600">
        Showing {filteredRepos.length} of {repositories.length} repositories
      </div>

      {loading ? (
        <LoadingSpinner />
      ) : (
        <RepositoryGrid repositories={filteredRepos} />
      )}
    </div>
  );
}

function StatusFilter({ value, onChange }: { value: string; onChange: (value: string) => void }) {
  const statuses = ['all', 'pending', 'in_progress', 'migration_complete', 'complete', 'failed'];
  
  return (
    <select
      value={value}
      onChange={(e) => onChange(e.target.value)}
      className="px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
    >
      {statuses.map((status) => (
        <option key={status} value={status}>
          {status === 'all' ? 'All Status' : status.charAt(0).toUpperCase() + status.slice(1).replace(/_/g, ' ')}
        </option>
      ))}
    </select>
  );
}

function RepositoryGrid({ repositories }: { repositories: Repository[] }) {
  if (repositories.length === 0) {
    return (
      <div className="text-center py-12 text-gray-500">
        No repositories found
      </div>
    );
  }

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
      {repositories.map((repo) => (
        <RepositoryCard key={repo.id} repository={repo} />
      ))}
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

