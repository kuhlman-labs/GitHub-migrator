import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '../../__tests__/test-utils';
import { BatchCard } from './BatchCard';
import type { Batch } from '../../types';

// Helper to create a mock batch
function createMockBatch(overrides: Partial<Batch> = {}): Batch {
  return {
    id: 1,
    name: 'Test Batch',
    description: 'A test batch for migration',
    type: 'manual',
    repository_count: 10,
    status: 'pending',
    created_at: '2024-01-01T00:00:00Z',
    ...overrides,
  };
}

describe('BatchCard', () => {
  const defaultProps = {
    batch: createMockBatch(),
    isSelected: false,
    onClick: vi.fn(),
    onStart: vi.fn(),
  };

  it('should render batch name', () => {
    render(<BatchCard {...defaultProps} />);
    expect(screen.getByText('Test Batch')).toBeInTheDocument();
  });

  it('should render repository count', () => {
    render(<BatchCard {...defaultProps} />);
    expect(screen.getByText('10 repos')).toBeInTheDocument();
  });

  it('should render status badge', () => {
    render(<BatchCard {...defaultProps} />);
    expect(screen.getByText('pending')).toBeInTheDocument();
  });

  it('should call onClick when card is clicked', () => {
    const onClick = vi.fn();
    render(<BatchCard {...defaultProps} onClick={onClick} />);
    
    fireEvent.click(screen.getByText('Test Batch').closest('div')!);
    
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  describe('ready status', () => {
    it('should show Start button when status is ready', () => {
      render(
        <BatchCard
          {...defaultProps}
          batch={createMockBatch({ status: 'ready' })}
        />
      );
      
      expect(screen.getByRole('button', { name: 'Start' })).toBeInTheDocument();
    });

    it('should call onStart when Start button is clicked', () => {
      const onStart = vi.fn();
      render(
        <BatchCard
          {...defaultProps}
          batch={createMockBatch({ status: 'ready' })}
          onStart={onStart}
        />
      );
      
      fireEvent.click(screen.getByRole('button', { name: 'Start' }));
      
      expect(onStart).toHaveBeenCalledTimes(1);
    });

    it('should not propagate click event from Start button', () => {
      const onClick = vi.fn();
      const onStart = vi.fn();
      render(
        <BatchCard
          {...defaultProps}
          batch={createMockBatch({ status: 'ready' })}
          onClick={onClick}
          onStart={onStart}
        />
      );
      
      fireEvent.click(screen.getByRole('button', { name: 'Start' }));
      
      expect(onStart).toHaveBeenCalledTimes(1);
      expect(onClick).not.toHaveBeenCalled();
    });
  });

  describe('pending status', () => {
    it('should show "Dry run needed" message for pending status', () => {
      render(
        <BatchCard
          {...defaultProps}
          batch={createMockBatch({ status: 'pending' })}
        />
      );
      
      expect(screen.getByText('Dry run needed')).toBeInTheDocument();
    });

    it('should not show Start button for pending status', () => {
      render(
        <BatchCard
          {...defaultProps}
          batch={createMockBatch({ status: 'pending' })}
        />
      );
      
      expect(screen.queryByRole('button', { name: 'Start' })).not.toBeInTheDocument();
    });
  });

  describe('scheduled_at', () => {
    it('should display scheduled date when provided', () => {
      render(
        <BatchCard
          {...defaultProps}
          batch={createMockBatch({ scheduled_at: '2024-06-15T10:00:00Z' })}
        />
      );
      
      // The date should be formatted and displayed
      const dateElements = screen.getByText(/2024/);
      expect(dateElements).toBeInTheDocument();
    });

    it('should not display scheduled date when not provided', () => {
      render(
        <BatchCard
          {...defaultProps}
          batch={createMockBatch({ scheduled_at: undefined })}
        />
      );
      
      // No calendar icon should be visible (part of the date display)
      expect(screen.queryByRole('img')).not.toBeInTheDocument();
    });
  });

  describe('selection state', () => {
    it('should have different styling when selected', () => {
      const { container } = render(
        <BatchCard {...defaultProps} isSelected={true} />
      );
      
      // Find the card div (first child after ThemeProvider wrappers)
      const card = container.querySelector('.p-4.rounded-lg') as HTMLElement;
      // Check that the style attribute contains the accent color
      expect(card.getAttribute('style')).toContain('accent-emphasis');
    });

    it('should have default styling when not selected', () => {
      const { container } = render(
        <BatchCard {...defaultProps} isSelected={false} />
      );
      
      const card = container.querySelector('.p-4.rounded-lg') as HTMLElement;
      expect(card.getAttribute('style')).toContain('borderColor-default');
    });
  });

  describe('other statuses', () => {
    const statuses = ['in_progress', 'completed', 'failed', 'cancelled'] as const;
    
    statuses.forEach((status) => {
      it(`should not show Start button for ${status} status`, () => {
        render(
          <BatchCard
            {...defaultProps}
            batch={createMockBatch({ status })}
          />
        );
        
        expect(screen.queryByRole('button', { name: 'Start' })).not.toBeInTheDocument();
      });
    });
  });
});

