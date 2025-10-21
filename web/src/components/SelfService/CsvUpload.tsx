import { useState, useRef } from 'react';

interface ParsedCsvData {
  repositories: string[];
  mappings?: Record<string, string>;
}

interface CsvUploadProps {
  onDataParsed: (data: ParsedCsvData) => void;
  disabled?: boolean;
}

export function CsvUpload({ onDataParsed, disabled = false }: CsvUploadProps) {
  const [isDragging, setIsDragging] = useState(false);
  const [parseError, setParseError] = useState<string | null>(null);
  const [previewData, setPreviewData] = useState<ParsedCsvData | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const parseCsvContent = (content: string): ParsedCsvData | null => {
    const lines = content
      .split('\n')
      .map(line => line.trim())
      .filter(line => line.length > 0);

    if (lines.length === 0) {
      setParseError('CSV file is empty');
      return null;
    }

    const repositories: string[] = [];
    const mappings: Record<string, string> = {};
    let hasMappings = false;

    for (let i = 0; i < lines.length; i++) {
      const line = lines[i];
      const lineNumber = i + 1;

      // Skip CSV header if present
      if (i === 0 && (line.toLowerCase().includes('source') || line.toLowerCase().includes('destination'))) {
        continue;
      }

      // Check if line contains comma (mapping format)
      if (line.includes(',')) {
        const parts = line.split(',').map(p => p.trim());
        
        if (parts.length !== 2) {
          setParseError(`Line ${lineNumber}: Invalid format. Expected "source_org/repo,dest_org/repo"`);
          return null;
        }

        const [source, dest] = parts;

        if (!source.includes('/')) {
          setParseError(`Line ${lineNumber}: Invalid source format "${source}". Expected "org/repo"`);
          return null;
        }

        if (!dest.includes('/')) {
          setParseError(`Line ${lineNumber}: Invalid destination format "${dest}". Expected "org/repo"`);
          return null;
        }

        repositories.push(source);
        mappings[source] = dest;
        hasMappings = true;
      } else {
        // Simple list format
        if (!line.includes('/')) {
          setParseError(`Line ${lineNumber}: Invalid format "${line}". Expected "org/repo"`);
          return null;
        }
        repositories.push(line);
      }
    }

    if (repositories.length === 0) {
      setParseError('No valid repositories found in CSV');
      return null;
    }

    setParseError(null);
    return {
      repositories,
      mappings: hasMappings ? mappings : undefined,
    };
  };

  const handleFileSelect = (file: File) => {
    if (!file.name.endsWith('.csv')) {
      setParseError('Please upload a CSV file');
      return;
    }

    const reader = new FileReader();
    reader.onload = (e) => {
      const content = e.target?.result as string;
      const data = parseCsvContent(content);
      if (data) {
        setPreviewData(data);
      }
    };
    reader.onerror = () => {
      setParseError('Failed to read file');
    };
    reader.readAsText(file);
  };

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
    if (!disabled) {
      setIsDragging(true);
    }
  };

  const handleDragLeave = (e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(false);
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(false);

    if (disabled) return;

    const file = e.dataTransfer.files[0];
    if (file) {
      handleFileSelect(file);
    }
  };

  const handleFileInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      handleFileSelect(file);
    }
  };

  const handleUseData = () => {
    if (previewData) {
      onDataParsed(previewData);
      setPreviewData(null);
      setParseError(null);
      if (fileInputRef.current) {
        fileInputRef.current.value = '';
      }
    }
  };

  const handleClear = () => {
    setPreviewData(null);
    setParseError(null);
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  };

  return (
    <div className="space-y-4">
      {/* Drag & Drop Area */}
      <div
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onDrop={handleDrop}
        className={`
          border-2 border-dashed rounded-lg p-8 text-center transition-colors
          ${isDragging ? 'border-blue-500 bg-blue-50' : 'border-gray-300 bg-gray-50'}
          ${disabled ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer hover:border-blue-400'}
        `}
      >
        <input
          ref={fileInputRef}
          type="file"
          accept=".csv"
          onChange={handleFileInputChange}
          disabled={disabled}
          className="hidden"
          id="csv-file-input"
        />
        
        <svg
          className="mx-auto h-12 w-12 text-gray-400"
          stroke="currentColor"
          fill="none"
          viewBox="0 0 48 48"
          aria-hidden="true"
        >
          <path
            d="M28 8H12a4 4 0 00-4 4v20m32-12v8m0 0v8a4 4 0 01-4 4H12a4 4 0 01-4-4v-4m32-4l-3.172-3.172a4 4 0 00-5.656 0L28 28M8 32l9.172-9.172a4 4 0 015.656 0L28 28m0 0l4 4m4-24h8m-4-4v8m-12 4h.02"
            strokeWidth={2}
            strokeLinecap="round"
            strokeLinejoin="round"
          />
        </svg>
        
        <div className="mt-4">
          <label
            htmlFor="csv-file-input"
            className="cursor-pointer font-medium text-blue-600 hover:text-blue-500"
          >
            Upload a CSV file
          </label>
          <p className="text-sm text-gray-500 mt-1">or drag and drop</p>
        </div>
        
        <p className="text-xs text-gray-500 mt-2">
          CSV file with repository names
        </p>
      </div>

      {/* Format Instructions */}
      <div className="bg-blue-50 border border-blue-200 rounded-lg p-4 text-sm">
        <h4 className="font-medium text-blue-900 mb-2">Supported CSV Formats:</h4>
        <div className="space-y-2 text-blue-800">
          <div>
            <strong>Simple list:</strong>
            <code className="block bg-white px-2 py-1 rounded mt-1 font-mono text-xs">
              org/repo1<br />
              org/repo2<br />
              org/repo3
            </code>
          </div>
          <div>
            <strong>With destination mappings:</strong>
            <code className="block bg-white px-2 py-1 rounded mt-1 font-mono text-xs">
              source-org/repo1,dest-org/repo1<br />
              source-org/repo2,other-org/repo2
            </code>
          </div>
        </div>
      </div>

      {/* Error Display */}
      {parseError && (
        <div className="bg-red-50 border border-red-200 rounded-lg p-4">
          <div className="flex">
            <div className="flex-shrink-0">
              <svg className="h-5 w-5 text-red-400" viewBox="0 0 20 20" fill="currentColor">
                <path
                  fillRule="evenodd"
                  d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
                  clipRule="evenodd"
                />
              </svg>
            </div>
            <div className="ml-3">
              <h3 className="text-sm font-medium text-red-800">CSV Parse Error</h3>
              <div className="mt-2 text-sm text-red-700">{parseError}</div>
            </div>
          </div>
        </div>
      )}

      {/* Preview Display */}
      {previewData && !parseError && (
        <div className="bg-green-50 border border-green-200 rounded-lg p-4">
          <div className="flex items-start justify-between mb-3">
            <div>
              <h3 className="text-sm font-medium text-green-900">CSV Parsed Successfully</h3>
              <p className="text-sm text-green-700 mt-1">
                Found {previewData.repositories.length} {previewData.repositories.length === 1 ? 'repository' : 'repositories'}
                {previewData.mappings && ` with custom destinations`}
              </p>
            </div>
          </div>

          {/* Preview List */}
          <div className="mt-3 max-h-60 overflow-y-auto bg-white rounded border border-green-200 p-3">
            <div className="space-y-1 text-sm font-mono">
              {previewData.repositories.slice(0, 10).map((repo, index) => (
                <div key={index} className="text-gray-700">
                  {repo}
                  {previewData.mappings?.[repo] && (
                    <span className="text-green-600"> → {previewData.mappings[repo]}</span>
                  )}
                </div>
              ))}
              {previewData.repositories.length > 10 && (
                <div className="text-gray-500 italic">
                  ... and {previewData.repositories.length - 10} more
                </div>
              )}
            </div>
          </div>

          {/* Action Buttons */}
          <div className="flex gap-3 mt-4">
            <button
              onClick={handleUseData}
              className="flex-1 px-4 py-2 bg-green-600 text-white rounded-md text-sm font-medium hover:bg-green-700 transition-colors"
            >
              Use This Data
            </button>
            <button
              onClick={handleClear}
              className="px-4 py-2 border border-gray-300 text-gray-700 rounded-md text-sm font-medium hover:bg-gray-50 transition-colors"
            >
              Clear
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

