import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '../../__tests__/test-utils';
import { Setup } from './index';

// Mock the SetupWizard component to avoid complex test setup
vi.mock('./SetupWizard', () => ({
  SetupWizard: () => <div data-testid="setup-wizard">Setup Wizard Content</div>,
}));

describe('Setup', () => {
  it('renders the page header with GitHub Migrator title', () => {
    render(<Setup />);

    expect(screen.getByRole('heading', { name: /GitHub Migrator/i })).toBeInTheDocument();
  });

  it('renders the Setup subtitle', () => {
    render(<Setup />);

    expect(screen.getByRole('heading', { name: /Setup/i })).toBeInTheDocument();
  });

  it('renders the GitHub icon', () => {
    const { container } = render(<Setup />);

    // The MarkGithubIcon should be present
    const icon = container.querySelector('svg');
    expect(icon).toBeInTheDocument();
  });

  it('renders the SetupWizard component', () => {
    render(<Setup />);

    expect(screen.getByTestId('setup-wizard')).toBeInTheDocument();
  });
});

