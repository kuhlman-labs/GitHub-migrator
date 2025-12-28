import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import {
  formatBytes,
  formatDuration,
  formatDate,
  formatRelativeTime,
  isStaleTimestamp,
  formatTimestampWithStaleness,
  formatDateForInput,
} from './format';

describe('formatBytes', () => {
  it('should return "0 Bytes" for 0', () => {
    expect(formatBytes(0)).toBe('0 Bytes');
  });

  it('should format bytes correctly', () => {
    expect(formatBytes(500)).toBe('500 Bytes');
  });

  it('should format kilobytes correctly', () => {
    expect(formatBytes(1024)).toBe('1 KB');
    expect(formatBytes(1536)).toBe('1.5 KB');
  });

  it('should format megabytes correctly', () => {
    expect(formatBytes(1048576)).toBe('1 MB');
    expect(formatBytes(1572864)).toBe('1.5 MB');
  });

  it('should format gigabytes correctly', () => {
    expect(formatBytes(1073741824)).toBe('1 GB');
  });

  it('should format terabytes correctly', () => {
    expect(formatBytes(1099511627776)).toBe('1 TB');
  });
});

describe('formatDuration', () => {
  it('should format seconds only', () => {
    expect(formatDuration(0)).toBe('0s');
    expect(formatDuration(30)).toBe('30s');
    expect(formatDuration(59)).toBe('59s');
  });

  it('should round to nearest second', () => {
    expect(formatDuration(30.4)).toBe('30s');
    expect(formatDuration(30.6)).toBe('31s');
  });

  it('should format minutes and seconds', () => {
    expect(formatDuration(60)).toBe('1m 0s');
    expect(formatDuration(90)).toBe('1m 30s');
    expect(formatDuration(3599)).toBe('59m 59s');
  });

  it('should format hours and minutes', () => {
    expect(formatDuration(3600)).toBe('1h 0m');
    expect(formatDuration(5400)).toBe('1h 30m');
    expect(formatDuration(7200)).toBe('2h 0m');
  });
});

describe('formatDate', () => {
  it('should format a date string', () => {
    const dateString = '2024-01-15T10:30:00Z';
    const result = formatDate(dateString);
    // Result depends on locale, but should be a non-empty string
    expect(result).toBeTruthy();
    expect(typeof result).toBe('string');
  });
});

describe('formatRelativeTime', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2024-06-15T12:00:00Z'));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('should return "just now" for very recent times', () => {
    const now = new Date();
    expect(formatRelativeTime(now.toISOString())).toBe('just now');
  });

  it('should format minutes ago', () => {
    const fiveMinutesAgo = new Date(Date.now() - 5 * 60 * 1000);
    expect(formatRelativeTime(fiveMinutesAgo.toISOString())).toBe('5m ago');
  });

  it('should format hours ago', () => {
    const threeHoursAgo = new Date(Date.now() - 3 * 60 * 60 * 1000);
    expect(formatRelativeTime(threeHoursAgo.toISOString())).toBe('3h ago');
  });

  it('should format days ago', () => {
    const fiveDaysAgo = new Date(Date.now() - 5 * 24 * 60 * 60 * 1000);
    expect(formatRelativeTime(fiveDaysAgo.toISOString())).toBe('5d ago');
  });

  it('should format months ago', () => {
    const twoMonthsAgo = new Date(Date.now() - 60 * 24 * 60 * 60 * 1000);
    expect(formatRelativeTime(twoMonthsAgo.toISOString())).toBe('2mo ago');
  });

  it('should format years ago', () => {
    const twoYearsAgo = new Date(Date.now() - 730 * 24 * 60 * 60 * 1000);
    expect(formatRelativeTime(twoYearsAgo.toISOString())).toBe('2y ago');
  });
});

describe('isStaleTimestamp', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2024-06-15T12:00:00Z'));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('should return false for recent timestamps', () => {
    const recentDate = new Date(Date.now() - 10 * 24 * 60 * 60 * 1000); // 10 days ago
    expect(isStaleTimestamp(recentDate.toISOString())).toBe(false);
  });

  it('should return true for old timestamps with default threshold', () => {
    const oldDate = new Date(Date.now() - 45 * 24 * 60 * 60 * 1000); // 45 days ago
    expect(isStaleTimestamp(oldDate.toISOString())).toBe(true);
  });

  it('should respect custom threshold', () => {
    const date = new Date(Date.now() - 10 * 24 * 60 * 60 * 1000); // 10 days ago
    expect(isStaleTimestamp(date.toISOString(), 5)).toBe(true);
    expect(isStaleTimestamp(date.toISOString(), 15)).toBe(false);
  });
});

describe('formatTimestampWithStaleness', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2024-06-15T12:00:00Z'));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('should return formatted timestamp with staleness info', () => {
    const recentDate = new Date(Date.now() - 5 * 24 * 60 * 60 * 1000); // 5 days ago
    const result = formatTimestampWithStaleness(recentDate.toISOString());

    expect(result).toHaveProperty('formatted');
    expect(result).toHaveProperty('isStale');
    expect(result).toHaveProperty('fullDate');
    expect(result.formatted).toBe('5d ago');
    expect(result.isStale).toBe(false);
  });

  it('should mark old timestamps as stale', () => {
    const oldDate = new Date(Date.now() - 45 * 24 * 60 * 60 * 1000); // 45 days ago
    const result = formatTimestampWithStaleness(oldDate.toISOString());

    expect(result.isStale).toBe(true);
  });
});

describe('formatDateForInput', () => {
  it('should return empty string for null', () => {
    expect(formatDateForInput(null)).toBe('');
  });

  it('should return empty string for undefined', () => {
    expect(formatDateForInput(undefined)).toBe('');
  });

  it('should return empty string for empty string', () => {
    expect(formatDateForInput('')).toBe('');
  });

  it('should format date string for datetime-local input', () => {
    // Use a fixed date to test
    const result = formatDateForInput('2024-06-15T14:30:00Z');
    // The format should be YYYY-MM-DDTHH:MM
    expect(result).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/);
  });
});

