import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '../../__tests__/test-utils';
import userEvent from '@testing-library/user-event';
import { SetupWizard } from './SetupWizard';

// Mock child components that might have complex dependencies
vi.mock('./ConnectionTest', () => ({
  ConnectionTest: ({ onValidated }: { onValidated: (valid: boolean) => void }) => (
    <button onClick={() => onValidated(true)}>Test Connection (mocked)</button>
  ),
}));

vi.mock('./RestartMonitor', () => ({
  RestartMonitor: () => <div data-testid="restart-monitor">Restart Monitor (mocked)</div>,
}));

describe('SetupWizard', () => {
  it('renders the wizard', () => {
    const { container } = render(<SetupWizard />);

    // Wizard should render
    expect(container).toBeDefined();
  });

  it('displays Next button', () => {
    render(<SetupWizard />);

    expect(screen.getByRole('button', { name: /Next/i })).toBeInTheDocument();
  });

  it('can navigate steps', async () => {
    const user = userEvent.setup();
    render(<SetupWizard />);

    // Click Next
    const nextButton = screen.getByRole('button', { name: /Next/i });
    await user.click(nextButton);

    // Back button should now be visible
    expect(screen.getByRole('button', { name: /Back/i })).toBeInTheDocument();
  });
});

