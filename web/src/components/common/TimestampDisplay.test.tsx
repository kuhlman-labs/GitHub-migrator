import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen } from '../../__tests__/test-utils';
import { TimestampDisplay } from './TimestampDisplay';

describe('TimestampDisplay', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2024-01-15T12:00:00Z'));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  describe('when timestamp is null or undefined', () => {
    it('should not render content for null timestamp', () => {
      render(<TimestampDisplay timestamp={null} />);
      expect(screen.queryByText('Updated:')).not.toBeInTheDocument();
    });

    it('should not render content for undefined timestamp', () => {
      render(<TimestampDisplay timestamp={undefined} />);
      expect(screen.queryByText('Updated:')).not.toBeInTheDocument();
    });
  });

  describe('rendering', () => {
    it('should render label and formatted time', () => {
      render(<TimestampDisplay timestamp="2024-01-15T10:00:00Z" label="Updated" />);
      
      expect(screen.getByText('Updated:')).toBeInTheDocument();
    });

    it('should use default label "Updated" when not provided', () => {
      render(<TimestampDisplay timestamp="2024-01-15T10:00:00Z" />);
      
      expect(screen.getByText('Updated:')).toBeInTheDocument();
    });

    it('should not show label when showLabel is false', () => {
      render(
        <TimestampDisplay 
          timestamp="2024-01-15T10:00:00Z" 
          label="Updated"
          showLabel={false} 
        />
      );
      
      expect(screen.queryByText('Updated:')).not.toBeInTheDocument();
    });

    it('should render custom label', () => {
      render(<TimestampDisplay timestamp="2024-01-15T10:00:00Z" label="Discovered" />);
      
      expect(screen.getByText('Discovered:')).toBeInTheDocument();
    });
  });

  describe('size variants', () => {
    it('should render with sm size', () => {
      render(<TimestampDisplay timestamp="2024-01-15T10:00:00Z" size="sm" label="Test" />);
      
      // Just verify it renders
      expect(screen.getByText('Test:')).toBeInTheDocument();
    });

    it('should render with md size', () => {
      render(<TimestampDisplay timestamp="2024-01-15T10:00:00Z" size="md" label="Test" />);
      
      // Just verify it renders
      expect(screen.getByText('Test:')).toBeInTheDocument();
    });
  });
});

