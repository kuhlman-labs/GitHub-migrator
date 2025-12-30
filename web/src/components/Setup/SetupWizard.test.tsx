import React, { useState } from 'react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '../../__tests__/test-utils';
import userEvent from '@testing-library/user-event';
import { SetupWizard } from './SetupWizard';

// Mock the API module
vi.mock('../../services/api', () => ({
  api: {
    validateDatabaseConnection: vi.fn().mockResolvedValue({ valid: true }),
    applySetup: vi.fn().mockResolvedValue({}),
  },
}));

// Mock child components that might have complex dependencies
vi.mock('./ConnectionTest', () => ({
  ConnectionTest: ({ onTest, label, disabled }: { onTest: () => Promise<{ valid: boolean }>; label?: string; disabled?: boolean }) => {
    const [tested, setTested] = useState(false);
    return (
      <button 
        disabled={disabled}
        onClick={async () => {
          try {
            await onTest();
            setTested(true);
          } catch {
            // Ignore errors in test
          }
        }}
      >
        {tested ? 'Connection Valid' : (label || 'Test Connection')}
      </button>
    );
  },
}));

vi.mock('./RestartMonitor', () => ({
  RestartMonitor: () => <div data-testid="restart-monitor">Restart Monitor (mocked)</div>,
}));

describe('SetupWizard', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders the wizard with Database step', () => {
    render(<SetupWizard />);

    // Should show Database Configuration heading
    expect(screen.getByText('Database Configuration')).toBeInTheDocument();
  });

  it('displays Continue button', () => {
    render(<SetupWizard />);

    expect(screen.getByRole('button', { name: /Continue/i })).toBeInTheDocument();
  });

  it('displays Back button as disabled on first step', () => {
    render(<SetupWizard />);

    const backButton = screen.getByRole('button', { name: /Back/i });
    expect(backButton).toBeDisabled();
  });

  it('shows database type selector', () => {
    render(<SetupWizard />);

    // Should have database type selector
    expect(screen.getByLabelText(/Database Type/i)).toBeInTheDocument();
  });

  it('shows SQLite as default database type', () => {
    render(<SetupWizard />);

    // SQLite should be the default option
    expect(screen.getByText(/SQLite.*Recommended/i)).toBeInTheDocument();
  });

  it('can navigate to step 2 after validating database', async () => {
    const user = userEvent.setup();
    render(<SetupWizard />);

    // Click Test Connection to validate (will trigger the mock)
    const testButton = screen.getByRole('button', { name: /Test Database Connection/i });
    await user.click(testButton);

    // Wait for state to update
    await waitFor(() => {
      expect(screen.getByText('Connection Valid')).toBeInTheDocument();
    });

    // Click Continue
    const continueButton = screen.getByRole('button', { name: /Continue/i });
    await user.click(continueButton);

    // Should now show Server Configuration
    await waitFor(() => {
      expect(screen.getByText('Server Configuration')).toBeInTheDocument();
    });
  });

  it('shows Apply & Restart on step 2', async () => {
    const user = userEvent.setup();
    render(<SetupWizard />);

    // Click Test Connection to validate
    const testButton = screen.getByRole('button', { name: /Test Database Connection/i });
    await user.click(testButton);

    // Wait for validation
    await waitFor(() => {
      expect(screen.getByText('Connection Valid')).toBeInTheDocument();
    });

    // Click Continue to go to step 2
    const continueButton = screen.getByRole('button', { name: /Continue/i });
    await user.click(continueButton);

    // Should show Apply & Restart button on step 2
    await waitFor(() => {
      expect(screen.getByRole('button', { name: /Apply & Restart/i })).toBeInTheDocument();
    });
  });

  it('can go back from step 2', async () => {
    const user = userEvent.setup();
    render(<SetupWizard />);

    // Click Test Connection to validate
    const testButton = screen.getByRole('button', { name: /Test Database Connection/i });
    await user.click(testButton);

    // Wait for validation
    await waitFor(() => {
      expect(screen.getByText('Connection Valid')).toBeInTheDocument();
    });

    // Click Continue to go to step 2
    const continueButton = screen.getByRole('button', { name: /Continue/i });
    await user.click(continueButton);

    // Wait for step 2
    await waitFor(() => {
      expect(screen.getByText('Server Configuration')).toBeInTheDocument();
    });

    // Click Back
    const backButton = screen.getByRole('button', { name: /Back/i });
    expect(backButton).not.toBeDisabled();
    await user.click(backButton);

    // Should be back on step 1
    await waitFor(() => {
      expect(screen.getByText('Database Configuration')).toBeInTheDocument();
    });
  });
});
