import { describe, it, expect } from 'vitest';
import { render, screen } from '../../__tests__/test-utils';
import { LoadingSpinner } from './LoadingSpinner';

describe('LoadingSpinner', () => {
  it('should render a spinner', () => {
    render(<LoadingSpinner />);
    expect(screen.getByRole('status')).toBeInTheDocument();
  });

  it('should have accessible aria-live attribute', () => {
    render(<LoadingSpinner />);
    const status = screen.getByRole('status');
    expect(status).toHaveAttribute('aria-live', 'polite');
  });

  it('should have loading label for accessibility', () => {
    render(<LoadingSpinner />);
    expect(screen.getByLabelText('Loading content')).toBeInTheDocument();
  });
});

