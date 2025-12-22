import { describe, it, expect } from 'vitest';
import { getBatchDuration, formatBatchDuration, type Batch } from './batch';

// Helper to create a batch with optional dates
function createBatch(overrides: Partial<Batch> = {}): Batch {
  return {
    id: 1,
    name: 'Test Batch',
    description: 'A test batch',
    type: 'manual',
    repository_count: 10,
    status: 'completed',
    created_at: '2024-01-01T00:00:00Z',
    ...overrides,
  };
}

describe('getBatchDuration', () => {
  it('should return null when started_at is missing', () => {
    const batch = createBatch({ completed_at: '2024-01-01T01:00:00Z' });
    expect(getBatchDuration(batch)).toBeNull();
  });

  it('should return null when completed_at is missing', () => {
    const batch = createBatch({ started_at: '2024-01-01T00:00:00Z' });
    expect(getBatchDuration(batch)).toBeNull();
  });

  it('should return null when both dates are missing', () => {
    const batch = createBatch();
    expect(getBatchDuration(batch)).toBeNull();
  });

  it('should calculate duration in seconds', () => {
    const batch = createBatch({
      started_at: '2024-01-01T00:00:00Z',
      completed_at: '2024-01-01T01:00:00Z',
    });
    expect(getBatchDuration(batch)).toBe(3600); // 1 hour = 3600 seconds
  });

  it('should handle short durations', () => {
    const batch = createBatch({
      started_at: '2024-01-01T00:00:00Z',
      completed_at: '2024-01-01T00:00:30Z',
    });
    expect(getBatchDuration(batch)).toBe(30); // 30 seconds
  });

  it('should handle multi-hour durations', () => {
    const batch = createBatch({
      started_at: '2024-01-01T00:00:00Z',
      completed_at: '2024-01-01T05:30:00Z',
    });
    expect(getBatchDuration(batch)).toBe(19800); // 5.5 hours = 19800 seconds
  });

  it('should handle multi-day durations', () => {
    const batch = createBatch({
      started_at: '2024-01-01T00:00:00Z',
      completed_at: '2024-01-03T12:00:00Z',
    });
    expect(getBatchDuration(batch)).toBe(216000); // 2.5 days = 216000 seconds
  });
});

describe('formatBatchDuration', () => {
  it('should return null when dates are missing', () => {
    const batch = createBatch();
    expect(formatBatchDuration(batch)).toBeNull();
  });

  it('should return null when started_at is missing', () => {
    const batch = createBatch({ completed_at: '2024-01-01T01:00:00Z' });
    expect(formatBatchDuration(batch)).toBeNull();
  });

  it('should return null when completed_at is missing', () => {
    const batch = createBatch({ started_at: '2024-01-01T00:00:00Z' });
    expect(formatBatchDuration(batch)).toBeNull();
  });

  it('should format seconds only', () => {
    const batch = createBatch({
      started_at: '2024-01-01T00:00:00Z',
      completed_at: '2024-01-01T00:00:45Z',
    });
    expect(formatBatchDuration(batch)).toBe('45s');
  });

  it('should format zero seconds', () => {
    const batch = createBatch({
      started_at: '2024-01-01T00:00:00Z',
      completed_at: '2024-01-01T00:00:00Z',
    });
    expect(formatBatchDuration(batch)).toBe('0s');
  });

  it('should format minutes and seconds', () => {
    const batch = createBatch({
      started_at: '2024-01-01T00:00:00Z',
      completed_at: '2024-01-01T00:05:30Z',
    });
    expect(formatBatchDuration(batch)).toBe('5m 30s');
  });

  it('should format exactly one minute', () => {
    const batch = createBatch({
      started_at: '2024-01-01T00:00:00Z',
      completed_at: '2024-01-01T00:01:00Z',
    });
    expect(formatBatchDuration(batch)).toBe('1m 0s');
  });

  it('should format hours, minutes, and seconds', () => {
    const batch = createBatch({
      started_at: '2024-01-01T00:00:00Z',
      completed_at: '2024-01-01T02:15:30Z',
    });
    expect(formatBatchDuration(batch)).toBe('2h 15m 30s');
  });

  it('should format exactly one hour', () => {
    const batch = createBatch({
      started_at: '2024-01-01T00:00:00Z',
      completed_at: '2024-01-01T01:00:00Z',
    });
    expect(formatBatchDuration(batch)).toBe('1h 0m 0s');
  });

  it('should format multi-hour durations', () => {
    const batch = createBatch({
      started_at: '2024-01-01T00:00:00Z',
      completed_at: '2024-01-01T10:30:45Z',
    });
    expect(formatBatchDuration(batch)).toBe('10h 30m 45s');
  });

  it('should handle long durations (days worth of hours)', () => {
    const batch = createBatch({
      started_at: '2024-01-01T00:00:00Z',
      completed_at: '2024-01-03T12:30:15Z',
    });
    // 60 hours, 30 minutes, 15 seconds
    expect(formatBatchDuration(batch)).toBe('60h 30m 15s');
  });
});

