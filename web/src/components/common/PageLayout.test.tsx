import { describe, it, expect } from 'vitest';
import { render, screen } from '../../__tests__/test-utils';
import { PageLayout } from './PageLayout';

describe('PageLayout', () => {
  it('should render children', () => {
    render(
      <PageLayout>
        <div data-testid="child">Child content</div>
      </PageLayout>
    );
    expect(screen.getByTestId('child')).toBeInTheDocument();
    expect(screen.getByText('Child content')).toBeInTheDocument();
  });

  it('should render as main element', () => {
    render(
      <PageLayout>
        <div>Content</div>
      </PageLayout>
    );
    expect(screen.getByRole('main')).toBeInTheDocument();
  });

  it('should have id="main-content" for accessibility', () => {
    render(
      <PageLayout>
        <div>Content</div>
      </PageLayout>
    );
    expect(screen.getByRole('main')).toHaveAttribute('id', 'main-content');
  });

  it('should apply max-width constraint by default', () => {
    render(
      <PageLayout>
        <div>Content</div>
      </PageLayout>
    );
    const main = screen.getByRole('main');
    expect(main.className).toContain('max-w-[1920px]');
    expect(main.className).toContain('mx-auto');
  });

  it('should not apply max-width when fullWidth is true', () => {
    render(
      <PageLayout fullWidth>
        <div>Content</div>
      </PageLayout>
    );
    const main = screen.getByRole('main');
    expect(main.className).not.toContain('max-w-[1920px]');
  });

  it('should apply custom className', () => {
    render(
      <PageLayout className="custom-class">
        <div>Content</div>
      </PageLayout>
    );
    const main = screen.getByRole('main');
    expect(main.className).toContain('custom-class');
  });

  it('should render multiple children', () => {
    render(
      <PageLayout>
        <div data-testid="child-1">First child</div>
        <div data-testid="child-2">Second child</div>
        <div data-testid="child-3">Third child</div>
      </PageLayout>
    );
    expect(screen.getByTestId('child-1')).toBeInTheDocument();
    expect(screen.getByTestId('child-2')).toBeInTheDocument();
    expect(screen.getByTestId('child-3')).toBeInTheDocument();
  });

  it('should have responsive padding classes', () => {
    render(
      <PageLayout>
        <div>Content</div>
      </PageLayout>
    );
    const main = screen.getByRole('main');
    expect(main.className).toContain('px-4');
    expect(main.className).toContain('sm:px-6');
    expect(main.className).toContain('lg:px-8');
    expect(main.className).toContain('py-8');
  });
});

