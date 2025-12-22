import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '../../__tests__/test-utils';
import { RestartInstructions } from './RestartInstructions';

describe('RestartInstructions', () => {
  it('renders success heading', () => {
    render(<RestartInstructions />);

    expect(screen.getByText('Configuration Saved Successfully!')).toBeInTheDocument();
  });

  it('renders success flash message', () => {
    render(<RestartInstructions />);

    expect(screen.getByText(/configuration has been saved to the/i)).toBeInTheDocument();
  });

  it('renders next steps section', () => {
    render(<RestartInstructions />);

    expect(screen.getByText('Next Steps')).toBeInTheDocument();
    expect(screen.getByText(/restart the server/i)).toBeInTheDocument();
  });

  it('shows development restart command', () => {
    render(<RestartInstructions />);

    expect(screen.getByText('Development:')).toBeInTheDocument();
    expect(screen.getByText('make run-server')).toBeInTheDocument();
  });

  it('shows Docker restart command', () => {
    render(<RestartInstructions />);

    expect(screen.getByText('Docker:')).toBeInTheDocument();
    expect(screen.getByText('docker-compose restart')).toBeInTheDocument();
  });

  it('shows systemd restart command', () => {
    render(<RestartInstructions />);

    expect(screen.getByText('Production (systemd):')).toBeInTheDocument();
    expect(screen.getByText('systemctl restart github-migrator')).toBeInTheDocument();
  });

  it('renders Go to Dashboard button', () => {
    render(<RestartInstructions />);

    expect(screen.getByRole('button', { name: 'Go to Dashboard' })).toBeInTheDocument();
  });

  it('redirects to dashboard when clicking button', () => {
    // Mock window.location.href
    const originalLocation = window.location;
    const mockHref = vi.fn();
    Object.defineProperty(window, 'location', {
      value: { href: '' },
      writable: true,
    });
    Object.defineProperty(window.location, 'href', {
      set: mockHref,
    });

    render(<RestartInstructions />);

    fireEvent.click(screen.getByRole('button', { name: 'Go to Dashboard' }));

    // Restore original location
    window.location = originalLocation;
  });
});

