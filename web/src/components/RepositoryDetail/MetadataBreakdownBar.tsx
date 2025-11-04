import { formatBytes } from '../../utils/format';

interface MetadataBreakdownBarProps {
  releases: number;
  issues: number;
  prs: number;
  attachments: number;
  total: number;
  limit: number;
}

export function MetadataBreakdownBar(props: MetadataBreakdownBarProps) {
  const percentage = (value: number) => {
    if (props.total === 0) return 0;
    return (value / props.total) * 100;
  };
  
  const limitPercentage = props.limit > 0 ? ((props.total / props.limit) * 100).toFixed(1) : '0';
  
  return (
    <div className="space-y-3">
      {/* Visual breakdown bar */}
      <div className="h-6 w-full bg-gray-200 rounded-lg overflow-hidden flex">
        {percentage(props.releases) > 0 && (
          <div 
            className="bg-purple-500 flex items-center justify-center text-xs text-white font-medium"
            style={{ width: `${percentage(props.releases)}%` }}
            title={`Releases: ${formatBytes(props.releases)}`}
          >
            {percentage(props.releases) > 15 && `${percentage(props.releases).toFixed(0)}%`}
          </div>
        )}
        {percentage(props.issues) > 0 && (
          <div 
            className="bg-blue-500 flex items-center justify-center text-xs text-white font-medium"
            style={{ width: `${percentage(props.issues)}%` }}
            title={`Issues: ${formatBytes(props.issues)}`}
          >
            {percentage(props.issues) > 15 && `${percentage(props.issues).toFixed(0)}%`}
          </div>
        )}
        {percentage(props.prs) > 0 && (
          <div 
            className="bg-green-500 flex items-center justify-center text-xs text-white font-medium"
            style={{ width: `${percentage(props.prs)}%` }}
            title={`Pull Requests: ${formatBytes(props.prs)}`}
          >
            {percentage(props.prs) > 15 && `${percentage(props.prs).toFixed(0)}%`}
          </div>
        )}
        {percentage(props.attachments) > 0 && (
          <div 
            className="bg-orange-500 flex items-center justify-center text-xs text-white font-medium"
            style={{ width: `${percentage(props.attachments)}%` }}
            title={`Attachments: ${formatBytes(props.attachments)}`}
          >
            {percentage(props.attachments) > 15 && `${percentage(props.attachments).toFixed(0)}%`}
          </div>
        )}
      </div>
      
      {/* Breakdown legend */}
      <div className="grid grid-cols-2 gap-2 text-sm">
        <div className="flex items-center gap-2">
          <span className="w-3 h-3 bg-purple-500 rounded flex-shrink-0"></span>
          <span className="text-gray-700">
            Releases: <span className="font-medium">{formatBytes(props.releases)}</span> ({percentage(props.releases).toFixed(0)}%)
          </span>
        </div>
        <div className="flex items-center gap-2">
          <span className="w-3 h-3 bg-blue-500 rounded flex-shrink-0"></span>
          <span className="text-gray-700">
            Issues: <span className="font-medium">{formatBytes(props.issues)}</span> ({percentage(props.issues).toFixed(0)}%)
          </span>
        </div>
        <div className="flex items-center gap-2">
          <span className="w-3 h-3 bg-green-500 rounded flex-shrink-0"></span>
          <span className="text-gray-700">
            PRs: <span className="font-medium">{formatBytes(props.prs)}</span> ({percentage(props.prs).toFixed(0)}%)
          </span>
        </div>
        <div className="flex items-center gap-2">
          <span className="w-3 h-3 bg-orange-500 rounded flex-shrink-0"></span>
          <span className="text-gray-700">
            Attachments: <span className="font-medium">{formatBytes(props.attachments)}</span> ({percentage(props.attachments).toFixed(0)}%)
          </span>
        </div>
      </div>
      
      {/* Total summary */}
      <div className="pt-2 border-t border-gray-200">
        <div className="flex items-center justify-between">
          <span className="font-medium text-gray-900">Total Metadata Size (Estimated):</span>
          <span className="font-semibold text-gray-900">
            {formatBytes(props.total)} / {formatBytes(props.limit)} ({limitPercentage}% of limit)
          </span>
        </div>
      </div>
    </div>
  );
}

