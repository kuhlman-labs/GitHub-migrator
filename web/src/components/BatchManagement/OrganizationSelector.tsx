import { useState, useRef, useEffect, useCallback } from 'react';
import { createPortal } from 'react-dom';
import { ChevronDownIcon } from '@primer/octicons-react';

interface OrganizationSelectorProps {
  organizations: string[];
  selectedOrganizations: string[];
  onChange: (selected: string[]) => void;
  loading?: boolean;
  placeholder?: string;
  searchPlaceholder?: string;
  emptyMessage?: string;
  renderLabel?: (value: string) => string;
  useFixedPosition?: boolean;
}

export function OrganizationSelector({
  organizations,
  selectedOrganizations,
  onChange,
  loading = false,
  placeholder = 'All Organizations',
  searchPlaceholder = 'Search organizations...',
  emptyMessage = 'No organizations found',
  renderLabel,
  useFixedPosition = false,
}: OrganizationSelectorProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [dropdownPosition, setDropdownPosition] = useState({ top: 0, left: 0, width: 0 });
  const buttonRef = useRef<HTMLButtonElement>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);

  const updatePosition = useCallback(() => {
    if (buttonRef.current) {
      const rect = buttonRef.current.getBoundingClientRect();
      setDropdownPosition({
        top: rect.bottom + 4,
        left: rect.left,
        width: rect.width,
      });
    }
  }, []); // No dependencies - function reference stays stable

  useEffect(() => {
    if (isOpen) {
      updatePosition();
      window.addEventListener('scroll', updatePosition, true);
      window.addEventListener('resize', updatePosition);
      return () => {
        window.removeEventListener('scroll', updatePosition, true);
        window.removeEventListener('resize', updatePosition);
      };
    }
  }, [isOpen, updatePosition]);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      const target = event.target as HTMLElement;
      
      // Check if click is inside button
      if (buttonRef.current?.contains(target)) {
        return;
      }
      
      // Check if click is inside dropdown (works with portal)
      // When using portal, dropdownRef.current is in document.body
      if (dropdownRef.current?.contains(target)) {
        return;
      }
      
      // Also check by traversing up from target to see if we're in the dropdown
      let element: HTMLElement | null = target;
      while (element) {
        if (element === dropdownRef.current) {
          return;
        }
        element = element.parentElement;
      }
      
      setIsOpen(false);
    };

    if (isOpen) {
      // Use setTimeout to avoid the click that opened the dropdown from closing it
      const timer = setTimeout(() => {
        document.addEventListener('mousedown', handleClickOutside);
      }, 10);
      return () => {
        clearTimeout(timer);
        document.removeEventListener('mousedown', handleClickOutside);
      };
    }
  }, [isOpen]);

  const filteredOrgs = organizations.filter((org) =>
    org.toLowerCase().includes(searchQuery.toLowerCase())
  );

  const handleToggle = (org: string) => {
    const newSelection = selectedOrganizations.includes(org)
      ? selectedOrganizations.filter((o) => o !== org)
      : [...selectedOrganizations, org];
    onChange(newSelection);
  };

  const handleSelectAll = () => {
    onChange(filteredOrgs);
  };

  const handleClearAll = () => {
    onChange([]);
  };

  const dropdownContent = (
    <div 
      ref={dropdownRef}
      onMouseDown={(e) => e.stopPropagation()}
      style={{
        position: useFixedPosition ? 'fixed' : 'absolute',
        top: useFixedPosition ? dropdownPosition.top : '100%',
        left: useFixedPosition ? dropdownPosition.left : 0,
        width: useFixedPosition ? dropdownPosition.width : '100%',
        marginTop: useFixedPosition ? 0 : 4,
        zIndex: 99999,
        backgroundColor: 'var(--bgColor-default)',
        border: '1px solid var(--borderColor-default)',
        borderRadius: '0.5rem',
        boxShadow: '0 10px 15px -3px rgba(0, 0, 0, 0.3), 0 4px 6px -2px rgba(0, 0, 0, 0.2)',
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
                type="button"
                onClick={() => handleSelectAll()}
                className="flex-1 px-2 py-1 text-xs rounded transition-opacity hover:opacity-80"
                style={{ color: 'var(--fgColor-accent)' }}
              >
                Select All
              </button>
              {selectedOrganizations.length > 0 && (
                <button
                  type="button"
                  onClick={() => handleClearAll()}
                  className="flex-1 px-2 py-1 text-xs rounded transition-opacity hover:opacity-80"
                  style={{ color: 'var(--fgColor-default)' }}
                >
                  Clear All
                </button>
              )}
            </div>
            <div className="p-1">
              {filteredOrgs.map((org) => (
                <div
                  key={org}
                  className="flex items-center gap-2 px-3 py-2 rounded cursor-pointer hover:bg-gray-700"
                  onClick={() => handleToggle(org)}
                  role="option"
                  aria-selected={selectedOrganizations.includes(org)}
                >
                  <input
                    type="checkbox"
                    checked={selectedOrganizations.includes(org)}
                    onChange={() => handleToggle(org)}
                    className="rounded text-blue-600 focus:ring-blue-500"
                    style={{ borderColor: 'var(--borderColor-default)' }}
                  />
                  <span className="text-sm" style={{ color: 'var(--fgColor-default)' }}>
                    {renderLabel ? renderLabel(org) : org}
                  </span>
                </div>
              ))}
            </div>
          </>
        )}
      </div>
    </div>
  );

  return (
    <div className="relative">
      <button
        ref={buttonRef}
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
        useFixedPosition 
          ? createPortal(dropdownContent, document.body)
          : dropdownContent
      )}
    </div>
  );
}
