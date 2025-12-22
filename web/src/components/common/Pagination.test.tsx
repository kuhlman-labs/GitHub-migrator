import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '../../__tests__/test-utils';
import { Pagination } from './Pagination';

describe('Pagination', () => {
  const defaultProps = {
    currentPage: 1,
    totalItems: 100,
    pageSize: 10,
    onPageChange: vi.fn(),
  };

  it('should render pagination info text', () => {
    render(<Pagination {...defaultProps} />);
    // Use regex to find text that might be split across elements
    expect(screen.getByText(/Showing/)).toBeInTheDocument();
    expect(screen.getByText(/results/)).toBeInTheDocument();
  });

  it('should not render when totalPages is 1 or less', () => {
    const { container } = render(
      <Pagination {...defaultProps} totalItems={10} pageSize={10} />
    );
    // When there's only 1 page, Pagination returns null
    // But ThemeProvider wraps it, so check for the inner content
    expect(container.querySelector('nav')).toBeNull();
  });

  it('should not render when totalItems is 0', () => {
    const { container } = render(
      <Pagination {...defaultProps} totalItems={0} />
    );
    expect(container.querySelector('nav')).toBeNull();
  });

  it('should calculate correct start and end items for first page', () => {
    render(<Pagination {...defaultProps} currentPage={1} />);
    // Check that the text contains the expected values
    const text = screen.getByText(/Showing/).closest('p')?.textContent;
    expect(text).toContain('1');
    expect(text).toContain('10');
  });

  it('should calculate correct start and end items for middle page', () => {
    render(<Pagination {...defaultProps} currentPage={5} />);
    expect(screen.getByText('41')).toBeInTheDocument();
    expect(screen.getByText('50')).toBeInTheDocument();
  });

  it('should calculate correct end item for last page', () => {
    render(
      <Pagination {...defaultProps} currentPage={10} totalItems={95} pageSize={10} />
    );
    const text = screen.getByText(/Showing/).closest('p')?.textContent;
    expect(text).toContain('91');
    expect(text).toContain('95');
  });

  it('should render with different page sizes', () => {
    render(<Pagination {...defaultProps} pageSize={25} totalItems={100} />);
    expect(screen.getByText('25')).toBeInTheDocument();
  });

  it('should handle partial last page', () => {
    render(
      <Pagination {...defaultProps} currentPage={4} totalItems={35} pageSize={10} />
    );
    const text = screen.getByText(/Showing/).closest('p')?.textContent;
    expect(text).toContain('31');
    expect(text).toContain('35');
  });

  it('should render page buttons', () => {
    render(<Pagination {...defaultProps} />);
    // Primer Pagination renders navigation buttons
    expect(screen.getByRole('navigation')).toBeInTheDocument();
  });
});

