import { XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Area, AreaChart } from 'recharts';
import { MigrationTimeSeriesPoint } from '../../types';

interface MigrationTrendChartProps {
  data: MigrationTimeSeriesPoint[];
}

export function MigrationTrendChart({ data }: MigrationTrendChartProps) {
  if (!data || data.length === 0) {
    return (
      <div className="bg-white rounded-lg shadow-sm p-6">
        <h2 className="text-lg font-medium text-gray-900 mb-4">Migration Trend (Last 30 Days)</h2>
        <div className="h-[300px] flex items-center justify-center text-gray-500">
          No migration data available for the selected period
        </div>
      </div>
    );
  }

  // Format data for display
  const chartData = data.map(point => ({
    ...point,
    displayDate: new Date(point.date).toLocaleDateString('en-US', { month: 'short', day: 'numeric' }),
  }));

  return (
    <div className="bg-white rounded-lg shadow-sm p-6">
      <h2 className="text-lg font-medium text-gray-900 mb-4">Migration Trend (Last 30 Days)</h2>
      <ResponsiveContainer width="100%" height={300}>
        <AreaChart data={chartData}>
          <defs>
            <linearGradient id="colorCount" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="#10B981" stopOpacity={0.8}/>
              <stop offset="95%" stopColor="#10B981" stopOpacity={0.1}/>
            </linearGradient>
          </defs>
          <CartesianGrid strokeDasharray="3 3" />
          <XAxis 
            dataKey="displayDate" 
            angle={-45} 
            textAnchor="end" 
            height={80}
            tick={{ fontSize: 12 }}
          />
          <YAxis />
          <Tooltip 
            labelFormatter={(label) => `Date: ${label}`}
            formatter={(value: number) => [`${value} repos`, 'Migrations']}
          />
          <Area 
            type="monotone" 
            dataKey="count" 
            stroke="#10B981" 
            strokeWidth={2}
            fillOpacity={1}
            fill="url(#colorCount)" 
          />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  );
}

