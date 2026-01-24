import { useState, useRef, useEffect } from 'react';
import { Box, Text, TextInput, Button, Flash, Spinner, Avatar, IconButton } from '@primer/react';
import { CopilotIcon, PaperAirplaneIcon, TrashIcon, PlusIcon } from '@primer/octicons-react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { copilotApi } from '../../services/api/copilot';
import type { CopilotMessage, CopilotSession } from '../../types/copilot';
import { PageHeader } from '../common/PageHeader';

export function CopilotAssistant() {
  const [message, setMessage] = useState('');
  const [currentSessionId, setCurrentSessionId] = useState<string | null>(null);
  const [messages, setMessages] = useState<CopilotMessage[]>([]);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const queryClient = useQueryClient();

  // Check Copilot status
  const { data: status, isLoading: statusLoading, error: statusError } = useQuery({
    queryKey: ['copilot-status'],
    queryFn: copilotApi.getStatus,
  });

  // Get sessions list
  const { data: sessionsData } = useQuery({
    queryKey: ['copilot-sessions'],
    queryFn: copilotApi.getSessions,
    enabled: status?.available,
  });

  // Send message mutation
  const sendMessageMutation = useMutation({
    mutationFn: copilotApi.sendMessage,
    onSuccess: (response) => {
      // Update current session ID if new session was created
      if (!currentSessionId) {
        setCurrentSessionId(response.session_id);
      }
      // Add assistant message
      setMessages(prev => [...prev, {
        id: response.message_id,
        session_id: response.session_id,
        role: 'assistant',
        content: response.content,
        tool_calls: response.tool_calls,
        tool_results: response.tool_results,
        created_at: new Date().toISOString(),
      }]);
      // Invalidate sessions query to update list
      queryClient.invalidateQueries({ queryKey: ['copilot-sessions'] });
    },
  });

  // Delete session mutation
  const deleteSessionMutation = useMutation({
    mutationFn: copilotApi.deleteSession,
    onSuccess: () => {
      if (currentSessionId) {
        setCurrentSessionId(null);
        setMessages([]);
      }
      queryClient.invalidateQueries({ queryKey: ['copilot-sessions'] });
    },
  });

  // Load session history when session changes
  useEffect(() => {
    if (currentSessionId) {
      copilotApi.getSessionHistory(currentSessionId).then(response => {
        setMessages(response.messages);
      }).catch(() => {
        // Session may have expired
        setCurrentSessionId(null);
        setMessages([]);
      });
    }
  }, [currentSessionId]);

  // Scroll to bottom when messages change
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const handleSend = () => {
    if (!message.trim() || sendMessageMutation.isPending) return;

    // Add user message immediately
    const userMessage: CopilotMessage = {
      id: Date.now(),
      session_id: currentSessionId || '',
      role: 'user',
      content: message,
      created_at: new Date().toISOString(),
    };
    setMessages(prev => [...prev, userMessage]);

    // Send to API
    sendMessageMutation.mutate({
      session_id: currentSessionId || undefined,
      message: message,
    });

    setMessage('');
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  const handleNewChat = () => {
    setCurrentSessionId(null);
    setMessages([]);
  };

  const handleSelectSession = (session: CopilotSession) => {
    setCurrentSessionId(session.id);
  };

  // Show loading state
  if (statusLoading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '60vh' }}>
        <Spinner size="large" />
      </Box>
    );
  }

  // Show error state
  if (statusError) {
    return (
      <Box sx={{ p: 4 }}>
        <Flash variant="danger">
          Failed to load Copilot status. Please try again later.
        </Flash>
      </Box>
    );
  }

  // Show unavailable state
  if (!status?.available) {
    return (
      <Box sx={{ p: 4 }}>
        <PageHeader
          title="Copilot Assistant"
          subtitle="AI-powered migration planning and execution"
        />
        <Box sx={{ mt: 4, maxWidth: 600 }}>
          <Flash variant="warning">
            <Text sx={{ fontWeight: 'bold' }}>Copilot is not available</Text>
            <Text sx={{ display: 'block', mt: 1 }}>
              {status?.unavailable_reason || 'Please enable Copilot in Settings to use this feature.'}
            </Text>
            {!status?.cli_installed && (
              <Text sx={{ display: 'block', mt: 2, fontSize: 1 }}>
                The Copilot CLI must be installed and configured. Go to Settings &gt; Copilot to configure.
              </Text>
            )}
            {status?.license_required && !status?.license_valid && (
              <Text sx={{ display: 'block', mt: 2, fontSize: 1 }}>
                A valid GitHub Copilot license is required. {status?.license_message}
              </Text>
            )}
          </Flash>
        </Box>
      </Box>
    );
  }

  const sessions = sessionsData?.sessions || [];

  return (
    <Box sx={{ display: 'flex', height: 'calc(100vh - 120px)' }}>
      {/* Sidebar - Session list */}
      <Box
        sx={{
          width: 280,
          borderRight: '1px solid',
          borderColor: 'border.default',
          display: 'flex',
          flexDirection: 'column',
          bg: 'canvas.subtle',
        }}
      >
        <Box sx={{ p: 3, borderBottom: '1px solid', borderColor: 'border.default' }}>
          <Button onClick={handleNewChat} leadingVisual={PlusIcon} sx={{ width: '100%' }}>
            New Chat
          </Button>
        </Box>
        <Box sx={{ flex: 1, overflowY: 'auto', p: 2 }}>
          {sessions.length === 0 ? (
            <Text sx={{ color: 'fg.muted', fontSize: 1, textAlign: 'center', mt: 4 }}>
              No previous chats
            </Text>
          ) : (
            sessions.map((session) => (
              <Box
                key={session.id}
                onClick={() => handleSelectSession(session)}
                sx={{
                  p: 2,
                  mb: 1,
                  borderRadius: 2,
                  cursor: 'pointer',
                  bg: currentSessionId === session.id ? 'accent.subtle' : 'transparent',
                  '&:hover': { bg: 'canvas.inset' },
                  display: 'flex',
                  justifyContent: 'space-between',
                  alignItems: 'center',
                }}
              >
                <Box sx={{ flex: 1, overflow: 'hidden' }}>
                  <Text sx={{ fontWeight: 'semibold', fontSize: 1, display: 'block', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {session.title || 'New Chat'}
                  </Text>
                  <Text sx={{ color: 'fg.muted', fontSize: 0 }}>
                    {session.message_count} messages
                  </Text>
                </Box>
                <IconButton
                  icon={TrashIcon}
                  aria-label="Delete session"
                  variant="invisible"
                  size="small"
                  onClick={(e) => {
                    e.stopPropagation();
                    deleteSessionMutation.mutate(session.id);
                  }}
                />
              </Box>
            ))
          )}
        </Box>
      </Box>

      {/* Main chat area */}
      <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
        {/* Header */}
        <Box sx={{ p: 3, borderBottom: '1px solid', borderColor: 'border.default' }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
            <CopilotIcon size={24} />
            <Box>
              <Text sx={{ fontWeight: 'bold', fontSize: 2 }}>Copilot Assistant</Text>
              <Text sx={{ color: 'fg.muted', fontSize: 1, display: 'block' }}>
                AI-powered migration planning and execution
              </Text>
            </Box>
          </Box>
        </Box>

        {/* Messages */}
        <Box sx={{ flex: 1, overflowY: 'auto', p: 3 }}>
          {messages.length === 0 ? (
            <Box sx={{ textAlign: 'center', mt: 8 }}>
              <CopilotIcon size={48} />
              <Text sx={{ display: 'block', fontSize: 2, fontWeight: 'bold', mt: 3 }}>
                How can I help with your migration?
              </Text>
              <Text sx={{ color: 'fg.muted', display: 'block', mt: 2, maxWidth: 500, mx: 'auto' }}>
                I can analyze repositories, find dependencies, create batches, plan migration waves, and more.
                Try asking something like:
              </Text>
              <Box sx={{ mt: 3, display: 'flex', flexDirection: 'column', gap: 2, alignItems: 'center' }}>
                {[
                  'Find repositories suitable for a pilot migration',
                  'What are the dependencies for my-org/my-repo?',
                  'Create a batch with low-complexity JavaScript repos',
                  'Plan migration waves to minimize downtime',
                ].map((suggestion) => (
                  <Button
                    key={suggestion}
                    variant="outline"
                    size="small"
                    onClick={() => setMessage(suggestion)}
                    sx={{ maxWidth: 400 }}
                  >
                    {suggestion}
                  </Button>
                ))}
              </Box>
            </Box>
          ) : (
            <>
              {messages.map((msg) => (
                <Box
                  key={msg.id}
                  sx={{
                    display: 'flex',
                    gap: 2,
                    mb: 3,
                    flexDirection: msg.role === 'user' ? 'row-reverse' : 'row',
                  }}
                >
                  <Avatar
                    size={32}
                    src={msg.role === 'user' ? undefined : undefined}
                    sx={{ flexShrink: 0 }}
                  />
                  <Box
                    sx={{
                      maxWidth: '70%',
                      p: 3,
                      borderRadius: 2,
                      bg: msg.role === 'user' ? 'accent.subtle' : 'canvas.subtle',
                    }}
                  >
                    <Text sx={{ whiteSpace: 'pre-wrap' }}>{msg.content}</Text>
                    {msg.tool_calls && msg.tool_calls.length > 0 && (
                      <Box sx={{ mt: 2, pt: 2, borderTop: '1px solid', borderColor: 'border.default' }}>
                        <Text sx={{ fontSize: 0, color: 'fg.muted', fontWeight: 'bold' }}>
                          Tools Used:
                        </Text>
                        {msg.tool_calls.map((tool) => (
                          <Text key={tool.id} sx={{ fontSize: 0, color: 'fg.muted', display: 'block' }}>
                            {tool.name} ({tool.status})
                          </Text>
                        ))}
                      </Box>
                    )}
                  </Box>
                </Box>
              ))}
              {sendMessageMutation.isPending && (
                <Box sx={{ display: 'flex', gap: 2, mb: 3 }}>
                  <Avatar size={32} />
                  <Box sx={{ p: 3, borderRadius: 2, bg: 'canvas.subtle' }}>
                    <Spinner size="small" />
                  </Box>
                </Box>
              )}
              <div ref={messagesEndRef} />
            </>
          )}
        </Box>

        {/* Input */}
        <Box sx={{ p: 3, borderTop: '1px solid', borderColor: 'border.default' }}>
          {sendMessageMutation.isError && (
            <Flash variant="danger" sx={{ mb: 2 }}>
              Failed to send message. Please try again.
            </Flash>
          )}
          <Box sx={{ display: 'flex', gap: 2 }}>
            <TextInput
              value={message}
              onChange={(e) => setMessage(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder="Ask about repositories, dependencies, batches, migration planning..."
              sx={{ flex: 1 }}
              disabled={sendMessageMutation.isPending}
            />
            <Button
              onClick={handleSend}
              disabled={!message.trim() || sendMessageMutation.isPending}
              variant="primary"
            >
              <PaperAirplaneIcon />
            </Button>
          </Box>
        </Box>
      </Box>
    </Box>
  );
}

export default CopilotAssistant;
