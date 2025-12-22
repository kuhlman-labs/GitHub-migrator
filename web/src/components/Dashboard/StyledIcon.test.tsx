import { describe, it, expect } from 'vitest';
import { render, screen } from '../../__tests__/test-utils';
import { StyledIcon } from './StyledIcon';
import { RepoIcon } from '@primer/octicons-react';

describe('StyledIcon', () => {
  it('should render icon with color', () => {
    render(<StyledIcon icon={<RepoIcon data-testid="icon" />} color="red" />);
    
    const wrapper = screen.getByTestId('icon').parentElement;
    expect(wrapper).toHaveAttribute('style', expect.stringContaining('color: red'));
  });

  it('should render icon with custom className', () => {
    render(
      <StyledIcon 
        icon={<RepoIcon data-testid="icon" />} 
        color="blue" 
        className="custom-class" 
      />
    );
    
    const wrapper = screen.getByTestId('icon').parentElement;
    expect(wrapper).toHaveClass('custom-class');
  });

  it('should clone and render the icon element', () => {
    render(<StyledIcon icon={<RepoIcon data-testid="test-icon" />} color="green" />);
    
    expect(screen.getByTestId('test-icon')).toBeInTheDocument();
  });

  it('should work with different icon colors', () => {
    const { rerender } = render(
      <StyledIcon icon={<RepoIcon data-testid="icon" />} color="var(--fgColor-accent)" />
    );
    
    let wrapper = screen.getByTestId('icon').parentElement;
    expect(wrapper).toHaveAttribute('style', expect.stringContaining('color'));

    rerender(
      <StyledIcon icon={<RepoIcon data-testid="icon" />} color="var(--fgColor-success)" />
    );
    
    wrapper = screen.getByTestId('icon').parentElement;
    expect(wrapper).toHaveAttribute('style', expect.stringContaining('color'));
  });
});

