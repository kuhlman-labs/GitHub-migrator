import { useState, useRef } from 'react';
import { Button } from '@primer/react';
import { UploadIcon } from '@primer/octicons-react';
import { parseImportFile, type ImportParseResult } from '../../utils/import';
import { LoadingSpinner } from '../common/LoadingSpinner';

interface ImportDialogProps {
  onImport: (parseResult: ImportParseResult) => void;
  onCancel: () => void;
}

export function ImportDialog({ onImport, onCancel }: ImportDialogProps) {
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [parsing, setParsing] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleFileSelect = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (file) {
      setSelectedFile(file);
      setError(null);
    }
  };

  const handleBrowseClick = () => {
    fileInputRef.current?.click();
  };

  const handleDrop = (event: React.DragEvent) => {
    event.preventDefault();
    const file = event.dataTransfer.files?.[0];
    if (file) {
      setSelectedFile(file);
      setError(null);
    }
  };

  const handleDragOver = (event: React.DragEvent) => {
    event.preventDefault();
  };

  const handleImport = async () => {
    if (!selectedFile) {
      setError('Please select a file');
      return;
    }

    setParsing(true);
    setError(null);

    try {
      const result = await parseImportFile(selectedFile);
      
      if (!result.success || result.errors.length > 0) {
        setError(result.errors.join('\n'));
        setParsing(false);
        return;
      }

      if (result.rows.length === 0) {
        setError('No repositories found in file');
        setParsing(false);
        return;
      }

      // Pass the result to parent for validation
      onImport(result);
    } catch (err: any) {
      setError(err.message || 'Failed to parse file');
      setParsing(false);
    }
  };

  const getFileIcon = () => {
    if (!selectedFile) return null;
    
    const ext = selectedFile.name.split('.').pop()?.toLowerCase();
    switch (ext) {
      case 'csv':
        return '📄';
      case 'xlsx':
      case 'xls':
        return '📊';
      case 'json':
        return '📋';
      default:
        return '📁';
    }
  };

  return (
    <div 
      className="fixed inset-0 flex items-center justify-center z-[100]"
      style={{ backgroundColor: 'rgba(0, 0, 0, 0.6)' }}
    >
      <div 
        className="relative rounded-lg shadow-xl max-w-2xl w-full mx-4"
        style={{ 
          backgroundColor: 'var(--bgColor-default)',
          border: '1px solid var(--borderColor-default)'
        }}
      >
        {/* Header */}
        <div 
          className="px-6 py-4 border-b"
          style={{ borderColor: 'var(--borderColor-default)' }}
        >
          <h2 className="text-xl font-semibold" style={{ color: 'var(--fgColor-default)' }}>
            Import Repositories from File
          </h2>
          <p className="text-sm mt-1" style={{ color: 'var(--fgColor-muted)' }}>
            Upload a CSV, Excel, or JSON file containing repository names
          </p>
        </div>

        {/* Content */}
        <div className="p-6">
          {/* File Upload Area */}
          <div
            onDrop={handleDrop}
            onDragOver={handleDragOver}
            className="border-2 border-dashed rounded-lg p-8 text-center transition-colors hover:bg-[var(--control-bgColor-hover)]"
            style={{ borderColor: 'var(--borderColor-default)' }}
          >
            <input
              ref={fileInputRef}
              type="file"
              accept=".csv,.xlsx,.xls,.json"
              onChange={handleFileSelect}
              className="hidden"
            />

            {selectedFile ? (
              <div className="space-y-3">
                <div className="text-4xl">{getFileIcon()}</div>
                <div>
                  <div className="font-medium" style={{ color: 'var(--fgColor-default)' }}>
                    {selectedFile.name}
                  </div>
                  <div className="text-sm mt-1" style={{ color: 'var(--fgColor-muted)' }}>
                    {(selectedFile.size / 1024).toFixed(1)} KB
                  </div>
                </div>
                <Button onClick={handleBrowseClick}>
                  Choose Different File
                </Button>
              </div>
            ) : (
              <div className="space-y-3">
                <UploadIcon size={48} className="mx-auto" />
                <div>
                  <div className="font-medium mb-2" style={{ color: 'var(--fgColor-default)' }}>
                    Drop file here or click to browse
                  </div>
                  <div className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>
                    Supported formats: CSV, Excel (.xlsx), JSON
                  </div>
                </div>
                <Button onClick={handleBrowseClick} leadingVisual={UploadIcon}>
                  Choose File
                </Button>
              </div>
            )}
          </div>

          {/* File Format Info */}
          <div 
            className="mt-4 p-4 rounded-lg text-sm"
            style={{ 
              backgroundColor: 'var(--bgColor-muted)',
              color: 'var(--fgColor-muted)'
            }}
          >
            <div className="font-medium mb-2" style={{ color: 'var(--fgColor-default)' }}>
              Required column:
            </div>
            <div className="mb-3">
              • <code className="px-1.5 py-0.5 rounded" style={{ backgroundColor: 'var(--bgColor-emphasis)' }}>repository</code> or <code className="px-1.5 py-0.5 rounded" style={{ backgroundColor: 'var(--bgColor-emphasis)' }}>full_name</code> - Repository full name (org/repo)
            </div>
            <div 
              className="mt-3 p-3 rounded border text-sm"
              style={{ 
                backgroundColor: 'var(--bgColor-default)',
                borderColor: 'var(--borderColor-accent-emphasis)',
                color: 'var(--fgColor-muted)'
              }}
            >
              <div className="font-medium mb-1" style={{ color: 'var(--fgColor-accent)' }}>
                ℹ️ Migration Settings
              </div>
              Migration settings (destination organization, migration API, exclusions, etc.) are configured at the batch level and will apply to all imported repositories.
            </div>
          </div>

          {/* Error Display */}
          {error && (
            <div 
              className="mt-4 p-4 rounded-lg text-sm whitespace-pre-wrap"
              style={{
                backgroundColor: 'var(--danger-subtle)',
                border: '1px solid var(--borderColor-danger)',
                color: 'var(--fgColor-danger)'
              }}
            >
              {error}
            </div>
          )}

          {/* Loading State */}
          {parsing && (
            <div className="mt-4 flex items-center justify-center gap-3 p-4">
              <LoadingSpinner />
              <span style={{ color: 'var(--fgColor-muted)' }}>Parsing file...</span>
            </div>
          )}
        </div>

        {/* Footer */}
        <div 
          className="px-6 py-4 border-t flex justify-end gap-2"
          style={{ 
            borderColor: 'var(--borderColor-default)',
            backgroundColor: 'var(--bgColor-muted)'
          }}
        >
          <Button onClick={onCancel} disabled={parsing}>
            Cancel
          </Button>
          <Button
            variant="primary"
            onClick={handleImport}
            disabled={!selectedFile || parsing}
          >
            {parsing ? 'Parsing...' : 'Import'}
          </Button>
        </div>
      </div>
    </div>
  );
}

