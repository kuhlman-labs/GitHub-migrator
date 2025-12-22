import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, act } from '../../__tests__/test-utils';
import { RefreshIndicator } from './RefreshIndicator';

describe('RefreshIndicator', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  describe('when not refreshing', () => {
    it('should not render anything', () => {
      render(<RefreshIndicator isRefreshing={false} />);
      expect(screen.queryByRole('status')).not.toBeInTheDocument();
    });
  });

  describe('when refreshing', () => {
    it('should not show indicator immediately (before delay)', () => {
      render(<RefreshIndicator isRefreshing={true} />);
      
      // Should not be visible before delay
      expect(screen.queryByRole('status')).not.toBeInTheDocument();
    });

    it('should show indicator after default delay (500ms)', async () => {
      render(<RefreshIndicator isRefreshing={true} />);

      // Advance past the default 500ms delay
      await act(async () => {
        vi.advanceTimersByTime(500);
      });

      expect(screen.getByRole('status')).toBeInTheDocument();
    });

    it('should show indicator after custom delay', async () => {
      render(<RefreshIndicator isRefreshing={true} delay={1000} />);

      // Should not be visible before custom delay
      await act(async () => {
        vi.advanceTimersByTime(500);
      });
      expect(screen.queryByRole('status')).not.toBeInTheDocument();

      // Should be visible after custom delay
      await act(async () => {
        vi.advanceTimersByTime(500);
      });
      expect(screen.getByRole('status')).toBeInTheDocument();
    });

    it('should hide indicator when refreshing stops', async () => {
      const { rerender } = render(<RefreshIndicator isRefreshing={true} />);

      // Show the indicator
      await act(async () => {
        vi.advanceTimersByTime(500);
      });
      expect(screen.getByRole('status')).toBeInTheDocument();

      // Stop refreshing
      rerender(<RefreshIndicator isRefreshing={false} />);
      await act(async () => {
        vi.advanceTimersByTime(0);
      });
      expect(screen.queryByRole('status')).not.toBeInTheDocument();
    });
  });

  describe('subtle mode (default)', () => {
    it('should render subtle indicator by default', async () => {
      render(<RefreshIndicator isRefreshing={true} />);

      await act(async () => {
        vi.advanceTimersByTime(500);
      });

      expect(screen.getByRole('status')).toBeInTheDocument();
      expect(screen.getByLabelText('Refreshing data')).toBeInTheDocument();
      // Should not have the "Updating..." text in subtle mode
      expect(screen.queryByText('Updating...')).not.toBeInTheDocument();
    });
  });

  describe('non-subtle mode', () => {
    it('should render full indicator with text', async () => {
      render(<RefreshIndicator isRefreshing={true} subtle={false} />);

      await act(async () => {
        vi.advanceTimersByTime(500);
      });

      expect(screen.getByRole('status')).toBeInTheDocument();
      expect(screen.getByText('Updating...')).toBeInTheDocument();
    });
  });

  describe('accessibility', () => {
    it('should have role="status"', async () => {
      render(<RefreshIndicator isRefreshing={true} />);

      await act(async () => {
        vi.advanceTimersByTime(500);
      });

      expect(screen.getByRole('status')).toBeInTheDocument();
    });

    it('should have aria-live="polite"', async () => {
      render(<RefreshIndicator isRefreshing={true} />);

      await act(async () => {
        vi.advanceTimersByTime(500);
      });

      expect(screen.getByRole('status')).toHaveAttribute('aria-live', 'polite');
    });

    it('should have accessible spinner label', async () => {
      render(<RefreshIndicator isRefreshing={true} />);

      await act(async () => {
        vi.advanceTimersByTime(500);
      });

      expect(screen.getByLabelText('Refreshing data')).toBeInTheDocument();
    });
  });

  describe('edge cases', () => {
    it('should handle rapid refresh state changes', async () => {
      const { rerender } = render(<RefreshIndicator isRefreshing={true} />);

      // Rapidly toggle refresh state before delay completes
      await act(async () => {
        vi.advanceTimersByTime(200);
      });
      
      rerender(<RefreshIndicator isRefreshing={false} />);
      
      await act(async () => {
        vi.advanceTimersByTime(300);
      });
      
      // Should not show indicator since refresh stopped before delay
      expect(screen.queryByRole('status')).not.toBeInTheDocument();
    });

    it('should handle delay of 0', async () => {
      render(<RefreshIndicator isRefreshing={true} delay={0} />);

      await act(async () => {
        vi.advanceTimersByTime(0);
      });

      expect(screen.getByRole('status')).toBeInTheDocument();
    });
  });
});

