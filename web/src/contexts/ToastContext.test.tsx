import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, act } from '@testing-library/react';
import { ThemeProvider } from '@primer/react';
import { ToastProvider, useToast } from './ToastContext';

// Test component that uses the toast context
function TestComponent() {
  const { showToast, showSuccess, showError, showWarning } = useToast();

  return (
    <div>
      <button onClick={() => showToast('Default message')}>Show Default</button>
      <button onClick={() => showSuccess('Success message')}>Show Success</button>
      <button onClick={() => showError('Error message')}>Show Error</button>
      <button onClick={() => showWarning('Warning message')}>Show Warning</button>
    </div>
  );
}

function TestWrapper({ children }: { children: React.ReactNode }) {
  return (
    <ThemeProvider>
      <ToastProvider>{children}</ToastProvider>
    </ThemeProvider>
  );
}

describe('ToastContext', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  describe('useToast hook', () => {
    it('should throw error when used outside ToastProvider', () => {
      const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {});
      
      expect(() => {
        render(
          <ThemeProvider>
            <TestComponent />
          </ThemeProvider>
        );
      }).toThrow('useToast must be used within ToastProvider');
      
      consoleError.mockRestore();
    });
  });

  describe('showToast', () => {
    it('should display a toast message', () => {
      render(
        <TestWrapper>
          <TestComponent />
        </TestWrapper>
      );

      fireEvent.click(screen.getByText('Show Default'));
      expect(screen.getByText('Default message')).toBeInTheDocument();
    });

    it('should auto-dismiss toast after 5 seconds', () => {
      render(
        <TestWrapper>
          <TestComponent />
        </TestWrapper>
      );

      fireEvent.click(screen.getByText('Show Default'));
      expect(screen.getByText('Default message')).toBeInTheDocument();

      act(() => {
        vi.advanceTimersByTime(5000);
      });

      expect(screen.queryByText('Default message')).not.toBeInTheDocument();
    });

    it('should allow manual dismissal of toast', () => {
      render(
        <TestWrapper>
          <TestComponent />
        </TestWrapper>
      );

      fireEvent.click(screen.getByText('Show Default'));
      expect(screen.getByText('Default message')).toBeInTheDocument();

      const dismissButton = screen.getByLabelText('Dismiss notification');
      fireEvent.click(dismissButton);

      expect(screen.queryByText('Default message')).not.toBeInTheDocument();
    });

    it('should display multiple toasts', () => {
      render(
        <TestWrapper>
          <TestComponent />
        </TestWrapper>
      );

      fireEvent.click(screen.getByText('Show Default'));
      fireEvent.click(screen.getByText('Show Success'));
      fireEvent.click(screen.getByText('Show Error'));

      expect(screen.getByText('Default message')).toBeInTheDocument();
      expect(screen.getByText('Success message')).toBeInTheDocument();
      expect(screen.getByText('Error message')).toBeInTheDocument();
    });
  });

  describe('showSuccess', () => {
    it('should display a success toast', () => {
      render(
        <TestWrapper>
          <TestComponent />
        </TestWrapper>
      );

      fireEvent.click(screen.getByText('Show Success'));
      expect(screen.getByText('Success message')).toBeInTheDocument();
    });
  });

  describe('showError', () => {
    it('should display an error toast', () => {
      render(
        <TestWrapper>
          <TestComponent />
        </TestWrapper>
      );

      fireEvent.click(screen.getByText('Show Error'));
      expect(screen.getByText('Error message')).toBeInTheDocument();
    });
  });

  describe('showWarning', () => {
    it('should display a warning toast', () => {
      render(
        <TestWrapper>
          <TestComponent />
        </TestWrapper>
      );

      fireEvent.click(screen.getByText('Show Warning'));
      expect(screen.getByText('Warning message')).toBeInTheDocument();
    });
  });

  describe('toast container', () => {
    it('should render toast container with accessibility attributes', () => {
      render(
        <TestWrapper>
          <TestComponent />
        </TestWrapper>
      );

      const container = screen.getByRole('region', { name: 'Notifications' });
      expect(container).toBeInTheDocument();
      expect(container).toHaveAttribute('aria-live', 'polite');
    });
  });
});

