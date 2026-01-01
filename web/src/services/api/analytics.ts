/**
 * Analytics and reporting API endpoints.
 */
import { client } from './client';
import type { Analytics, ExecutiveReport, DashboardActionItems } from '../../types';

export const analyticsApi = {
  async getSummary(filters?: { organization?: string; project?: string; batch_id?: string; source_id?: number }): Promise<Analytics> {
    const { data } = await client.get('/analytics/summary', { params: filters });
    return data;
  },

  async getProgress() {
    const { data } = await client.get('/analytics/progress');
    return data;
  },

  async getExecutiveReport(filters?: {
    organization?: string;
    batch_id?: string;
  }): Promise<ExecutiveReport> {
    const { data } = await client.get('/analytics/executive-report', { params: filters });
    return data;
  },

  async exportExecutiveReport(
    format: 'csv' | 'json',
    filters?: { organization?: string; batch_id?: string }
  ): Promise<Blob> {
    const { data } = await client.get('/analytics/executive-report/export', {
      params: { format, ...filters },
      responseType: 'blob',
    });
    return data;
  },

  async exportDetailedDiscoveryReport(
    format: 'csv' | 'json',
    filters?: { organization?: string; project?: string; batch_id?: string }
  ): Promise<Blob> {
    const { data } = await client.get('/analytics/detailed-discovery-report/export', {
      params: { format, ...filters },
      responseType: 'blob',
    });
    return data;
  },

  async getDashboardActionItems(): Promise<DashboardActionItems> {
    const { data } = await client.get('/dashboard/action-items');
    return data;
  },
};

