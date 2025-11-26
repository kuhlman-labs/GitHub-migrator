import { useState, useRef, useEffect } from 'react';
import { ChevronDownIcon } from '@primer/octicons-react';

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
        className="w-full px-3 py-2 text-left rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 transition-opacity hover:opacity-80"
        style={{
          border: '1px solid var(--borderColor-default)',
          backgroundColor: 'var(--control-bgColor-rest)',
          color: 'var(--fgColor-default)'
        }}
      >
        <div className="flex items-center justify-between">
          <span className="text-sm">
            {selectedOrganizations.length === 0
              ? placeholder
              : `${selectedOrganizations.length} selected`}
          </span>
          <span style={{ color: 'var(--fgColor-muted)' }}>
          <ChevronDownIcon
              className={`transition-transform ${isOpen ? 'rotate-180' : ''}`}
            size={16}
          />
          </span>
        </div>
      </button>

      {isOpen && (
        <div 
          className="absolute z-50 mt-1 w-full rounded-lg shadow-lg"
          style={{
            backgroundColor: 'var(--bgColor-default)',
            border: '1px solid var(--borderColor-default)'
          }}
        >
          <div className="p-2" style={{ borderBottom: '1px solid var(--borderColor-default)' }}>
            <input
              type="text"
              placeholder={searchPlaceholder}
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full px-3 py-2 text-sm rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
              style={{
                border: '1px solid var(--borderColor-default)',
                backgroundColor: 'var(--control-bgColor-rest)',
                color: 'var(--fgColor-default)'
              }}
              onClick={(e) => e.stopPropagation()}
            />
          </div>

          <div className="max-h-64 overflow-y-auto">
            {loading ? (
              <div className="p-4 text-center text-sm" style={{ color: 'var(--fgColor-muted)' }}>Loading...</div>
            ) : filteredOrgs.length === 0 ? (
              <div className="p-4 text-center text-sm" style={{ color: 'var(--fgColor-muted)' }}>{emptyMessage}</div>
            ) : (
              <>
                <div className="flex gap-2 p-2" style={{ borderBottom: '1px solid var(--borderColor-muted)' }}>
                  <button
                    onClick={handleSelectAll}
                    className="flex-1 px-2 py-1 text-xs rounded transition-opacity hover:opacity-80"
                    style={{ color: 'var(--fgColor-accent)' }}
                  >
                    Select All
                  </button>
                  {selectedOrganizations.length > 0 && (
                    <button
                      onClick={handleClearAll}
                      className="flex-1 px-2 py-1 text-xs rounded transition-opacity hover:opacity-80"
                      style={{ color: 'var(--fgColor-default)' }}
                    >
                      Clear All
                    </button>
                  )}
                </div>
                <div className="p-1">
                  {filteredOrgs.map((org) => (
                    <label
                      key={org}
                      className="flex items-center gap-2 px-3 py-2 rounded cursor-pointer transition-opacity hover:opacity-80"
                    >
                      <input
                        type="checkbox"
                        checked={selectedOrganizations.includes(org)}
                        onChange={() => handleToggle(org)}
                        className="rounded text-blue-600 focus:ring-blue-500"
                        style={{ borderColor: 'var(--borderColor-default)' }}
                      />
                      <span className="text-sm" style={{ color: 'var(--fgColor-default)' }}>{org}</span>
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

