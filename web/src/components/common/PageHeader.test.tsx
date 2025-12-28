import { describe, it, expect } from 'vitest';
import { render, screen } from '../../__tests__/test-utils';
import { PageHeader } from './PageHeader';

describe('PageHeader', () => {
  it('should render title', () => {
    render(<PageHeader title="Test Title" />);
    expect(screen.getByRole('heading', { name: 'Test Title' })).toBeInTheDocument();
  });

  it('should render title as h1', () => {
    render(<PageHeader title="Test Title" />);
    const heading = screen.getByRole('heading', { level: 1 });
    expect(heading).toHaveTextContent('Test Title');
  });

  it('should render description when provided', () => {
    render(<PageHeader title="Title" description="This is a description" />);
    expect(screen.getByText('This is a description')).toBeInTheDocument();
  });

  it('should not render description when not provided', () => {
    render(<PageHeader title="Title" />);
    expect(screen.queryByText('This is a description')).not.toBeInTheDocument();
  });

  it('should render actions when provided', () => {
    render(
      <PageHeader
        title="Title"
        actions={<button>Action Button</button>}
      />
    );
    expect(screen.getByRole('button', { name: 'Action Button' })).toBeInTheDocument();
  });

  it('should not render actions container when not provided', () => {
    const { container } = render(<PageHeader title="Title" />);
    // The actions div should not be present when no actions
    expect(container.querySelector('.flex.items-center.gap-4')).not.toBeInTheDocument();
  });

  it('should render multiple action buttons', () => {
    render(
      <PageHeader
        title="Title"
        actions={
          <>
            <button>Button 1</button>
            <button>Button 2</button>
          </>
        }
      />
    );
    expect(screen.getByRole('button', { name: 'Button 1' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Button 2' })).toBeInTheDocument();
  });

  it('should render with all props', () => {
    render(
      <PageHeader
        title="Dashboard"
        description="Overview of migration progress"
        actions={<button>Start Discovery</button>}
      />
    );

    expect(screen.getByRole('heading', { name: 'Dashboard' })).toBeInTheDocument();
    expect(screen.getByText('Overview of migration progress')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Start Discovery' })).toBeInTheDocument();
  });
});

