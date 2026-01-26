import { client } from './client';
import type {
  CopilotStatus,
  ChatRequest,
  ChatResponse,
  SessionsResponse,
  SessionHistoryResponse,
  CLIValidationResponse,
  StreamEvent,
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
   * Test the stream endpoint connection without sending a real message.
   * This helps diagnose connection issues before attempting to stream.
   */
  testConnection: async (): Promise<{ ok: boolean; error?: string }> => {
    try {
      // Use fetch with a HEAD request to test the endpoint
      const baseUrl = client.defaults.baseURL || '/api/v1';
      const response = await fetch(`${baseUrl}/copilot/chat/stream?message=test`, {
        method: 'GET',
        credentials: 'include',
        headers: {
          'Accept': 'text/event-stream',
        },
      });
      
      // Even if we get an error response, we at least know the endpoint is reachable
      console.log('Copilot connection test response:', response.status, response.statusText);
      
      if (response.status === 401) {
        return { ok: false, error: 'Authentication required - you may need to log in again' };
      }
      if (response.status === 403) {
        return { ok: false, error: 'Copilot is not enabled in settings' };
      }
      if (!response.ok) {
        return { ok: false, error: `Server returned ${response.status}: ${response.statusText}` };
      }
      
      // Close the connection since this is just a test
      response.body?.cancel();
      return { ok: true };
    } catch (err) {
      console.error('Copilot connection test failed:', err);
      return { ok: false, error: `Connection failed: ${err}` };
    }
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
    callbacks?: StreamCallbacks
  ): { eventSource: EventSource; abort: () => void } => {
    const params = new URLSearchParams({
      message: message,
    });
    if (sessionId) {
      params.append('session_id', sessionId);
    }

    const baseUrl = client.defaults.baseURL || '/api/v1';
    const url = `${baseUrl}/copilot/chat/stream?${params.toString()}`;
    
    console.log('Copilot: Creating EventSource connection to:', url);
    
    let eventSource: EventSource;
    try {
      eventSource = new EventSource(url, {
        withCredentials: true,
      });
      console.log('Copilot: EventSource created, readyState:', eventSource.readyState);
    } catch (err) {
      console.error('Copilot: Failed to create EventSource:', err);
      if (callbacks?.onError) {
        callbacks.onError(`Failed to create connection: ${err}`);
      }
      return {
        eventSource: null as unknown as EventSource,
        abort: () => {},
      };
    }

    // Log when connection opens successfully
    eventSource.addEventListener('open', () => {
      console.log('Copilot: EventSource connection opened successfully');
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
            callbacks.onError('Connection error: Failed to parse server response');
          }
        }
      } else {
        // Connection error - EventSource couldn't connect or received non-SSE response
        // Check the readyState to get more info
        const readyState = eventSource.readyState;
        let errorMessage = 'Connection error';
        if (readyState === EventSource.CONNECTING) {
          errorMessage = 'Connection failed: Unable to connect to server. This may be due to authentication issues.';
        } else if (readyState === EventSource.CLOSED) {
          errorMessage = 'Connection closed: Server rejected the connection. Please check if you are logged in.';
        }
        console.error('EventSource error:', { readyState, event, url });
        if (callbacks?.onError) {
          callbacks.onError(errorMessage);
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
