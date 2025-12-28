import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '../../__tests__/test-utils';
import { CollapsibleValidationSection } from './CollapsibleValidationSection';

describe('CollapsibleValidationSection', () => {
  const defaultProps = {
    id: 'test-section',
    title: 'Test Section',
    status: 'passed' as const,
    expanded: false,
    onToggle: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders section title', () => {
    render(
      <CollapsibleValidationSection {...defaultProps}>
        <p>Content</p>
      </CollapsibleValidationSection>
    );

    expect(screen.getByText('Test Section')).toBeInTheDocument();
  });

  it('hides content when collapsed', () => {
    render(
      <CollapsibleValidationSection {...defaultProps} expanded={false}>
        <p>Hidden Content</p>
      </CollapsibleValidationSection>
    );

    expect(screen.queryByText('Hidden Content')).not.toBeInTheDocument();
  });

  it('shows content when expanded', () => {
    render(
      <CollapsibleValidationSection {...defaultProps} expanded={true}>
        <p>Visible Content</p>
      </CollapsibleValidationSection>
    );

    expect(screen.getByText('Visible Content')).toBeInTheDocument();
  });

  it('calls onToggle when clicked', () => {
    const mockOnToggle = vi.fn();

    render(
      <CollapsibleValidationSection {...defaultProps} onToggle={mockOnToggle}>
        <p>Content</p>
      </CollapsibleValidationSection>
    );

    fireEvent.click(screen.getByRole('button'));
    expect(mockOnToggle).toHaveBeenCalled();
  });

  it('sets aria-expanded correctly when collapsed', () => {
    render(
      <CollapsibleValidationSection {...defaultProps} expanded={false}>
        <p>Content</p>
      </CollapsibleValidationSection>
    );

    expect(screen.getByRole('button')).toHaveAttribute('aria-expanded', 'false');
  });

  it('sets aria-expanded correctly when expanded', () => {
    render(
      <CollapsibleValidationSection {...defaultProps} expanded={true}>
        <p>Content</p>
      </CollapsibleValidationSection>
    );

    expect(screen.getByRole('button')).toHaveAttribute('aria-expanded', 'true');
  });

  it('renders blocking status styling', () => {
    const { container } = render(
      <CollapsibleValidationSection {...defaultProps} status="blocking">
        <p>Content</p>
      </CollapsibleValidationSection>
    );

    const section = container.querySelector('.border-red-200');
    expect(section).toBeInTheDocument();
  });

  it('renders warning status styling', () => {
    const { container } = render(
      <CollapsibleValidationSection {...defaultProps} status="warning">
        <p>Content</p>
      </CollapsibleValidationSection>
    );

    const section = container.querySelector('.border-yellow-200');
    expect(section).toBeInTheDocument();
  });

  it('renders passed status styling', () => {
    const { container } = render(
      <CollapsibleValidationSection {...defaultProps} status="passed">
        <p>Content</p>
      </CollapsibleValidationSection>
    );

    const section = container.querySelector('.border-green-200');
    expect(section).toBeInTheDocument();
  });

  it('renders SVG icon for each status', () => {
    const { container: blockingContainer } = render(
      <CollapsibleValidationSection {...defaultProps} status="blocking">
        <p>Content</p>
      </CollapsibleValidationSection>
    );
    expect(blockingContainer.querySelector('svg')).toBeInTheDocument();

    const { container: warningContainer } = render(
      <CollapsibleValidationSection {...defaultProps} status="warning">
        <p>Content</p>
      </CollapsibleValidationSection>
    );
    expect(warningContainer.querySelector('svg')).toBeInTheDocument();

    const { container: passedContainer } = render(
      <CollapsibleValidationSection {...defaultProps} status="passed">
        <p>Content</p>
      </CollapsibleValidationSection>
    );
    expect(passedContainer.querySelector('svg')).toBeInTheDocument();
  });
});

