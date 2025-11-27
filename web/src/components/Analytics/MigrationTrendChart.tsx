import { XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Area, AreaChart } from 'recharts';
import { MigrationTimeSeriesPoint } from '../../types';

interface MigrationTrendChartProps {
  data: MigrationTimeSeriesPoint[];
}

export function MigrationTrendChart({ data }: MigrationTrendChartProps) {
  if (!data || data.length === 0) {
    return (
      <div className="rounded-lg shadow-sm p-6" style={{ backgroundColor: 'var(--bgColor-default)' }}>
        <h2 className="text-lg font-medium mb-4" style={{ color: 'var(--fgColor-default)' }}>Migration Trend (Last 30 Days)</h2>
        <p className="text-sm mb-4" style={{ color: 'var(--fgColor-muted)' }}>Daily migration activity showing velocity trends and patterns to forecast completion timelines.</p>
        <div className="h-[300px] flex items-center justify-center" style={{ color: 'var(--fgColor-muted)' }}>
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
    <div className="rounded-lg shadow-sm p-6" style={{ backgroundColor: 'var(--bgColor-default)' }}>
      <h2 className="text-lg font-medium mb-4" style={{ color: 'var(--fgColor-default)' }}>Migration Trend (Last 30 Days)</h2>
      <p className="text-sm mb-4" style={{ color: 'var(--fgColor-muted)' }}>Daily migration activity showing velocity trends and patterns to forecast completion timelines.</p>
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

