import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '../../__tests__/test-utils';
import { ConnectionTest } from './ConnectionTest';

describe('ConnectionTest', () => {
  it('renders with default label', () => {
    render(
      <ConnectionTest onTest={vi.fn()} />
    );

    expect(screen.getByRole('button', { name: 'Test Connection' })).toBeInTheDocument();
  });

  it('renders with custom label', () => {
    render(
      <ConnectionTest onTest={vi.fn()} label="Test Database" />
    );

    expect(screen.getByRole('button', { name: 'Test Database' })).toBeInTheDocument();
  });

  it('disables button when disabled prop is true', () => {
    render(
      <ConnectionTest onTest={vi.fn()} disabled={true} />
    );

    expect(screen.getByRole('button')).toBeDisabled();
  });

  it('shows loading state during test', async () => {
    const onTest = vi.fn(() => new Promise<{ valid: boolean }>((resolve) => {
      setTimeout(() => resolve({ valid: true }), 100);
    }));

    render(<ConnectionTest onTest={onTest} />);

    fireEvent.click(screen.getByRole('button'));

    expect(screen.getByText('Testing...')).toBeInTheDocument();
    expect(screen.getByRole('button')).toBeDisabled();

    await waitFor(() => {
      expect(screen.queryByText('Testing...')).not.toBeInTheDocument();
    });
  });

  it('displays success message when connection succeeds', async () => {
    const onTest = vi.fn().mockResolvedValue({ valid: true });

    render(<ConnectionTest onTest={onTest} />);

    fireEvent.click(screen.getByRole('button'));

    await waitFor(() => {
      expect(screen.getByText('Connection successful!')).toBeInTheDocument();
    });
  });

  it('displays error message when connection fails', async () => {
    const onTest = vi.fn().mockResolvedValue({
      valid: false,
      error: 'Invalid credentials',
    });

    render(<ConnectionTest onTest={onTest} />);

    fireEvent.click(screen.getByRole('button'));

    await waitFor(() => {
      expect(screen.getByText('Connection failed')).toBeInTheDocument();
      expect(screen.getByText('Invalid credentials')).toBeInTheDocument();
    });
  });

  it('handles thrown errors', async () => {
    const onTest = vi.fn().mockRejectedValue(new Error('Network error'));

    render(<ConnectionTest onTest={onTest} />);

    fireEvent.click(screen.getByRole('button'));

    await waitFor(() => {
      expect(screen.getByText('Connection failed')).toBeInTheDocument();
      expect(screen.getByText('Network error')).toBeInTheDocument();
    });
  });

  it('handles non-Error thrown objects', async () => {
    const onTest = vi.fn().mockRejectedValue('Something went wrong');

    render(<ConnectionTest onTest={onTest} />);

    fireEvent.click(screen.getByRole('button'));

    await waitFor(() => {
      expect(screen.getByText('Connection failed')).toBeInTheDocument();
      expect(screen.getByText('Connection test failed')).toBeInTheDocument();
    });
  });

  it('displays warnings when present', async () => {
    const onTest = vi.fn().mockResolvedValue({
      valid: true,
      warnings: ['Rate limit is near limit', 'API version is deprecated'],
    });

    render(<ConnectionTest onTest={onTest} />);

    fireEvent.click(screen.getByRole('button'));

    await waitFor(() => {
      expect(screen.getByText('Connection successful!')).toBeInTheDocument();
      expect(screen.getByText('Warnings')).toBeInTheDocument();
      expect(screen.getByText('Rate limit is near limit')).toBeInTheDocument();
      expect(screen.getByText('API version is deprecated')).toBeInTheDocument();
    });
  });

  it('displays details when present', async () => {
    const onTest = vi.fn().mockResolvedValue({
      valid: true,
      details: {
        'API Version': 'v3',
        'Rate Limit': '5000/5000',
      },
    });

    render(<ConnectionTest onTest={onTest} />);

    fireEvent.click(screen.getByRole('button'));

    await waitFor(() => {
      expect(screen.getByText('Connection successful!')).toBeInTheDocument();
      expect(screen.getByText(/API Version: v3/)).toBeInTheDocument();
      expect(screen.getByText(/Rate Limit: 5000\/5000/)).toBeInTheDocument();
    });
  });

  it('clears previous result when testing again', async () => {
    const onTest = vi.fn()
      .mockResolvedValueOnce({ valid: true })
      .mockResolvedValueOnce({ valid: false, error: 'Failed' });

    render(<ConnectionTest onTest={onTest} />);

    // First test - success
    fireEvent.click(screen.getByRole('button'));
    await waitFor(() => {
      expect(screen.getByText('Connection successful!')).toBeInTheDocument();
    });

    // Second test - failure
    fireEvent.click(screen.getByRole('button'));
    
    // During testing, result should be cleared
    await waitFor(() => {
      expect(screen.queryByText('Connection successful!')).not.toBeInTheDocument();
    });

    await waitFor(() => {
      expect(screen.getByText('Connection failed')).toBeInTheDocument();
    });
  });
});

