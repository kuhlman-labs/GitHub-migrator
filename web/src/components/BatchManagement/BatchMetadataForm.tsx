import { ChevronDownIcon } from '@primer/octicons-react';
import { formatDateForInput } from '../../utils/format';
import { Button, SuccessButton } from '../common/buttons';

interface MigrationSettings {
  destinationOrg: string;
  migrationAPI: 'GEI' | 'ELM';
  excludeReleases: boolean;
  excludeAttachments: boolean;
}

interface BatchMetadataFormProps {
  batchName: string;
  setBatchName: (name: string) => void;
  batchDescription: string;
  setBatchDescription: (description: string) => void;
  scheduledAt: string;
  setScheduledAt: (date: string) => void;
  migrationSettings: MigrationSettings;
  onMigrationSettingsChange: (settings: Partial<MigrationSettings>) => void;
  showMigrationSettings: boolean;
  setShowMigrationSettings: (show: boolean) => void;
  organizations: string[];
  loading: boolean;
  isEditMode: boolean;
  currentBatchReposCount: number;
  error: string | null;
  onSave: (startImmediately: boolean) => void;
  onClose: () => void;
}

export function BatchMetadataForm({
  batchName,
  setBatchName,
  batchDescription,
  setBatchDescription,
  scheduledAt,
  setScheduledAt,
  migrationSettings,
  onMigrationSettingsChange,
  showMigrationSettings,
  setShowMigrationSettings,
  organizations,
  loading,
  isEditMode,
  currentBatchReposCount,
  error,
  onSave,
  onClose,
}: BatchMetadataFormProps) {
  const { destinationOrg, migrationAPI, excludeReleases, excludeAttachments } = migrationSettings;

  const configuredSettingsCount = [
    destinationOrg ? 1 : 0,
    excludeReleases ? 1 : 0,
    excludeAttachments ? 1 : 0,
    migrationAPI !== 'GEI' ? 1 : 0,
  ].reduce((a, b) => a + b, 0);

  return (
    <div 
      className="flex-shrink-0 shadow-[0_-4px_6px_-1px_rgba(0,0,0,0.1)]"
      style={{ 
        borderTop: '1px solid var(--borderColor-default)',
        backgroundColor: 'var(--bgColor-default)' 
      }}
    >
      {/* Essential Fields - Always Visible */}
      <div className="p-3 space-y-2.5">
        <div>
          <label className="block text-xs font-semibold mb-1" style={{ color: 'var(--fgColor-default)' }}>
            Batch Name *
          </label>
          <input
            type="text"
            value={batchName}
            onChange={(e) => setBatchName(e.target.value)}
            placeholder="e.g., Wave 1, Q1 Migration"
            className="w-full px-2.5 py-1.5 text-sm rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            style={{
              border: '1px solid var(--borderColor-default)',
              backgroundColor: 'var(--control-bgColor-rest)',
              color: 'var(--fgColor-default)'
            }}
            disabled={loading}
            required
          />
        </div>

        <div>
          <label className="block text-xs font-semibold mb-1" style={{ color: 'var(--fgColor-default)' }}>
            Description
          </label>
          <textarea
            value={batchDescription}
            onChange={(e) => setBatchDescription(e.target.value)}
            placeholder="Optional description"
            rows={1}
            className="w-full px-2.5 py-1.5 text-sm rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent resize-y"
            style={{
              border: '1px solid var(--borderColor-default)',
              backgroundColor: 'var(--control-bgColor-rest)',
              color: 'var(--fgColor-default)'
            }}
            disabled={loading}
          />
        </div>
      </div>

      {/* Collapsible Migration Settings */}
      <div style={{ borderTop: '1px solid var(--borderColor-default)' }}>
        <button
          type="button"
          onClick={() => setShowMigrationSettings(!showMigrationSettings)}
          className="w-full px-3 py-2.5 flex items-center justify-between text-sm font-medium hover:opacity-80 transition-opacity"
          style={{ color: 'var(--fgColor-default)' }}
        >
          <div className="flex items-center gap-2">
            <svg className="w-4 h-4" style={{ color: 'var(--fgColor-muted)' }} fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
            </svg>
            <span>Migration Settings</span>
            {configuredSettingsCount > 0 && (
              <span 
                className="px-1.5 py-0.5 text-xs rounded-full font-medium"
                style={{
                  backgroundColor: 'var(--accent-subtle)',
                  color: 'var(--fgColor-accent)'
                }}
              >
                {configuredSettingsCount} configured
              </span>
            )}
          </div>
          <span style={{ color: 'var(--fgColor-muted)' }}>
            <ChevronDownIcon
              className={`transition-transform ${showMigrationSettings ? 'rotate-180' : ''}`}
              size={20}
            />
          </span>
        </button>

        {showMigrationSettings && (
          <div 
            className="p-3 space-y-2.5"
            style={{ 
              backgroundColor: 'var(--bgColor-muted)',
              borderTop: '1px solid var(--borderColor-default)' 
            }}
          >
            <div>
              <label className="block text-xs font-semibold mb-1" style={{ color: 'var(--fgColor-default)' }}>
                Destination Organization
                <span className="ml-1 font-normal text-xs" style={{ color: 'var(--fgColor-muted)' }}>— Default for repos without specific destination</span>
              </label>
              <input
                type="text"
                value={destinationOrg}
                onChange={(e) => onMigrationSettingsChange({ destinationOrg: e.target.value })}
                placeholder="Leave blank to use source org"
                list="organizations-list"
                className="w-full px-2.5 py-1.5 text-sm rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                style={{
                  border: '1px solid var(--borderColor-default)',
                  backgroundColor: 'var(--control-bgColor-rest)',
                  color: 'var(--fgColor-default)'
                }}
                disabled={loading}
              />
              <datalist id="organizations-list">
                {organizations.map((org) => (
                  <option key={org} value={org} />
                ))}
              </datalist>
            </div>

            <div>
              <label className="block text-xs font-semibold mb-1" style={{ color: 'var(--fgColor-default)' }}>
                Migration API
              </label>
              <select
                value={migrationAPI}
                onChange={(e) => onMigrationSettingsChange({ migrationAPI: e.target.value as 'GEI' | 'ELM' })}
                className="w-full px-2.5 py-1.5 text-sm rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                style={{
                  border: '1px solid var(--borderColor-default)',
                  backgroundColor: 'var(--control-bgColor-rest)',
                  color: 'var(--fgColor-default)'
                }}
                disabled={loading}
              >
                <option value="GEI">GEI (GitHub Enterprise Importer)</option>
                <option value="ELM">ELM (Enterprise Live Migrator) - Future</option>
              </select>
            </div>

            <div className="flex items-start gap-2">
              <input
                type="checkbox"
                id="exclude-releases"
                checked={excludeReleases}
                onChange={(e) => onMigrationSettingsChange({ excludeReleases: e.target.checked })}
                className="mt-0.5 h-4 w-4 rounded text-blue-600 focus:ring-2 focus:ring-blue-500"
                style={{ borderColor: 'var(--borderColor-default)' }}
                disabled={loading}
              />
              <label htmlFor="exclude-releases" className="text-xs cursor-pointer" style={{ color: 'var(--fgColor-default)' }}>
                <span className="font-semibold">Exclude Releases</span>
                <span className="block mt-0.5" style={{ color: 'var(--fgColor-muted)' }}>Skip releases during migration (repo settings override)</span>
              </label>
            </div>

            <div className="flex items-start gap-2">
              <input
                type="checkbox"
                id="exclude-attachments"
                checked={excludeAttachments}
                onChange={(e) => onMigrationSettingsChange({ excludeAttachments: e.target.checked })}
                className="mt-0.5 h-4 w-4 rounded text-blue-600 focus:ring-2 focus:ring-blue-500"
                style={{ borderColor: 'var(--borderColor-default)' }}
                disabled={loading}
              />
              <label htmlFor="exclude-attachments" className="text-xs cursor-pointer" style={{ color: 'var(--fgColor-default)' }}>
                <span className="font-semibold">Exclude Attachments</span>
                <span className="block mt-0.5" style={{ color: 'var(--fgColor-muted)' }}>Skip file attachments (images, files attached to Issues/PRs) to reduce archive size (repo settings override)</span>
              </label>
            </div>
          </div>
        )}
      </div>

      {/* Scheduled Date Section */}
      <div className="p-3" style={{ borderTop: '1px solid var(--borderColor-default)' }}>
        <div className="relative z-[60]">
          <label className="block text-xs font-semibold mb-1" style={{ color: 'var(--fgColor-default)' }}>
            Scheduled Date (Optional)
          </label>
          <input
            type="datetime-local"
            value={scheduledAt}
            onChange={(e) => setScheduledAt(e.target.value)}
            min={formatDateForInput(new Date().toISOString())}
            className="w-full px-2.5 py-1.5 text-sm rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            style={{
              border: '1px solid var(--borderColor-default)',
              backgroundColor: 'var(--control-bgColor-rest)',
              color: 'var(--fgColor-default)'
            }}
            disabled={loading}
          />
        </div>
      </div>

      {/* Error Message */}
      {error && (
        <div className="px-3 pb-3">
          <div 
            className="p-2 rounded text-sm"
            style={{
              backgroundColor: 'var(--danger-subtle)',
              color: 'var(--fgColor-danger)',
              border: '1px solid var(--borderColor-danger)'
            }}
          >
            {error}
          </div>
        </div>
      )}

      {/* Action Buttons */}
      <div className="p-3 flex gap-2" style={{ borderTop: '1px solid var(--borderColor-default)' }}>
        <Button
          onClick={onClose}
          disabled={loading}
        >
          Cancel
        </Button>
        {isEditMode ? (
          <SuccessButton
            onClick={() => onSave(false)}
            disabled={loading || !batchName.trim() || currentBatchReposCount === 0}
            className="flex-1"
          >
            {loading ? 'Saving...' : 'Update Batch'}
          </SuccessButton>
        ) : (
          <>
            <Button
              onClick={() => onSave(false)}
              disabled={loading || !batchName.trim() || currentBatchReposCount === 0}
              variant="primary"
              className="flex-1"
            >
              {loading ? 'Saving...' : 'Create Batch'}
            </Button>
            <SuccessButton
              onClick={() => onSave(true)}
              disabled={loading || !batchName.trim() || currentBatchReposCount === 0}
              className="flex-1"
            >
              {loading ? 'Starting...' : 'Create & Start'}
            </SuccessButton>
          </>
        )}
      </div>
    </div>
  );
}

