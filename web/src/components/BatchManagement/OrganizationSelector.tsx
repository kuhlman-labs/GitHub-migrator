import { useState, useRef, useEffect } from 'react';

interface OrganizationSelectorProps {
  organizations: string[];
  selectedOrganizations: string[];
  onChange: (selected: string[]) => void;
  loading?: boolean;
  placeholder?: string;
  searchPlaceholder?: string;
  emptyMessage?: string;
}

export function OrganizationSelector({
  organizations,
  selectedOrganizations,
  onChange,
  loading = false,
  placeholder = 'All Organizations',
  searchPlaceholder = 'Search organizations...',
  emptyMessage = 'No organizations found',
}: OrganizationSelectorProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const dropdownRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    };

    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside);
    }
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, [isOpen]);

  const filteredOrgs = organizations.filter((org) =>
    org.toLowerCase().includes(searchQuery.toLowerCase())
  );

  const handleToggle = (org: string) => {
    if (selectedOrganizations.includes(org)) {
      onChange(selectedOrganizations.filter((o) => o !== org));
    } else {
      onChange([...selectedOrganizations, org]);
    }
  };

  const handleSelectAll = () => {
    onChange(filteredOrgs);
  };

  const handleClearAll = () => {
    onChange([]);
  };

  return (
    <div className="relative" ref={dropdownRef}>
      <button
        type="button"
        onClick={() => setIsOpen(!isOpen)}
        className="w-full px-3 py-2 text-left border border-gray-300 rounded-lg bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-blue-500 transition-colors"
      >
        <div className="flex items-center justify-between">
          <span className="text-sm text-gray-700">
            {selectedOrganizations.length === 0
              ? placeholder
              : `${selectedOrganizations.length} selected`}
          </span>
          <svg
            className={`w-4 h-4 text-gray-500 transition-transform ${isOpen ? 'rotate-180' : ''}`}
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
          </svg>
        </div>
      </button>

      {isOpen && (
        <div className="absolute z-50 mt-1 w-full bg-white border border-gray-200 rounded-lg shadow-lg">
          <div className="p-2 border-b border-gray-200">
            <input
              type="text"
              placeholder={searchPlaceholder}
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full px-3 py-2 text-sm border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
              onClick={(e) => e.stopPropagation()}
            />
          </div>

          <div className="max-h-64 overflow-y-auto">
            {loading ? (
              <div className="p-4 text-center text-sm text-gray-500">Loading...</div>
            ) : filteredOrgs.length === 0 ? (
              <div className="p-4 text-center text-sm text-gray-500">{emptyMessage}</div>
            ) : (
              <>
                <div className="flex gap-2 p-2 border-b border-gray-100">
                  <button
                    onClick={handleSelectAll}
                    className="flex-1 px-2 py-1 text-xs text-blue-600 hover:bg-blue-50 rounded transition-colors"
                  >
                    Select All
                  </button>
                  {selectedOrganizations.length > 0 && (
                    <button
                      onClick={handleClearAll}
                      className="flex-1 px-2 py-1 text-xs text-gray-600 hover:bg-gray-50 rounded transition-colors"
                    >
                      Clear All
                    </button>
                  )}
                </div>
                <div className="p-1">
                  {filteredOrgs.map((org) => (
                    <label
                      key={org}
                      className="flex items-center gap-2 px-3 py-2 hover:bg-blue-50 rounded cursor-pointer transition-colors"
                    >
                      <input
                        type="checkbox"
                        checked={selectedOrganizations.includes(org)}
                        onChange={() => handleToggle(org)}
                        className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                      />
                      <span className="text-sm text-gray-700">{org}</span>
                    </label>
                  ))}
                </div>
              </>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

