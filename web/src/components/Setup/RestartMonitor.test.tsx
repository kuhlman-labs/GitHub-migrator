import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, act } from '../../__tests__/test-utils';
import { RestartMonitor } from './RestartMonitor';

describe('RestartMonitor', () => {
  const mockOnServerOnline = vi.fn();

  beforeEach(() => {
    vi.useFakeTimers();
    vi.clearAllMocks();
    global.fetch = vi.fn();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('renders restarting state initially', () => {
    render(<RestartMonitor onServerOnline={mockOnServerOnline} />);

    expect(screen.getByText(/Server Restarting/)).toBeInTheDocument();
    expect(screen.getByText('Please wait while the server restarts with new configuration')).toBeInTheDocument();
  });

  it('transitions to checking state after delay', async () => {
    render(<RestartMonitor onServerOnline={mockOnServerOnline} />);

    // Fast forward past the initial 5 second delay
    await act(async () => {
      vi.advanceTimersByTime(5000);
    });

    expect(screen.getByText(/Checking Server Status/)).toBeInTheDocument();
  });

  it('shows online state when health check succeeds', async () => {
    (global.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({ ok: true });

    render(<RestartMonitor onServerOnline={mockOnServerOnline} />);

    // Fast forward past the initial delay
    await act(async () => {
      vi.advanceTimersByTime(5000);
    });

    // Wait for health check interval
    await act(async () => {
      vi.advanceTimersByTime(2000);
    });

    expect(screen.getByText('Server Online!')).toBeInTheDocument();
  });

  it('calls onServerOnline after showing online state', async () => {
    (global.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({ ok: true });

    render(<RestartMonitor onServerOnline={mockOnServerOnline} />);

    // Fast forward past initial delay and health check
    await act(async () => {
      vi.advanceTimersByTime(5000);
    });

    await act(async () => {
      vi.advanceTimersByTime(2000);
    });

    // Fast forward for the redirect delay (1500ms)
    await act(async () => {
      vi.advanceTimersByTime(1500);
    });

    expect(mockOnServerOnline).toHaveBeenCalled();
  });
});

