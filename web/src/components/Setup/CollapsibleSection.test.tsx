import { describe, it, expect } from 'vitest';
import { render, screen, fireEvent } from '../../__tests__/test-utils';
import { CollapsibleSection } from './CollapsibleSection';

describe('CollapsibleSection', () => {
  it('renders title and description', () => {
    render(
      <CollapsibleSection
        title="Test Section"
        description="This is a test description"
      >
        <p>Content</p>
      </CollapsibleSection>
    );

    expect(screen.getByText('Test Section')).toBeInTheDocument();
    expect(screen.getByText('This is a test description')).toBeInTheDocument();
  });

  it('shows optional label when isOptional is true', () => {
    render(
      <CollapsibleSection
        title="Optional Section"
        isOptional={true}
      >
        <p>Content</p>
      </CollapsibleSection>
    );

    expect(screen.getByText('Optional')).toBeInTheDocument();
  });

  it('does not show optional label when isOptional is false', () => {
    render(
      <CollapsibleSection
        title="Required Section"
        isOptional={false}
      >
        <p>Content</p>
      </CollapsibleSection>
    );

    expect(screen.queryByText('Optional')).not.toBeInTheDocument();
  });

  it('is collapsed by default', () => {
    render(
      <CollapsibleSection title="Collapsed Section">
        <p>Hidden Content</p>
      </CollapsibleSection>
    );

    expect(screen.queryByText('Hidden Content')).not.toBeInTheDocument();
  });

  it('is expanded by default when defaultExpanded is true', () => {
    render(
      <CollapsibleSection
        title="Expanded Section"
        defaultExpanded={true}
      >
        <p>Visible Content</p>
      </CollapsibleSection>
    );

    expect(screen.getByText('Visible Content')).toBeInTheDocument();
  });

  it('toggles content visibility when clicked', () => {
    render(
      <CollapsibleSection title="Toggle Section">
        <p>Toggleable Content</p>
      </CollapsibleSection>
    );

    // Content should be hidden initially
    expect(screen.queryByText('Toggleable Content')).not.toBeInTheDocument();

    // Click to expand
    fireEvent.click(screen.getByRole('button'));
    expect(screen.getByText('Toggleable Content')).toBeInTheDocument();

    // Click to collapse
    fireEvent.click(screen.getByRole('button'));
    expect(screen.queryByText('Toggleable Content')).not.toBeInTheDocument();
  });

  it('renders children correctly when expanded', () => {
    render(
      <CollapsibleSection title="Section" defaultExpanded={true}>
        <div>
          <input type="text" placeholder="Enter text" />
          <button>Submit</button>
        </div>
      </CollapsibleSection>
    );

    expect(screen.getByPlaceholderText('Enter text')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Submit' })).toBeInTheDocument();
  });

  it('handles missing description', () => {
    render(
      <CollapsibleSection title="No Description">
        <p>Content</p>
      </CollapsibleSection>
    );

    expect(screen.getByText('No Description')).toBeInTheDocument();
  });
});

