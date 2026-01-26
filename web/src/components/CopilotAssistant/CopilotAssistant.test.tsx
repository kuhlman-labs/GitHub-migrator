import { describe, it, expect, vi, beforeEach, beforeAll } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { ThemeProvider } from '@primer/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { CopilotAssistant } from './index';
import { ToastProvider } from '../../contexts/ToastContext';
import { AuthProvider } from '../../contexts/AuthContext';

// Mock scrollIntoView which doesn't exist in JSDOM
beforeAll(() => {
  Element.prototype.scrollIntoView = vi.fn();
  
  // Mock matchMedia for responsive behavior
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  });
});

// Mock the copilot API
const mockGetStatus = vi.fn();
const mockGetSessions = vi.fn();
const mockStreamMessage = vi.fn();
const mockDeleteSession = vi.fn();
const mockGetSessionHistory = vi.fn();

vi.mock('../../services/api/copilot', () => ({
  copilotApi: {
    getStatus: () => mockGetStatus(),
    getSessions: () => mockGetSessions(),
    streamMessage: (message: string, sessionId: string | undefined, callbacks: unknown) => mockStreamMessage(message, sessionId, callbacks),
    deleteSession: (id: string) => mockDeleteSession(id),
    getSessionHistory: (id: string) => mockGetSessionHistory(id),
    validateCLI: vi.fn(),
  },
}));

// Mock the main API for auth config
vi.mock('../../services/api', () => ({
  api: {
    getAuthConfig: vi.fn().mockResolvedValue({ enabled: false }),
    getCurrentUser: vi.fn().mockRejectedValue(new Error('Not authenticated')),
    logout: vi.fn(),
  },
}));

function TestWrapper({ children }: { children: React.ReactNode }) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        gcTime: 0,
      },
    },
  });
  
  return (
    <MemoryRouter>
      <QueryClientProvider client={queryClient}>
        <ThemeProvider>
          <AuthProvider>
            <ToastProvider>{children}</ToastProvider>
          </AuthProvider>
        </ThemeProvider>
      </QueryClientProvider>
    </MemoryRouter>
  );
}

