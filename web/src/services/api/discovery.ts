/**
 * Discovery-related API endpoints for GitHub and Azure DevOps.
 */
import { client } from './client';
import type { DiscoveryProgress, Organization, Project, GitHubTeam } from '../../types';

export const discoveryApi = {
  // GitHub Discovery
  async start(params: { organization?: string; enterprise_slug?: string; workers?: number; source_id?: number }) {
    const { data } = await client.post('/discovery/start', params);
    return data;
  },

  async getStatus() {
    const { data } = await client.get('/discovery/status');
    return data;
  },

  async getProgress(): Promise<DiscoveryProgress | null> {
    const { data } = await client.get('/discovery/progress');
    if (data.status === 'none') {
      return null;
    }
    return data;
  },

  async cancel() {
    const { data } = await client.post('/discovery/cancel');
    return data;
  },

  // Azure DevOps Discovery
  async startADO(params: { organization?: string; project?: string; workers?: number; source_id?: number }) {
    const { data } = await client.post('/ado/discover', params);
    return data;
  },

  async getADOStatus(organization?: string) {
    const { data } = await client.get('/ado/discovery/status', { params: { organization } });
    return data;
  },

  async listADOProjects(organization?: string) {
    const { data } = await client.get('/ado/projects', { params: { organization } });
    return data.projects || [];
  },

  async getADOProject(organization: string, project: string) {
    const { data } = await client.get(
      `/ado/projects/${encodeURIComponent(organization)}/${encodeURIComponent(project)}`
    );
    return data;
  },

  // Organizations
  async listOrganizations(sourceId?: number): Promise<Organization[]> {
    const params: Record<string, string> = {};
    if (sourceId !== undefined) {
      params.source_id = String(sourceId);
    }
    const { data } = await client.get('/organizations', { params });
    return data;
  },

  async listProjects(sourceId?: number): Promise<Project[]> {
    const params = sourceId !== undefined ? { source_id: sourceId } : undefined;
    const { data } = await client.get('/projects', { params });
    return data;
  },

  async getOrganizationList(): Promise<string[]> {
    const { data } = await client.get('/organizations/list');
    return data;
  },

  async listTeams(organization?: string, sourceId?: number): Promise<GitHubTeam[]> {
    const params: Record<string, string | number> = {};
    if (organization) params.organization = organization;
    if (sourceId !== undefined) params.source_id = sourceId;
    const { data } = await client.get('/teams', { params });
    return data;
  },
};

