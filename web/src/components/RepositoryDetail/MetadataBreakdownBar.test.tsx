import { describe, it, expect } from 'vitest';
import { render, screen } from '../../__tests__/test-utils';
import { MetadataBreakdownBar } from './MetadataBreakdownBar';

describe('MetadataBreakdownBar', () => {
  const defaultProps = {
    releases: 1024 * 1024, // 1 MB
    issues: 512 * 1024, // 512 KB
    prs: 256 * 1024, // 256 KB
    attachments: 256 * 1024, // 256 KB
    total: 2 * 1024 * 1024, // 2 MB
    limit: 10 * 1024 * 1024, // 10 MB
  };

  it('renders the breakdown bar', () => {
    const { container } = render(<MetadataBreakdownBar {...defaultProps} />);

    // Check for the visual bar container
    const bar = container.querySelector('.h-6.w-full.bg-gray-200');
    expect(bar).toBeInTheDocument();
  });

  it('displays releases in the legend', () => {
    render(<MetadataBreakdownBar {...defaultProps} />);

    expect(screen.getByText(/Releases:/)).toBeInTheDocument();
    expect(screen.getByText(/1 MB/)).toBeInTheDocument();
  });

  it('displays issues in the legend', () => {
    render(<MetadataBreakdownBar {...defaultProps} />);

    expect(screen.getByText(/Issues:/)).toBeInTheDocument();
    expect(screen.getByText(/512 KB/)).toBeInTheDocument();
  });

  it('displays PRs in the legend', () => {
    render(<MetadataBreakdownBar {...defaultProps} />);

    expect(screen.getByText(/PRs:/)).toBeInTheDocument();
  });

  it('displays attachments in the legend', () => {
    render(<MetadataBreakdownBar {...defaultProps} />);

    expect(screen.getByText(/Attachments:/)).toBeInTheDocument();
  });

  it('displays total metadata size', () => {
    render(<MetadataBreakdownBar {...defaultProps} />);

    expect(screen.getByText('Total Metadata Size (Estimated):')).toBeInTheDocument();
    expect(screen.getByText(/2 MB \/ 10 MB/)).toBeInTheDocument();
  });

  it('displays percentage of limit', () => {
    render(<MetadataBreakdownBar {...defaultProps} />);

    // 2 MB / 10 MB = 20%
    expect(screen.getByText(/20\.0% of limit/)).toBeInTheDocument();
  });

  it('handles zero total gracefully', () => {
    render(
      <MetadataBreakdownBar
        {...defaultProps}
        releases={0}
        issues={0}
        prs={0}
        attachments={0}
        total={0}
      />
    );

    // Should show 0% for all categories
    expect(screen.getByText(/Releases:.*0%/)).toBeInTheDocument();
  });

  it('handles zero limit gracefully', () => {
    render(
      <MetadataBreakdownBar
        {...defaultProps}
        limit={0}
      />
    );

    expect(screen.getByText(/0% of limit/)).toBeInTheDocument();
  });

  it('renders colored segments for each category', () => {
    const { container } = render(<MetadataBreakdownBar {...defaultProps} />);

    // Check for colored segments in the bar
    expect(container.querySelector('.bg-purple-500')).toBeInTheDocument();
    expect(container.querySelector('.bg-blue-500')).toBeInTheDocument();
    expect(container.querySelector('.bg-green-500')).toBeInTheDocument();
    expect(container.querySelector('.bg-orange-500')).toBeInTheDocument();
  });

  it('hides segments with zero value', () => {
    const { container } = render(
      <MetadataBreakdownBar
        releases={0}
        issues={defaultProps.issues}
        prs={0}
        attachments={0}
        total={defaultProps.issues}
        limit={defaultProps.limit}
      />
    );

    // Only issues should be visible in the bar
    expect(container.querySelector('.h-6 .bg-blue-500')).toBeInTheDocument();
    expect(container.querySelector('.h-6 .bg-purple-500')).not.toBeInTheDocument();
    expect(container.querySelector('.h-6 .bg-green-500')).not.toBeInTheDocument();
    expect(container.querySelector('.h-6 .bg-orange-500')).not.toBeInTheDocument();
  });

  it('shows percentages in bar when segment is large enough', () => {
    const { container } = render(
      <MetadataBreakdownBar
        releases={1000}
        issues={0}
        prs={0}
        attachments={0}
        total={1000}
        limit={10000}
      />
    );

    // When releases is 100% of total and > 15%, it should show percentage
    const releasesSegment = container.querySelector('.h-6 .bg-purple-500');
    expect(releasesSegment).toBeInTheDocument();
    expect(releasesSegment?.textContent).toBe('100%');
  });
});

