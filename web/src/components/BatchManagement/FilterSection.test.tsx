import { describe, it, expect } from 'vitest';
import { render, screen, fireEvent } from '../../__tests__/test-utils';
import { FilterSection } from './FilterSection';

describe('FilterSection', () => {
  it('should render section title', () => {
    render(
      <FilterSection title="Test Section">
        <div>Content</div>
      </FilterSection>
    );

    expect(screen.getByText('Test Section')).toBeInTheDocument();
  });

  it('should show content by default when defaultExpanded is true', () => {
    render(
      <FilterSection title="Test Section" defaultExpanded={true}>
        <div>Test Content</div>
      </FilterSection>
    );

    expect(screen.getByText('Test Content')).toBeInTheDocument();
  });

  it('should hide content by default when defaultExpanded is false', () => {
    render(
      <FilterSection title="Test Section" defaultExpanded={false}>
        <div>Hidden Content</div>
      </FilterSection>
    );

    expect(screen.queryByText('Hidden Content')).not.toBeInTheDocument();
  });

  it('should toggle content visibility when button is clicked', () => {
    render(
      <FilterSection title="Test Section" defaultExpanded={true}>
        <div>Toggle Content</div>
      </FilterSection>
    );

    // Initially visible
    expect(screen.getByText('Toggle Content')).toBeInTheDocument();

    // Click to collapse
    const toggleButton = screen.getByRole('button');
    fireEvent.click(toggleButton);

    // Now hidden
    expect(screen.queryByText('Toggle Content')).not.toBeInTheDocument();

    // Click to expand
    fireEvent.click(toggleButton);

    // Visible again
    expect(screen.getByText('Toggle Content')).toBeInTheDocument();
  });

  it('should use true as default for defaultExpanded', () => {
    render(
      <FilterSection title="Default Expanded">
        <div>Default Visible</div>
      </FilterSection>
    );

    expect(screen.getByText('Default Visible')).toBeInTheDocument();
  });

  it('should render children correctly', () => {
    render(
      <FilterSection title="Parent">
        <input type="text" placeholder="Child input" />
        <button>Child button</button>
      </FilterSection>
    );

    expect(screen.getByPlaceholderText('Child input')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Child button' })).toBeInTheDocument();
  });
});

