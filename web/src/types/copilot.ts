// Copilot API types

export interface CopilotStatus {
  enabled: boolean;
  available: boolean;
  cli_installed: boolean;
  cli_version?: string;
  unavailable_reason?: string;
}

export interface CopilotSession {
  id: string;
  title: string;
  message_count: number;
  created_at: string;
  updated_at: string;
  expires_at: string;
}

export interface CopilotMessage {
  id: number;
  session_id: string;
  role: 'user' | 'assistant' | 'system';
  content: string;
  tool_calls?: ToolCall[];
  tool_results?: ToolResult[];
  created_at: string;
}

export interface ToolCall {
  id: string;
  name: string;
  args: Record<string, unknown>;
  status: 'pending' | 'completed' | 'failed';
  duration_ms?: number;
}

export interface ToolResult {
  tool_call_id: string;
  success: boolean;
  result?: unknown;
  error?: string;
}

export interface ChatRequest {
  session_id?: string;
  message: string;
  model?: string;
}

export interface ModelInfo {
  id: string;
  name: string;
  description?: string;
  is_default?: boolean;
}

export interface ModelsResponse {
  models: ModelInfo[];
  default_model: string;
}

export interface ChatResponse {
  session_id: string;
  message_id: number;
  content: string;
  tool_calls?: ToolCall[];
  tool_results?: ToolResult[];
  done: boolean;
}

export interface SessionsResponse {
  sessions: CopilotSession[];
  count: number;
}

export interface SessionHistoryResponse {
  session_id: string;
  messages: CopilotMessage[];
  count: number;
}

export interface CLIValidationResponse {
  available: boolean;
  version?: string;
  error?: string;
}

// Streaming types for SSE events

export type StreamEventType = 'delta' | 'tool_call' | 'tool_result' | 'done' | 'error' | 'session';

export interface StreamEvent {
  type: StreamEventType;
  content?: string;
  tool_call?: ToolCall;
  tool_result?: ToolResult;
  error?: string;
  session_id?: string;
}

export interface StreamSessionEvent {
  session_id: string;
}

export interface StreamDeltaEvent {
  type: 'delta';
  content: string;
}

export interface StreamToolCallEvent {
  type: 'tool_call';
  tool_call: ToolCall;
}

export interface StreamToolResultEvent {
  type: 'tool_result';
  tool_result: ToolResult;
}

export interface StreamDoneEvent {
  type: 'done';
  content: string;
}

export interface StreamErrorEvent {
  type: 'error';
  error: string;
}
