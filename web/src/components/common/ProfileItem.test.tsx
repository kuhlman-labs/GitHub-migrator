import { describe, it, expect } from 'vitest';
import { render, screen } from '../../__tests__/test-utils';
import { ProfileItem } from './ProfileItem';

describe('ProfileItem', () => {
  it('should render label and value', () => {
    render(<ProfileItem label="Name" value="Test User" />);

    expect(screen.getByText('Name:')).toBeInTheDocument();
    expect(screen.getByText('Test User')).toBeInTheDocument();
  });

  it('should render with ReactNode value', () => {
    render(
      <ProfileItem 
        label="Status" 
        value={<span data-testid="status-badge">Active</span>} 
      />
    );

    expect(screen.getByText('Status:')).toBeInTheDocument();
    expect(screen.getByTestId('status-badge')).toBeInTheDocument();
    expect(screen.getByText('Active')).toBeInTheDocument();
  });

  it('should have proper flex layout', () => {
    render(<ProfileItem label="Email" value="test@example.com" />);

    const labelElement = screen.getByText('Email:');
    const container = labelElement.closest('.flex');
    
    expect(container).toHaveClass('flex', 'justify-between', 'items-center');
  });
});