describe('CopilotAssistant', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetSessions.mockResolvedValue({ sessions: [], count: 0 });
  });

  it('shows loading state initially', () => {
    mockGetStatus.mockReturnValue(new Promise(() => {})); // Never resolves
    render(
      <TestWrapper>
        <CopilotAssistant />
      </TestWrapper>
    );
    
    // Loading state should show spinner with role="status" and aria-label
    expect(screen.getByRole('status', { name: 'Loading Copilot' })).toBeInTheDocument();
  });

  it('shows unavailable message when Copilot is not available', async () => {
    mockGetStatus.mockResolvedValue({
      enabled: false,
      available: false,
      cli_installed: false,
      license_required: false,
      license_valid: false,
      unavailable_reason: 'Copilot is not enabled in settings',
    });

    render(
      <TestWrapper>
        <CopilotAssistant />
      </TestWrapper>
    );

    await waitFor(() => {
      expect(screen.getByText('Copilot is not available')).toBeInTheDocument();
    });

    expect(screen.getByText(/Copilot is not enabled in settings/)).toBeInTheDocument();
  });

  it('shows chat interface when Copilot is available', async () => {
    mockGetStatus.mockResolvedValue({
      enabled: true,
      available: true,
      cli_installed: true,
      license_required: false,
      license_valid: true,
    });

    render(
      <TestWrapper>
        <CopilotAssistant />
      </TestWrapper>
    );

    await waitFor(() => {
      expect(screen.getByText('Copilot Assistant')).toBeInTheDocument();
    });

    // Should show the welcome message
    expect(screen.getByText('How can I help with your migration?')).toBeInTheDocument();
    
    // Should show suggestion buttons
    expect(screen.getByText('Find repositories suitable for a pilot migration')).toBeInTheDocument();
  });

  it('allows typing a message and sending', async () => {
    mockGetStatus.mockResolvedValue({
      enabled: true,
      available: true,
      cli_installed: true,
      license_required: false,
      license_valid: true,
    });
    
    // Mock streamMessage to return an abort function and call the callbacks
    mockStreamMessage.mockImplementation((_message: string, _sessionId: string | undefined, callbacks: {
      onSessionId?: (sessionId: string) => void;
      onDone?: (content: string) => void;
    }) => {
      // Simulate streaming behavior
      setTimeout(() => {
        callbacks.onSessionId?.('new-session-123');
        callbacks.onDone?.('I can help you find repositories for migration.');
      }, 10);
      return { abort: vi.fn() };
    });

    render(
      <TestWrapper>
        <CopilotAssistant />
      </TestWrapper>
    );

    await waitFor(() => {
      expect(screen.getByText('How can I help with your migration?')).toBeInTheDocument();
    });

    // Type a message
    const input = screen.getByPlaceholderText(/Ask about repositories/);
    fireEvent.change(input, { target: { value: 'Find pilot repos' } });

    // Click send button
    const sendButton = screen.getByRole('button', { name: 'Send message' });
    fireEvent.click(sendButton);

    // Should call the streaming API
    await waitFor(() => {
      expect(mockStreamMessage).toHaveBeenCalledWith(
        'Find pilot repos',
        undefined,
        expect.any(Object)
      );
    });
  });

  it('shows new chat button', async () => {
    mockGetStatus.mockResolvedValue({
      enabled: true,
      available: true,
      cli_installed: true,
      license_required: false,
      license_valid: true,
    });

    render(
      <TestWrapper>
        <CopilotAssistant />
      </TestWrapper>
    );

    await waitFor(() => {
      expect(screen.getByText('New Chat')).toBeInTheDocument();
    });
  });

  it('shows no previous chats message in sidebar', async () => {
    mockGetStatus.mockResolvedValue({
      enabled: true,
      available: true,
      cli_installed: true,
      license_required: false,
      license_valid: true,
    });

    render(
      <TestWrapper>
        <CopilotAssistant />
      </TestWrapper>
    );

    await waitFor(() => {
      expect(screen.getByText('No previous chats')).toBeInTheDocument();
    });
  });

  it('shows session list when sessions exist', async () => {
    mockGetStatus.mockResolvedValue({
      enabled: true,
      available: true,
      cli_installed: true,
      license_required: false,
      license_valid: true,
    });

    mockGetSessions.mockResolvedValue({
      sessions: [
        {
          id: 'session-1',
          title: 'Migration Planning',
          message_count: 5,
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T01:00:00Z',
          expires_at: '2024-01-02T00:00:00Z',
        },
      ],
      count: 1,
    });

    render(
      <TestWrapper>
        <CopilotAssistant />
      </TestWrapper>
    );

    await waitFor(() => {
      expect(screen.getByText('Migration Planning')).toBeInTheDocument();
    });

    expect(screen.getByText('5 messages')).toBeInTheDocument();
  });

  it('shows license warning when license is required but invalid', async () => {
    mockGetStatus.mockResolvedValue({
      enabled: true,
      available: false,
      cli_installed: true,
      license_required: true,
      license_valid: false,
      license_message: 'No Copilot license found',
      unavailable_reason: 'No Copilot license found',
    });

    render(
      <TestWrapper>
        <CopilotAssistant />
      </TestWrapper>
    );

    await waitFor(() => {
      expect(screen.getByText('Copilot is not available')).toBeInTheDocument();
    });

    // Use getAllByText since the message appears in multiple places
    expect(screen.getAllByText(/No Copilot license found/).length).toBeGreaterThan(0);
  });

  it('shows CLI not installed warning', async () => {
    mockGetStatus.mockResolvedValue({
      enabled: true,
      available: false,
      cli_installed: false,
      license_required: false,
      license_valid: true,
      unavailable_reason: 'Copilot CLI is not installed',
    });

    render(
      <TestWrapper>
        <CopilotAssistant />
      </TestWrapper>
    );

    await waitFor(() => {
      expect(screen.getByText('Copilot is not available')).toBeInTheDocument();
    });

    expect(screen.getByText(/The Copilot CLI must be installed and configured/)).toBeInTheDocument();
  });

  it('clicking suggestion fills input', async () => {
    mockGetStatus.mockResolvedValue({
      enabled: true,
      available: true,
      cli_installed: true,
      license_required: false,
      license_valid: true,
    });

    render(
      <TestWrapper>
        <CopilotAssistant />
      </TestWrapper>
    );

    await waitFor(() => {
      expect(screen.getByText('How can I help with your migration?')).toBeInTheDocument();
    });

    // Click a suggestion
    const suggestion = screen.getByText('Find repositories suitable for a pilot migration');
    fireEvent.click(suggestion);

    // Input should be filled
    const input = screen.getByPlaceholderText(/Ask about repositories/) as HTMLInputElement;
    expect(input.value).toBe('Find repositories suitable for a pilot migration');
  });

  it('has proper accessibility attributes on sidebar', async () => {
    mockGetStatus.mockResolvedValue({
      enabled: true,
      available: true,
      cli_installed: true,
      license_required: false,
      license_valid: true,
    });

    render(
      <TestWrapper>
        <CopilotAssistant />
      </TestWrapper>
    );

    await waitFor(() => {
      expect(screen.getByText('Copilot Assistant')).toBeInTheDocument();
    });

    // Check sidebar accessibility
    expect(screen.getByRole('complementary', { name: 'Chat history sidebar' })).toBeInTheDocument();
    expect(screen.getByRole('listbox', { name: 'Previous chats' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Start new chat' })).toBeInTheDocument();
  });

  it('has proper accessibility attributes on message area', async () => {
    mockGetStatus.mockResolvedValue({
      enabled: true,
      available: true,
      cli_installed: true,
      license_required: false,
      license_valid: true,
    });

    render(
      <TestWrapper>
        <CopilotAssistant />
      </TestWrapper>
    );

    await waitFor(() => {
      expect(screen.getByText('Copilot Assistant')).toBeInTheDocument();
    });

    // Check message area accessibility
    expect(screen.getByRole('log', { name: 'Chat messages' })).toBeInTheDocument();
    expect(screen.getByLabelText('Message input')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Send message' })).toBeInTheDocument();
  });

  it('has sidebar toggle button', async () => {
    mockGetStatus.mockResolvedValue({
      enabled: true,
      available: true,
      cli_installed: true,
      license_required: false,
      license_valid: true,
    });

    render(
      <TestWrapper>
        <CopilotAssistant />
      </TestWrapper>
    );

    await waitFor(() => {
      expect(screen.getByText('Copilot Assistant')).toBeInTheDocument();
    });

    // Check toggle button exists
    expect(screen.getByRole('button', { name: /sidebar/i })).toBeInTheDocument();
  });

  it('shows thinking indicator while streaming', async () => {
    mockGetStatus.mockResolvedValue({
      enabled: true,
      available: true,
      cli_installed: true,
      license_required: false,
      license_valid: true,
    });

    // Mock streamMessage to simulate loading state (never completes)
    mockStreamMessage.mockImplementation(() => {
      return { abort: vi.fn() };
    });

    render(
      <TestWrapper>
        <CopilotAssistant />
      </TestWrapper>
    );

    await waitFor(() => {
      expect(screen.getByText('How can I help with your migration?')).toBeInTheDocument();
    });

    // Type and send a message
    const input = screen.getByPlaceholderText(/Ask about repositories/);
    fireEvent.change(input, { target: { value: 'Test message' } });
    
    const sendButton = screen.getByRole('button', { name: 'Send message' });
    fireEvent.click(sendButton);

    // Should show thinking indicator
    await waitFor(() => {
      expect(screen.getByText('Thinking...')).toBeInTheDocument();
    });

    // Should show stop button
    expect(screen.getByRole('button', { name: 'Stop response' })).toBeInTheDocument();
  });

  it('shows retry button after error', async () => {
    mockGetStatus.mockResolvedValue({
      enabled: true,
      available: true,
      cli_installed: true,
      license_required: false,
      license_valid: true,
    });

    // Mock streamMessage to simulate an error
    mockStreamMessage.mockImplementation((_message: string, _sessionId: string | undefined, callbacks: {
      onError?: (error: string) => void;
    }) => {
      setTimeout(() => {
        callbacks.onError?.('Connection failed');
      }, 10);
      return { abort: vi.fn() };
    });

    render(
      <TestWrapper>
        <CopilotAssistant />
      </TestWrapper>
    );

    await waitFor(() => {
      expect(screen.getByText('How can I help with your migration?')).toBeInTheDocument();
    });

    // Type and send a message
    const input = screen.getByPlaceholderText(/Ask about repositories/);
    fireEvent.change(input, { target: { value: 'Test message' } });
    
    const sendButton = screen.getByRole('button', { name: 'Send message' });
    fireEvent.click(sendButton);

    // Should show retry button
    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Retry last message' })).toBeInTheDocument();
    });
  });

  it('session list items have proper ARIA attributes', async () => {
    mockGetStatus.mockResolvedValue({
      enabled: true,
      available: true,
      cli_installed: true,
      license_required: false,
      license_valid: true,
    });

    mockGetSessions.mockResolvedValue({
      sessions: [
        {
          id: 'session-1',
          title: 'Test Chat',
          message_count: 3,
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T01:00:00Z',
          expires_at: '2024-01-02T00:00:00Z',
        },
      ],
      count: 1,
    });

    render(
      <TestWrapper>
        <CopilotAssistant />
      </TestWrapper>
    );

    await waitFor(() => {
      expect(screen.getByText('Test Chat')).toBeInTheDocument();
    });

    // Check session item has proper role
    const sessionItem = screen.getByRole('option');
    expect(sessionItem).toHaveAttribute('aria-selected', 'false');
    
    // Check delete button is accessible
    expect(screen.getByRole('button', { name: /Delete chat: Test Chat/i })).toBeInTheDocument();
  });

  it('shows authentication required message when API returns 401', async () => {
    // Simulate 401 error from the API
    const authError = new Error('Unauthorized') as Error & { response?: { status: number } };
    authError.response = { status: 401 };
    mockGetStatus.mockRejectedValue(authError);

    render(
      <TestWrapper>
        <CopilotAssistant />
      </TestWrapper>
    );

    await waitFor(() => {
      expect(screen.getByText('Authentication Required')).toBeInTheDocument();
    });

    // Should show instructions about configuring auth
    expect(screen.getByText(/Authentication is not enabled/)).toBeInTheDocument();
    expect(screen.getByText(/Settings/)).toBeInTheDocument();
  });
});
