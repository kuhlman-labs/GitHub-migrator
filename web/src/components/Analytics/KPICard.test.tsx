import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '../../__tests__/test-utils';
import { KPICard } from './KPICard';
import { RepoIcon } from '@primer/octicons-react';

describe('KPICard', () => {
  describe('basic rendering', () => {
    it('should render title and value', () => {
      render(<KPICard title="Total Repositories" value={100} />);

      expect(screen.getByText('Total Repositories')).toBeInTheDocument();
      expect(screen.getByText('100')).toBeInTheDocument();
    });

    it('should render subtitle when provided', () => {
      render(
        <KPICard 
          title="Total Repositories" 
          value={100} 
          subtitle="50 migrated" 
        />
      );

      expect(screen.getByText('50 migrated')).toBeInTheDocument();
    });

    it('should render string values', () => {
      render(<KPICard title="Progress" value="75%" />);

      expect(screen.getByText('75%')).toBeInTheDocument();
    });

    it('should render icon when provided', () => {
      render(
        <KPICard 
          title="Repos" 
          value={100} 
          icon={<RepoIcon data-testid="repo-icon" />} 
        />
      );

      expect(screen.getByTestId('repo-icon')).toBeInTheDocument();
    });
  });

  describe('tooltip', () => {
    it('should render tooltip when provided', () => {
      render(
        <KPICard 
          title="Total" 
          value={100} 
          tooltip="This is a helpful tooltip" 
        />
      );

      expect(screen.getByText('This is a helpful tooltip')).toBeInTheDocument();
    });

    it('should not render tooltip when not provided', () => {
      render(<KPICard title="Total" value={100} />);

      expect(screen.queryByText('This is a helpful tooltip')).not.toBeInTheDocument();
    });
  });

  describe('colors', () => {
    it('should apply blue color by default', () => {
      render(<KPICard title="Test" value={100} />);

      const card = screen.getByText('100').closest('div');
      expect(card).toHaveStyle({ color: 'var(--fgColor-accent)' });
    });

    it('should apply green color', () => {
      render(<KPICard title="Test" value={100} color="green" />);

      const valueElement = screen.getByText('100');
      expect(valueElement).toHaveStyle({ color: 'var(--fgColor-success)' });
    });

    it('should apply purple color', () => {
      render(<KPICard title="Test" value={100} color="purple" />);

      const valueElement = screen.getByText('100');
      expect(valueElement).toHaveStyle({ color: 'var(--fgColor-done)' });
    });

    it('should apply yellow color', () => {
      render(<KPICard title="Test" value={100} color="yellow" />);

      const valueElement = screen.getByText('100');
      expect(valueElement).toHaveStyle({ color: 'var(--fgColor-attention)' });
    });
  });

  describe('interactivity', () => {
    it('should call onClick when clicked', () => {
      const onClick = vi.fn();
      render(<KPICard title="Test" value={100} onClick={onClick} />);

      const card = screen.getByRole('button');
      fireEvent.click(card);

      expect(onClick).toHaveBeenCalledTimes(1);
    });

    it('should not render as button when onClick not provided', () => {
      render(<KPICard title="Test" value={100} />);

      expect(screen.queryByRole('button')).not.toBeInTheDocument();
    });

    it('should be keyboard accessible when clickable', () => {
      const onClick = vi.fn();
      render(<KPICard title="Test" value={100} onClick={onClick} />);

      const card = screen.getByRole('button');
      
      fireEvent.keyDown(card, { key: 'Enter' });
      expect(onClick).toHaveBeenCalledTimes(1);
    });

    it('should respond to Space key when clickable', () => {
      const onClick = vi.fn();
      render(<KPICard title="Test" value={100} onClick={onClick} />);

      const card = screen.getByRole('button');
      
      fireEvent.keyDown(card, { key: ' ' });
      expect(onClick).toHaveBeenCalledTimes(1);
    });

    it('should not respond to other keys', () => {
      const onClick = vi.fn();
      render(<KPICard title="Test" value={100} onClick={onClick} />);

      const card = screen.getByRole('button');
      
      fireEvent.keyDown(card, { key: 'Tab' });
      expect(onClick).not.toHaveBeenCalled();
    });
  });

  describe('accessibility', () => {
    it('should have role="button" when clickable', () => {
      render(<KPICard title="Test" value={100} onClick={() => {}} />);

      expect(screen.getByRole('button')).toBeInTheDocument();
    });

    it('should have tabIndex=0 when clickable', () => {
      render(<KPICard title="Test" value={100} onClick={() => {}} />);

      const card = screen.getByRole('button');
      expect(card).toHaveAttribute('tabindex', '0');
    });

    it('should have aria-label when clickable', () => {
      render(<KPICard title="Total Repos" value={100} onClick={() => {}} />);

      const card = screen.getByRole('button');
      expect(card).toHaveAttribute('aria-label', 'View repositories: Total Repos');
    });

    it('should not have role, tabIndex, or aria-label when not clickable', () => {
      const { container } = render(<KPICard title="Test" value={100} />);

      const card = container.firstChild as HTMLElement;
      expect(card).not.toHaveAttribute('role');
      expect(card).not.toHaveAttribute('tabindex');
      expect(card).not.toHaveAttribute('aria-label');
    });
  });
});

