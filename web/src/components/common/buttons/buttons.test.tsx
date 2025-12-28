import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { ThemeProvider } from '@primer/react';
import {
  SuccessButton,
  AttentionButton,
  BorderedButton,
  SecondaryButton,
  FilterDropdownButton,
} from './index';
import { XIcon } from '@primer/octicons-react';

// Helper to render components with ThemeProvider
const renderWithTheme = (component: React.ReactNode) => {
  return render(
    <ThemeProvider colorMode="light">
      {component}
    </ThemeProvider>
  );
};

describe('SuccessButton', () => {
  it('renders with children', () => {
    renderWithTheme(<SuccessButton>Start Migration</SuccessButton>);
    expect(screen.getByText('Start Migration')).toBeInTheDocument();
  });

  it('calls onClick when clicked', () => {
    const handleClick = vi.fn();
    renderWithTheme(<SuccessButton onClick={handleClick}>Click Me</SuccessButton>);
    fireEvent.click(screen.getByText('Click Me'));
    expect(handleClick).toHaveBeenCalledTimes(1);
  });

  it('is disabled when disabled prop is true', () => {
    renderWithTheme(<SuccessButton disabled>Disabled</SuccessButton>);
    expect(screen.getByText('Disabled').closest('button')).toBeDisabled();
  });

  it('applies custom sx styles', () => {
    renderWithTheme(<SuccessButton sx={{ marginTop: '10px' }}>Styled</SuccessButton>);
    expect(screen.getByText('Styled')).toBeInTheDocument();
  });
});

describe('AttentionButton', () => {
  it('renders with children', () => {
    renderWithTheme(<AttentionButton>Mark as Won&apos;t Migrate</AttentionButton>);
    expect(screen.getByText("Mark as Won't Migrate")).toBeInTheDocument();
  });

  it('calls onClick when clicked', () => {
    const handleClick = vi.fn();
    renderWithTheme(<AttentionButton onClick={handleClick}>Attention</AttentionButton>);
    fireEvent.click(screen.getByText('Attention'));
    expect(handleClick).toHaveBeenCalledTimes(1);
  });

  it('is disabled when disabled prop is true', () => {
    renderWithTheme(<AttentionButton disabled>Disabled</AttentionButton>);
    expect(screen.getByText('Disabled').closest('button')).toBeDisabled();
  });
});

describe('BorderedButton', () => {
  it('renders with children', () => {
    renderWithTheme(<BorderedButton>Export</BorderedButton>);
    expect(screen.getByText('Export')).toBeInTheDocument();
  });

  it('calls onClick when clicked', () => {
    const handleClick = vi.fn();
    renderWithTheme(<BorderedButton onClick={handleClick}>Click</BorderedButton>);
    fireEvent.click(screen.getByText('Click'));
    expect(handleClick).toHaveBeenCalledTimes(1);
  });

  it('supports leading visual', () => {
    renderWithTheme(<BorderedButton leadingVisual={XIcon}>With Icon</BorderedButton>);
    expect(screen.getByText('With Icon')).toBeInTheDocument();
  });
});

describe('SecondaryButton', () => {
  it('renders with children', () => {
    renderWithTheme(<SecondaryButton>Cancel</SecondaryButton>);
    expect(screen.getByText('Cancel')).toBeInTheDocument();
  });

  it('calls onClick when clicked', () => {
    const handleClick = vi.fn();
    renderWithTheme(<SecondaryButton onClick={handleClick}>Close</SecondaryButton>);
    fireEvent.click(screen.getByText('Close'));
    expect(handleClick).toHaveBeenCalledTimes(1);
  });

  it('is disabled when disabled prop is true', () => {
    renderWithTheme(<SecondaryButton disabled>Disabled</SecondaryButton>);
    expect(screen.getByText('Disabled').closest('button')).toBeDisabled();
  });
});

describe('FilterDropdownButton', () => {
  it('renders with children', () => {
    renderWithTheme(<FilterDropdownButton>All Organizations</FilterDropdownButton>);
    expect(screen.getByText('All Organizations')).toBeInTheDocument();
  });

  it('calls onClick when clicked', () => {
    const handleClick = vi.fn();
    renderWithTheme(<FilterDropdownButton onClick={handleClick}>Filter</FilterDropdownButton>);
    fireEvent.click(screen.getByText('Filter'));
    expect(handleClick).toHaveBeenCalledTimes(1);
  });

  it('is disabled when disabled prop is true', () => {
    renderWithTheme(<FilterDropdownButton disabled>Disabled</FilterDropdownButton>);
    expect(screen.getByText('Disabled').closest('button')).toBeDisabled();
  });
});

