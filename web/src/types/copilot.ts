// Copilot API types

export interface CopilotStatus {
  enabled: boolean;
  available: boolean;
  cli_installed: boolean;
  cli_version?: string;
  license_required: boolean;
  license_valid: boolean;
  license_message?: string;
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
