import { describe, it, expect, vi, beforeEach, beforeAll } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { ThemeProvider } from '@primer/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { CopilotAssistant } from './index';

// Mock scrollIntoView which doesn't exist in JSDOM
beforeAll(() => {
  Element.prototype.scrollIntoView = vi.fn();
});

// Mock the copilot API
const mockGetStatus = vi.fn();
const mockGetSessions = vi.fn();
const mockSendMessage = vi.fn();
const mockDeleteSession = vi.fn();
const mockGetSessionHistory = vi.fn();

vi.mock('../../services/api/copilot', () => ({
  copilotApi: {
    getStatus: () => mockGetStatus(),
    getSessions: () => mockGetSessions(),
    sendMessage: (req: unknown) => mockSendMessage(req),
    deleteSession: (id: string) => mockDeleteSession(id),
    getSessionHistory: (id: string) => mockGetSessionHistory(id),
    validateCLI: vi.fn(),
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
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>{children}</ThemeProvider>
    </QueryClientProvider>
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
    
    // Loading state should show spinner with "Loading" accessible text
    expect(screen.getByText('Loading')).toBeInTheDocument();
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
    
    mockSendMessage.mockResolvedValue({
      session_id: 'new-session-123',
      message_id: 1,
      content: 'I can help you find repositories for migration.',
      done: true,
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
    const sendButton = screen.getByRole('button', { name: '' }); // PaperAirplaneIcon has no accessible name
    fireEvent.click(sendButton);

    // Should call the API
    await waitFor(() => {
      expect(mockSendMessage).toHaveBeenCalledWith({
        session_id: undefined,
        message: 'Find pilot repos',
      });
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
});
