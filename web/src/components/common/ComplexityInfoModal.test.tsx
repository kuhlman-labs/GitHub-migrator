import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '../../__tests__/test-utils';
import { ComplexityInfoModal } from './ComplexityInfoModal';

describe('ComplexityInfoModal', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should render the trigger button', () => {
    render(<ComplexityInfoModal />);

    expect(screen.getByText('How is complexity calculated?')).toBeInTheDocument();
  });

  it('should open modal when trigger button is clicked', () => {
    render(<ComplexityInfoModal />);

    const triggerButton = screen.getByText('How is complexity calculated?');
    fireEvent.click(triggerButton);

    expect(screen.getByText('Repository Complexity Scoring')).toBeInTheDocument();
  });

  it('should close modal when close button is clicked', () => {
    render(<ComplexityInfoModal />);

    // Open modal
    const triggerButton = screen.getByText('How is complexity calculated?');
    fireEvent.click(triggerButton);

    // Verify modal is open
    expect(screen.getByText('Repository Complexity Scoring')).toBeInTheDocument();

    // Click close button
    const closeButton = screen.getByLabelText('Close');
    fireEvent.click(closeButton);

    // Modal should be closed
    expect(screen.queryByText('Repository Complexity Scoring')).not.toBeInTheDocument();
  });

  it('should close modal when "Got it!" button is clicked', () => {
    render(<ComplexityInfoModal />);

    // Open modal
    const triggerButton = screen.getByText('How is complexity calculated?');
    fireEvent.click(triggerButton);

    // Click Got it! button
    const gotItButton = screen.getByText('Got it!');
    fireEvent.click(gotItButton);

    // Modal should be closed
    expect(screen.queryByText('Repository Complexity Scoring')).not.toBeInTheDocument();
  });

  it('should close modal when backdrop is clicked', () => {
    render(<ComplexityInfoModal />);

    // Open modal
    const triggerButton = screen.getByText('How is complexity calculated?');
    fireEvent.click(triggerButton);

    // Click backdrop
    const backdrop = document.querySelector('.absolute.inset-0');
    if (backdrop) {
      fireEvent.click(backdrop);
    }

    // Modal should be closed
    expect(screen.queryByText('Repository Complexity Scoring')).not.toBeInTheDocument();
  });

  it('should show Azure DevOps specific content when source is azuredevops', () => {
    render(<ComplexityInfoModal source="azuredevops" />);

    const triggerButton = screen.getByText('How is complexity calculated?');
    fireEvent.click(triggerButton);

    expect(screen.getByText('Azure DevOps Migration Complexity')).toBeInTheDocument();
    expect(screen.getByText(/Azure DevOps-specific complexity score/)).toBeInTheDocument();
  });

  it('should show GitHub specific content when source is github', () => {
    render(<ComplexityInfoModal source="github" />);

    const triggerButton = screen.getByText('How is complexity calculated?');
    fireEvent.click(triggerButton);

    expect(screen.getByText('GitHub Migration Complexity')).toBeInTheDocument();
    expect(screen.getByText(/GitHub-specific complexity score/)).toBeInTheDocument();
  });

  it('should show GitHub specific sections for github source', () => {
    render(<ComplexityInfoModal source="github" />);

    const triggerButton = screen.getByText('How is complexity calculated?');
    fireEvent.click(triggerButton);

    // Should show GitHub-specific scoring factors
    expect(screen.getByText('Scoring Factors')).toBeInTheDocument();
    expect(screen.getByText('Repository Size')).toBeInTheDocument();
    expect(screen.getByText('High Impact (3-4 points each)')).toBeInTheDocument();
  });

  it('should show complexity categories section', () => {
    render(<ComplexityInfoModal />);

    const triggerButton = screen.getByText('How is complexity calculated?');
    fireEvent.click(triggerButton);

    expect(screen.getByText('Complexity Categories')).toBeInTheDocument();
    expect(screen.getByText('Simple')).toBeInTheDocument();
    expect(screen.getByText('Medium')).toBeInTheDocument();
    expect(screen.getByText('Complex')).toBeInTheDocument();
    expect(screen.getByText('Very Complex')).toBeInTheDocument();
  });

  it('should show Overview section', () => {
    render(<ComplexityInfoModal />);

    const triggerButton = screen.getByText('How is complexity calculated?');
    fireEvent.click(triggerButton);

    expect(screen.getByText('Overview')).toBeInTheDocument();
  });

  it('should show activity level note in Overview', () => {
    render(<ComplexityInfoModal />);

    const triggerButton = screen.getByText('How is complexity calculated?');
    fireEvent.click(triggerButton);

    expect(screen.getByText(/Activity levels are calculated using quantiles/)).toBeInTheDocument();
  });

  it('should use "all" as default source', () => {
    render(<ComplexityInfoModal />);

    const triggerButton = screen.getByText('How is complexity calculated?');
    fireEvent.click(triggerButton);

    // Default title for "all" source
    expect(screen.getByText('Repository Complexity Scoring')).toBeInTheDocument();
    expect(screen.getByText(/source-specific complexity scores/)).toBeInTheDocument();
  });
});

