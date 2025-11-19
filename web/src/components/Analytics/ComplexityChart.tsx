import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Cell } from 'recharts';
import { useNavigate } from 'react-router-dom';
import { ComplexityDistribution } from '../../types';
import { getRepositoriesUrl } from '../../utils/filters';
import { ComplexityInfoModal } from '../common/ComplexityInfoModal';

interface ComplexityChartProps {
  data: ComplexityDistribution[];
  source?: 'github' | 'azuredevops' | 'all';
}

const COMPLEXITY_COLORS: Record<string, string> = {
  simple: '#10B981',
  medium: '#F59E0B',
  complex: '#F97316',
  very_complex: '#EF4444',
};

const COMPLEXITY_LABELS: Record<string, string> = {
  simple: 'Simple',
  medium: 'Medium',
  complex: 'Complex',
  very_complex: 'Very Complex',
};

export function ComplexityChart({ data, source = 'all' }: ComplexityChartProps) {
  const navigate = useNavigate();

  // Ensure all complexity levels are shown, even if they have 0 count
  const allCategories = ['simple', 'medium', 'complex', 'very_complex'];
  const dataMap = new Map((data || []).map(item => [item.category, item.count]));
  
  const chartData = allCategories.map(category => ({
    category,
    count: dataMap.get(category) || 0,
    name: COMPLEXITY_LABELS[category] || category,
    fill: COMPLEXITY_COLORS[category] || '#9CA3AF',
  }));

  const handleBarClick = (entry: any) => {
    if (entry && entry.category) {
      const url = getRepositoriesUrl({ complexity: [entry.category] });
      navigate(url);
    }
  };

  // Dynamic description based on source
  const getDescription = () => {
    if (source === 'azuredevops') {
      return 'Repositories categorized by ADO → GitHub migration complexity factors including TFVC detection, pipeline types, Azure Boards, wikis, and test plans.';
    } else if (source === 'github') {
      return 'Repositories categorized by GitHub migration complexity factors including size, LFS, submodules, environments, secrets, and more.';
    } else {
      return 'Repositories categorized by migration complexity. Scoring varies by source: GitHub factors include size, LFS, and environments; Azure DevOps includes TFVC, pipelines, and boards.';
    }
  };

  return (
    <div className="bg-white rounded-lg shadow-sm p-6">
      <div className="flex justify-between items-start mb-4">
        <div>
          <h2 className="text-lg font-medium text-gray-900 mb-1">Repository Complexity Distribution</h2>
          <p className="text-sm text-gray-600">
            {getDescription()}
          </p>
        </div>
        <ComplexityInfoModal source={source} />
      </div>
      <ResponsiveContainer width="100%" height={300}>
        <BarChart data={chartData}>
          <CartesianGrid strokeDasharray="3 3" />
          <XAxis dataKey="name" />
          <YAxis />
          <Tooltip 
            formatter={(value: number) => [`${value} repos`, 'Count']}
            cursor={{ fill: 'rgba(59, 130, 246, 0.1)' }}
          />
          <Bar 
            dataKey="count" 
            radius={[8, 8, 0, 0]}
            onClick={handleBarClick}
            cursor="pointer"
          >
            {chartData.map((entry, index) => (
              <Cell key={`cell-${index}`} fill={entry.fill} />
            ))}
          </Bar>
        </BarChart>
      </ResponsiveContainer>
      
      {/* Legend */}
      <div className="mt-4 flex flex-wrap gap-4 justify-center">
        {chartData.map((item) => (
          <button
            key={item.category}
            onClick={() => navigate(getRepositoriesUrl({ complexity: [item.category] }))}
            className="flex items-center gap-2 px-3 py-1.5 rounded hover:bg-gray-100 transition-colors cursor-pointer"
          >
            <div 
              className="w-4 h-4 rounded" 
              style={{ backgroundColor: item.fill }}
            />
            <span className="text-sm text-gray-700">
              {item.name}: {item.count}
            </span>
          </button>
        ))}
      </div>
    </div>
  );
}

