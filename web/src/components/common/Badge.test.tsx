import { describe, it, expect } from 'vitest';
import { render, screen } from '../../__tests__/test-utils';
import { Badge } from './Badge';

describe('Badge', () => {
  it('should render children', () => {
    render(<Badge>Test Badge</Badge>);
    expect(screen.getByText('Test Badge')).toBeInTheDocument();
  });

  it('should render with default color (gray)', () => {
    render(<Badge>Default Badge</Badge>);
    const badge = screen.getByText('Default Badge');
    expect(badge).toBeInTheDocument();
  });

  it('should render with blue color', () => {
    render(<Badge color="blue">Blue Badge</Badge>);
    expect(screen.getByText('Blue Badge')).toBeInTheDocument();
  });

  it('should render with green color', () => {
    render(<Badge color="green">Green Badge</Badge>);
    expect(screen.getByText('Green Badge')).toBeInTheDocument();
  });

  it('should render with yellow color', () => {
    render(<Badge color="yellow">Yellow Badge</Badge>);
    expect(screen.getByText('Yellow Badge')).toBeInTheDocument();
  });

  it('should render with red color', () => {
    render(<Badge color="red">Red Badge</Badge>);
    expect(screen.getByText('Red Badge')).toBeInTheDocument();
  });

  it('should render with purple color', () => {
    render(<Badge color="purple">Purple Badge</Badge>);
    expect(screen.getByText('Purple Badge')).toBeInTheDocument();
  });

  it('should render with orange color', () => {
    render(<Badge color="orange">Orange Badge</Badge>);
    expect(screen.getByText('Orange Badge')).toBeInTheDocument();
  });

  it('should render with pink color', () => {
    render(<Badge color="pink">Pink Badge</Badge>);
    expect(screen.getByText('Pink Badge')).toBeInTheDocument();
  });

  it('should render with indigo color', () => {
    render(<Badge color="indigo">Indigo Badge</Badge>);
    expect(screen.getByText('Indigo Badge')).toBeInTheDocument();
  });

  it('should render with teal color', () => {
    render(<Badge color="teal">Teal Badge</Badge>);
    expect(screen.getByText('Teal Badge')).toBeInTheDocument();
  });

  it('should render complex children', () => {
    render(
      <Badge color="green">
        <span data-testid="icon">✓</span>
        <span>Success</span>
      </Badge>
    );
    expect(screen.getByTestId('icon')).toBeInTheDocument();
    expect(screen.getByText('Success')).toBeInTheDocument();
  });
});

