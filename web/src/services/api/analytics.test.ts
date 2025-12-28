import { describe, it, expect, vi, beforeEach } from 'vitest';
import { analyticsApi } from './analytics';
import { client } from './client';

// Mock the axios client
vi.mock('./client', () => ({
  client: {
    get: vi.fn(),
  },
}));

describe('analyticsApi', () => {
  const mockClient = client as unknown as {
    get: ReturnType<typeof vi.fn>;
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('getSummary', () => {
    it('should fetch analytics summary without filters', async () => {
      const mockData = { total_repositories: 100, migrated_count: 50 };
      mockClient.get.mockResolvedValue({ data: mockData });

      const result = await analyticsApi.getSummary();

      expect(mockClient.get).toHaveBeenCalledWith('/analytics/summary', { params: undefined });
      expect(result).toEqual(mockData);
    });

    it('should fetch analytics summary with filters', async () => {
      const mockData = { total_repositories: 25, migrated_count: 10 };
      mockClient.get.mockResolvedValue({ data: mockData });

      const result = await analyticsApi.getSummary({ organization: 'my-org', batch_id: '5' });

      expect(mockClient.get).toHaveBeenCalledWith('/analytics/summary', {
        params: { organization: 'my-org', batch_id: '5' },
      });
      expect(result).toEqual(mockData);
    });
  });

  describe('getProgress', () => {
    it('should fetch migration progress', async () => {
      const mockData = { completion_percentage: 75 };
      mockClient.get.mockResolvedValue({ data: mockData });

      const result = await analyticsApi.getProgress();

      expect(mockClient.get).toHaveBeenCalledWith('/analytics/progress');
      expect(result).toEqual(mockData);
    });
  });

  describe('getExecutiveReport', () => {
    it('should fetch executive report without filters', async () => {
      const mockData = { source_type: 'github', executive_summary: {} };
      mockClient.get.mockResolvedValue({ data: mockData });

      const result = await analyticsApi.getExecutiveReport();

      expect(mockClient.get).toHaveBeenCalledWith('/analytics/executive-report', { params: undefined });
      expect(result).toEqual(mockData);
    });

    it('should fetch executive report with filters', async () => {
      const mockData = { source_type: 'github', executive_summary: {} };
      mockClient.get.mockResolvedValue({ data: mockData });

      const result = await analyticsApi.getExecutiveReport({ organization: 'my-org' });

      expect(mockClient.get).toHaveBeenCalledWith('/analytics/executive-report', {
        params: { organization: 'my-org' },
      });
      expect(result).toEqual(mockData);
    });
  });

  describe('exportExecutiveReport', () => {
    it('should export executive report as CSV', async () => {
      const mockBlob = new Blob(['csv,data']);
      mockClient.get.mockResolvedValue({ data: mockBlob });

      const result = await analyticsApi.exportExecutiveReport('csv');

      expect(mockClient.get).toHaveBeenCalledWith('/analytics/executive-report/export', {
        params: { format: 'csv' },
        responseType: 'blob',
      });
      expect(result).toEqual(mockBlob);
    });

    it('should export executive report as JSON with filters', async () => {
      const mockBlob = new Blob(['{}']);
      mockClient.get.mockResolvedValue({ data: mockBlob });

      const result = await analyticsApi.exportExecutiveReport('json', { organization: 'my-org' });

      expect(mockClient.get).toHaveBeenCalledWith('/analytics/executive-report/export', {
        params: { format: 'json', organization: 'my-org' },
        responseType: 'blob',
      });
      expect(result).toEqual(mockBlob);
    });
  });

  describe('exportDetailedDiscoveryReport', () => {
    it('should export detailed discovery report as CSV', async () => {
      const mockBlob = new Blob(['csv,data']);
      mockClient.get.mockResolvedValue({ data: mockBlob });

      const result = await analyticsApi.exportDetailedDiscoveryReport('csv');

      expect(mockClient.get).toHaveBeenCalledWith('/analytics/detailed-discovery-report/export', {
        params: { format: 'csv' },
        responseType: 'blob',
      });
      expect(result).toEqual(mockBlob);
    });

    it('should export with project and batch filters', async () => {
      const mockBlob = new Blob(['{}']);
      mockClient.get.mockResolvedValue({ data: mockBlob });

      const result = await analyticsApi.exportDetailedDiscoveryReport('json', {
        organization: 'my-org',
        project: 'my-project',
        batch_id: '3',
      });

      expect(mockClient.get).toHaveBeenCalledWith('/analytics/detailed-discovery-report/export', {
        params: { format: 'json', organization: 'my-org', project: 'my-project', batch_id: '3' },
        responseType: 'blob',
      });
      expect(result).toEqual(mockBlob);
    });
  });

  describe('getDashboardActionItems', () => {
    it('should fetch dashboard action items', async () => {
      const mockData = {
        failed_migrations: [],
        pending_approvals: [],
        stale_batches: [],
      };
      mockClient.get.mockResolvedValue({ data: mockData });

      const result = await analyticsApi.getDashboardActionItems();

      expect(mockClient.get).toHaveBeenCalledWith('/dashboard/action-items');
      expect(result).toEqual(mockData);
    });
  });
});

