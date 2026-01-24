import client from './client';
import type {
  CopilotStatus,
  ChatRequest,
  ChatResponse,
  SessionsResponse,
  SessionHistoryResponse,
  CLIValidationResponse,
} from '../../types/copilot';

/**
 * Copilot API service
 */
export const copilotApi = {
  /**
   * Get Copilot status for the current user
   */
  getStatus: async (): Promise<CopilotStatus> => {
    const response = await client.get<CopilotStatus>('/copilot/status');
    return response.data;
  },

  /**
   * Send a message to Copilot
   */
  sendMessage: async (request: ChatRequest): Promise<ChatResponse> => {
    const response = await client.post<ChatResponse>('/copilot/chat', request);
    return response.data;
  },

  /**
   * List all chat sessions for the current user
   */
  getSessions: async (): Promise<SessionsResponse> => {
    const response = await client.get<SessionsResponse>('/copilot/sessions');
    return response.data;
  },

  /**
   * Get message history for a session
   */
  getSessionHistory: async (sessionId: string): Promise<SessionHistoryResponse> => {
    const response = await client.get<SessionHistoryResponse>(`/copilot/sessions/${sessionId}/history`);
    return response.data;
  },

  /**
   * Delete a chat session
   */
  deleteSession: async (sessionId: string): Promise<void> => {
    await client.delete(`/copilot/sessions/${sessionId}`);
  },

  /**
   * Validate Copilot CLI installation
   */
  validateCLI: async (cliPath: string): Promise<CLIValidationResponse> => {
    const response = await client.post<CLIValidationResponse>('/copilot/validate-cli', { cli_path: cliPath });
    return response.data;
  },
};

export default copilotApi;
