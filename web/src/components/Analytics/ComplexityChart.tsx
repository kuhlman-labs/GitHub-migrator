import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Cell } from 'recharts';
import { ComplexityDistribution } from '../../types';

interface ComplexityChartProps {
  data: ComplexityDistribution[];
}

const COMPLEXITY_COLORS: Record<string, string> = {
  low: '#10B981',
  medium: '#F59E0B',
  high: '#F97316',
  very_high: '#EF4444',
};

const COMPLEXITY_LABELS: Record<string, string> = {
  low: 'Low',
  medium: 'Medium',
  high: 'High',
  very_high: 'Very High',
};

export function ComplexityChart({ data }: ComplexityChartProps) {
  if (!data || data.length === 0) {
    return (
      <div className="bg-white rounded-lg shadow-sm p-6">
        <h2 className="text-lg font-medium text-gray-900 mb-4">Repository Complexity Distribution</h2>
        <div className="h-[300px] flex items-center justify-center text-gray-500">
          No complexity data available
        </div>
      </div>
    );
  }

  const chartData = data.map(item => ({
    ...item,
    name: COMPLEXITY_LABELS[item.category] || item.category,
    fill: COMPLEXITY_COLORS[item.category] || '#9CA3AF',
  }));

  return (
    <div className="bg-white rounded-lg shadow-sm p-6">
      <h2 className="text-lg font-medium text-gray-900 mb-4">Repository Complexity Distribution</h2>
      <p className="text-sm text-gray-600 mb-4">
        Based on size, LFS, submodules, large files, and branch protections
      </p>
      <ResponsiveContainer width="100%" height={300}>
        <BarChart data={chartData}>
          <CartesianGrid strokeDasharray="3 3" />
          <XAxis dataKey="name" />
          <YAxis />
          <Tooltip 
            formatter={(value: number) => [`${value} repos`, 'Count']}
          />
          <Bar dataKey="count" radius={[8, 8, 0, 0]}>
            {chartData.map((entry, index) => (
              <Cell key={`cell-${index}`} fill={entry.fill} />
            ))}
          </Bar>
        </BarChart>
      </ResponsiveContainer>
      
      {/* Legend */}
      <div className="mt-4 flex flex-wrap gap-4 justify-center">
        {chartData.map((item) => (
          <div key={item.category} className="flex items-center gap-2">
            <div 
              className="w-4 h-4 rounded" 
              style={{ backgroundColor: item.fill }}
            />
            <span className="text-sm text-gray-700">
              {item.name}: {item.count}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

