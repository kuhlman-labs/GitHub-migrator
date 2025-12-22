import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '../../__tests__/test-utils';
import { DropdownMenu } from './DropdownMenu';

describe('DropdownMenu', () => {
  const mockItems = [
    { label: 'Option 1', onClick: vi.fn() },
    { label: 'Option 2', onClick: vi.fn() },
    { label: 'Option 3', onClick: vi.fn(), disabled: true },
  ];

  const defaultProps = {
    label: 'Actions',
    items: mockItems,
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('rendering', () => {
    it('should render trigger button with label', () => {
      render(<DropdownMenu {...defaultProps} />);
      expect(screen.getByRole('button', { name: /actions/i })).toBeInTheDocument();
    });

    it('should not show menu initially', () => {
      render(<DropdownMenu {...defaultProps} />);
      expect(screen.queryByRole('menu')).not.toBeInTheDocument();
    });

    it('should render with leading icon', () => {
      render(
        <DropdownMenu
          {...defaultProps}
          leadingIcon={<span data-testid="icon">📥</span>}
        />
      );
      expect(screen.getByTestId('icon')).toBeInTheDocument();
    });

    it('should be disabled when disabled prop is true', () => {
      render(<DropdownMenu {...defaultProps} disabled />);
      expect(screen.getByRole('button', { name: /actions/i })).toBeDisabled();
    });
  });

  describe('menu interaction', () => {
    it('should open menu when trigger button is clicked', () => {
      render(<DropdownMenu {...defaultProps} />);
      
      fireEvent.click(screen.getByRole('button', { name: /actions/i }));
      expect(screen.getByRole('menu')).toBeInTheDocument();
    });

    it('should close menu when trigger button is clicked again', () => {
      render(<DropdownMenu {...defaultProps} />);
      const trigger = screen.getByRole('button', { name: /actions/i });
      
      fireEvent.click(trigger);
      expect(screen.getByRole('menu')).toBeInTheDocument();
      
      fireEvent.click(trigger);
      expect(screen.queryByRole('menu')).not.toBeInTheDocument();
    });

    it('should display menu items', () => {
      render(<DropdownMenu {...defaultProps} />);
      fireEvent.click(screen.getByRole('button', { name: /actions/i }));
      
      expect(screen.getByRole('menuitem', { name: 'Option 1' })).toBeInTheDocument();
      expect(screen.getByRole('menuitem', { name: 'Option 2' })).toBeInTheDocument();
      expect(screen.getByRole('menuitem', { name: 'Option 3' })).toBeInTheDocument();
    });

    it('should call onClick when menu item is clicked', () => {
      render(<DropdownMenu {...defaultProps} />);
      fireEvent.click(screen.getByRole('button', { name: /actions/i }));
      
      fireEvent.click(screen.getByRole('menuitem', { name: 'Option 1' }));
      expect(mockItems[0].onClick).toHaveBeenCalledTimes(1);
    });

    it('should close menu after item is clicked', () => {
      render(<DropdownMenu {...defaultProps} />);
      fireEvent.click(screen.getByRole('button', { name: /actions/i }));
      
      fireEvent.click(screen.getByRole('menuitem', { name: 'Option 1' }));
      expect(screen.queryByRole('menu')).not.toBeInTheDocument();
    });

    it('should not call onClick for disabled items', () => {
      render(<DropdownMenu {...defaultProps} />);
      fireEvent.click(screen.getByRole('button', { name: /actions/i }));
      
      fireEvent.click(screen.getByRole('menuitem', { name: 'Option 3' }));
      expect(mockItems[2].onClick).not.toHaveBeenCalled();
    });

    it('should close menu when Escape is pressed', () => {
      render(<DropdownMenu {...defaultProps} />);
      fireEvent.click(screen.getByRole('button', { name: /actions/i }));
      expect(screen.getByRole('menu')).toBeInTheDocument();
      
      fireEvent.keyDown(document, { key: 'Escape' });
      expect(screen.queryByRole('menu')).not.toBeInTheDocument();
    });

    it('should close menu when clicking outside', () => {
      render(
        <div>
          <DropdownMenu {...defaultProps} />
          <button data-testid="outside">Outside</button>
        </div>
      );
      
      fireEvent.click(screen.getByRole('button', { name: /actions/i }));
      expect(screen.getByRole('menu')).toBeInTheDocument();
      
      fireEvent.mouseDown(screen.getByTestId('outside'));
      expect(screen.queryByRole('menu')).not.toBeInTheDocument();
    });
  });

  describe('accessibility', () => {
    it('should have aria-haspopup="menu"', () => {
      render(<DropdownMenu {...defaultProps} />);
      expect(screen.getByRole('button', { name: /actions/i })).toHaveAttribute('aria-haspopup', 'menu');
    });

    it('should have aria-expanded="false" when closed', () => {
      render(<DropdownMenu {...defaultProps} />);
      expect(screen.getByRole('button', { name: /actions/i })).toHaveAttribute('aria-expanded', 'false');
    });

    it('should have aria-expanded="true" when open', () => {
      render(<DropdownMenu {...defaultProps} />);
      fireEvent.click(screen.getByRole('button', { name: /actions/i }));
      expect(screen.getByRole('button', { name: /actions/i })).toHaveAttribute('aria-expanded', 'true');
    });

    it('should have role="menu" on dropdown', () => {
      render(<DropdownMenu {...defaultProps} />);
      fireEvent.click(screen.getByRole('button', { name: /actions/i }));
      expect(screen.getByRole('menu')).toBeInTheDocument();
    });

    it('should have role="menuitem" on items', () => {
      render(<DropdownMenu {...defaultProps} />);
      fireEvent.click(screen.getByRole('button', { name: /actions/i }));
      expect(screen.getAllByRole('menuitem')).toHaveLength(3);
    });
  });
});

