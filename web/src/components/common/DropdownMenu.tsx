import { useState, useRef, useEffect } from 'react';
import { Button } from '@primer/react';
import { ChevronDownIcon } from '@primer/octicons-react';

export interface DropdownMenuItem {
  label: string;
  onClick: () => void;
  disabled?: boolean;
}

export interface DropdownMenuProps {
  label: string;
  items: DropdownMenuItem[];
  leadingIcon?: React.ReactNode;
  disabled?: boolean;
  variant?: 'default' | 'primary' | 'invisible';
}

/**
 * A reusable dropdown menu component.
 * Use for action menus like export options, bulk actions, etc.
 *
 * @example
 * <DropdownMenu
 *   label="Export"
 *   leadingIcon={<DownloadIcon size={16} />}
 *   items={[
 *     { label: 'Export as CSV', onClick: () => handleExport('csv') },
 *     { label: 'Export as Excel', onClick: () => handleExport('excel') },
 *     { label: 'Export as JSON', onClick: () => handleExport('json') },
 *   ]}
 * />
 */
export function DropdownMenu({
  label,
  items,
  leadingIcon,
  disabled = false,
  variant = 'invisible',
}: DropdownMenuProps) {
  const [isOpen, setIsOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);
  const buttonRef = useRef<HTMLButtonElement>(null);

  // Close menu when clicking outside
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setIsOpen(false);
      }
    };

    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside);
    }
    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
    };
  }, [isOpen]);

  // Handle escape key
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isOpen) {
        setIsOpen(false);
        buttonRef.current?.focus();
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [isOpen]);

  const handleItemClick = (item: DropdownMenuItem) => {
    if (!item.disabled) {
      item.onClick();
      setIsOpen(false);
    }
  };

  return (
    <div ref={menuRef} className="relative">
      <Button
        ref={buttonRef}
        variant={variant}
        onClick={() => setIsOpen(!isOpen)}
        disabled={disabled}
        leadingVisual={() => leadingIcon || null}
        trailingVisual={ChevronDownIcon}
        aria-expanded={isOpen}
        aria-haspopup="menu"
      >
        {label}
      </Button>

      {isOpen && (
        <div
          className="absolute right-0 mt-2 w-48 rounded-lg shadow-lg z-20"
          style={{
            backgroundColor: 'var(--bgColor-default)',
            border: '1px solid var(--borderColor-default)',
            boxShadow: 'var(--shadow-floating-large)',
          }}
          role="menu"
        >
          <div className="py-1">
            {items.map((item, index) => (
              <button
                key={index}
                onClick={() => handleItemClick(item)}
                disabled={item.disabled}
                className="w-full text-left px-4 py-2 text-sm transition-colors hover:bg-[var(--control-bgColor-hover)] disabled:opacity-50 disabled:cursor-not-allowed"
                style={{ color: 'var(--fgColor-default)' }}
                role="menuitem"
              >
                {item.label}
              </button>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

