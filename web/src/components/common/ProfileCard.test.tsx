import { describe, it, expect } from 'vitest';
import { render, screen } from '../../__tests__/test-utils';
import { ProfileCard } from './ProfileCard';

describe('ProfileCard', () => {
  it('should render title', () => {
    render(
      <ProfileCard title="Test Title">
        <div>Content</div>
      </ProfileCard>
    );

    expect(screen.getByText('Test Title')).toBeInTheDocument();
  });

  it('should render children', () => {
    render(
      <ProfileCard title="Test">
        <p>Child paragraph</p>
        <span>Child span</span>
      </ProfileCard>
    );

    expect(screen.getByText('Child paragraph')).toBeInTheDocument();
    expect(screen.getByText('Child span')).toBeInTheDocument();
  });

  it('should have proper heading level', () => {
    render(
      <ProfileCard title="Section Title">
        <div>Content</div>
      </ProfileCard>
    );

    const heading = screen.getByRole('heading', { level: 3 });
    expect(heading).toHaveTextContent('Section Title');
  });

  it('should render with proper styling classes', () => {
    render(
      <ProfileCard title="Styled Card">
        <div>Content</div>
      </ProfileCard>
    );

    // Find the card by its heading
    const heading = screen.getByRole('heading', { level: 3 });
    const card = heading.closest('.rounded-lg');
    expect(card).toBeInTheDocument();
    expect(card).toHaveClass('rounded-lg', 'shadow-sm', 'p-6');
  });
});

