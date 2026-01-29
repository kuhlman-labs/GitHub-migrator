import { client } from './client';
import type {
  CopilotStatus,
  ChatRequest,
  ChatResponse,
  SessionsResponse,
  SessionHistoryResponse,
  CLIValidationResponse,
  StreamEvent,
  ModelsResponse,
} from '../../types/copilot';

/**
 * Callback types for streaming events
 */
export interface StreamCallbacks {
  onDelta?: (content: string) => void;
  onToolCall?: (toolCall: StreamEvent['tool_call']) => void;
  onToolResult?: (toolResult: StreamEvent['tool_result']) => void;
  onDone?: (content: string) => void;
  onError?: (error: string) => void;
  onSessionId?: (sessionId: string) => void;
}

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
   * Get available AI models
   */
  getModels: async (): Promise<ModelsResponse> => {
    const response = await client.get<ModelsResponse>('/copilot/models');
    return response.data;
  },

  /**
   * Send a message to Copilot (non-streaming)
   */
  sendMessage: async (request: ChatRequest): Promise<ChatResponse> => {
    const response = await client.post<ChatResponse>('/copilot/chat', request);
    return response.data;
  },

  /**
   * Send a message to Copilot with streaming response
   * Returns an EventSource that can be used to receive streaming events
   */
  streamMessage: (
    message: string,
    sessionId?: string,
    model?: string,
    callbacks?: StreamCallbacks
  ): { eventSource: EventSource; abort: () => void } => {
    const params = new URLSearchParams({
      message: message,
    });
    if (sessionId) {
      params.append('session_id', sessionId);
    }
    if (model) {
      params.append('model', model);
    }

    const baseUrl = client.defaults.baseURL || '/api/v1';
    const url = `${baseUrl}/copilot/chat/stream?${params.toString()}`;
    
    const eventSource = new EventSource(url, {
      withCredentials: true,
    });

    // Handle session event (provides session ID for new sessions)
    eventSource.addEventListener('session', (event) => {
      try {
        const data = JSON.parse(event.data);
        if (data.session_id && callbacks?.onSessionId) {
          callbacks.onSessionId(data.session_id);
        }
      } catch {
        console.error('Failed to parse session event:', event.data);
      }
    });

    // Handle delta events (streaming content)
    eventSource.addEventListener('delta', (event) => {
      try {
        const data = JSON.parse(event.data) as StreamEvent;
        if (data.content && callbacks?.onDelta) {
          callbacks.onDelta(data.content);
        }
      } catch {
        console.error('Failed to parse delta event:', event.data);
      }
    });

    // Handle tool_call events
    eventSource.addEventListener('tool_call', (event) => {
      try {
        const data = JSON.parse(event.data) as StreamEvent;
        if (data.tool_call && callbacks?.onToolCall) {
          callbacks.onToolCall(data.tool_call);
        }
      } catch {
        console.error('Failed to parse tool_call event:', event.data);
      }
    });

    // Handle tool_result events
    eventSource.addEventListener('tool_result', (event) => {
      try {
        const data = JSON.parse(event.data) as StreamEvent;
        if (data.tool_result && callbacks?.onToolResult) {
          callbacks.onToolResult(data.tool_result);
        }
      } catch {
        console.error('Failed to parse tool_result event:', event.data);
      }
    });

    // Handle done event
    eventSource.addEventListener('done', (event) => {
      try {
        const data = JSON.parse(event.data) as StreamEvent;
        if (callbacks?.onDone) {
          callbacks.onDone(data.content || '');
        }
        eventSource.close();
      } catch {
        console.error('Failed to parse done event:', event.data);
      }
    });

    // Handle error event
    eventSource.addEventListener('error', (event) => {
      if (event instanceof MessageEvent) {
        try {
          const data = JSON.parse(event.data) as StreamEvent;
          if (data.error && callbacks?.onError) {
            callbacks.onError(data.error);
          }
        } catch {
          // If it's not a JSON error, it might be a connection error
          if (callbacks?.onError) {
            callbacks.onError('Connection error');
          }
        }
      } else {
        // Connection error
        if (callbacks?.onError) {
          callbacks.onError('Connection error');
        }
      }
      eventSource.close();
    });

    return {
      eventSource,
      abort: () => {
        eventSource.close();
      },
    };
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
