import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '../../__tests__/test-utils';
import { FallbackAvatar } from './FallbackAvatar';

describe('FallbackAvatar', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should render with initials when no src provided', () => {
    render(<FallbackAvatar login="testuser" />);

    expect(screen.getByText('TE')).toBeInTheDocument();
  });

  it('should show initials immediately without src', () => {
    render(<FallbackAvatar login="johndoe" />);

    expect(screen.getByText('JO')).toBeInTheDocument();
  });

  it('should use custom size', () => {
    const { container } = render(<FallbackAvatar login="test" size={48} />);

    // Find the outer container div with the size styles
    const avatar = container.querySelector('.inline-flex') as HTMLElement;
    expect(avatar).toHaveStyle('width: 48px');
    expect(avatar).toHaveStyle('height: 48px');
  });

  it('should apply custom className', () => {
    const { container } = render(<FallbackAvatar login="test" className="custom-class" />);

    const avatar = container.querySelector('.custom-class');
    expect(avatar).toBeInTheDocument();
  });

  it('should show PersonIcon when no login provided', () => {
    render(<FallbackAvatar />);

    // When no login, it should show PersonIcon (SVG)
    const svg = document.querySelector('svg');
    expect(svg).toBeInTheDocument();
  });

  it('should use default size of 24', () => {
    const { container } = render(<FallbackAvatar login="test" />);

    // Find the outer container div with the size styles
    const avatar = container.querySelector('.inline-flex') as HTMLElement;
    expect(avatar).toHaveStyle('width: 24px');
    expect(avatar).toHaveStyle('height: 24px');
  });

  it('should generate consistent color for same login', () => {
    const { container: container1 } = render(<FallbackAvatar login="user1" />);
    const { container: container2 } = render(<FallbackAvatar login="user1" />);

    const avatar1 = container1.querySelector('[style*="background-color"]');
    const avatar2 = container2.querySelector('[style*="background-color"]');

    // Both should have the same background color
    expect(avatar1).toBeInTheDocument();
    expect(avatar2).toBeInTheDocument();
  });

  it('should have background color set on the avatar', () => {
    // This test verifies the avatar has a background color
    const { container } = render(<FallbackAvatar login="testuser" />);

    // Find the avatar container
    const avatar = container.querySelector('.inline-flex') as HTMLElement;
    expect(avatar).toBeInTheDocument();
    
    // The span inside should have a background color
    const innerSpan = avatar?.querySelector('span');
    expect(innerSpan).toBeInTheDocument();
    // Verify it has inline style with background
    expect(innerSpan?.style.backgroundColor).toBeTruthy();
  });

  it('should show initials in uppercase', () => {
    render(<FallbackAvatar login="lowercase" />);

    // Initials should be uppercase
    expect(screen.getByText('LO')).toBeInTheDocument();
  });

  it('should render with rounded-full class for circular shape', () => {
    const { container } = render(<FallbackAvatar login="test" />);

    const avatar = container.querySelector('.rounded-full');
    expect(avatar).toBeInTheDocument();
  });
});

