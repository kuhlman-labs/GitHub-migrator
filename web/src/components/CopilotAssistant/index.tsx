import { useState, useRef, useEffect } from 'react';
import { Text, TextInput, Button, Flash, Spinner, IconButton } from '@primer/react';
import { CopilotIcon, PaperAirplaneIcon, TrashIcon, PlusIcon, PersonIcon } from '@primer/octicons-react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { copilotApi } from '../../services/api/copilot';
import type { CopilotMessage, CopilotSession } from '../../types/copilot';
import { PageHeader } from '../common/PageHeader';

export function CopilotAssistant() {
  const [message, setMessage] = useState('');
  const [currentSessionId, setCurrentSessionId] = useState<string | null>(null);
  const [messages, setMessages] = useState<CopilotMessage[]>([]);
  const [isTransmitting, setIsTransmitting] = useState(false); // Track active message transmission
  const isTransmittingRef = useRef(false); // Ref to track current transmitting state (avoids stale closure)
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const queryClient = useQueryClient();

  // Keep ref in sync with state
  useEffect(() => {
    isTransmittingRef.current = isTransmitting;
  }, [isTransmitting]);

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
    onMutate: () => {
      setIsTransmitting(true);
    },
    onSuccess: (response) => {
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
      // Update current session ID if new session was created (do this after adding message)
      if (!currentSessionId) {
        setCurrentSessionId(response.session_id);
      }
      // Invalidate sessions query to update list
      queryClient.invalidateQueries({ queryKey: ['copilot-sessions'] });
    },
    onSettled: () => {
      setIsTransmitting(false);
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

  // Load session history when session changes (but not during active transmission)
  useEffect(() => {
    if (currentSessionId && !isTransmitting) {
      copilotApi.getSessionHistory(currentSessionId).then(response => {
        // Only update if we're still not transmitting (avoid race condition)
        // Use ref to get current value, not stale closure value
        if (!isTransmittingRef.current) {
          setMessages(response.messages);
        }
      }).catch(() => {
        // Session may have expired
        setCurrentSessionId(null);
        setMessages([]);
      });
    }
  }, [currentSessionId, isTransmitting]);

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
      <div className="flex justify-center items-center" style={{ height: '60vh' }}>
        <Spinner size="large" />
      </div>
    );
  }

  // Show error state
  if (statusError) {
    return (
      <div className="p-4">
        <Flash variant="danger">
          Failed to load Copilot status. Please try again later.
        </Flash>
      </div>
    );
  }

  // Show unavailable state
  if (!status?.available) {
    return (
      <div className="p-4">
        <PageHeader
          title="Copilot Assistant"
          description="AI-powered migration planning and execution"
        />
        <div className="mt-4" style={{ maxWidth: 600 }}>
          <Flash variant="warning">
            <Text className="font-bold">Copilot is not available</Text>
            <Text className="block mt-1">
              {status?.unavailable_reason || 'Please enable Copilot in Settings to use this feature.'}
            </Text>
            {!status?.cli_installed && (
              <Text className="block mt-2 text-sm">
                The Copilot CLI must be installed and configured. Go to Settings &gt; Copilot to configure.
              </Text>
            )}
            {status?.license_required && !status?.license_valid && (
              <Text className="block mt-2 text-sm">
                A valid GitHub Copilot license is required. {status?.license_message}
              </Text>
            )}
          </Flash>
        </div>
      </div>
    );
  }

  const sessions = sessionsData?.sessions || [];

  return (
    <div className="flex" style={{ height: 'calc(100vh - 120px)' }}>
      {/* Sidebar - Session list */}
      <div
        className="flex flex-col"
        style={{
          width: 280,
          borderRight: '1px solid var(--borderColor-default)',
          backgroundColor: 'var(--bgColor-muted)',
        }}
      >
        <div className="p-3" style={{ borderBottom: '1px solid var(--borderColor-default)' }}>
          <Button onClick={handleNewChat} leadingVisual={PlusIcon} className="w-full">
            New Chat
          </Button>
        </div>
        <div className="flex-1 overflow-y-auto p-2">
          {sessions.length === 0 ? (
            <Text className="text-center mt-4" style={{ color: 'var(--fgColor-muted)', fontSize: '0.875rem' }}>
              No previous chats
            </Text>
          ) : (
            sessions.map((session) => (
              <div
                key={session.id}
                onClick={() => handleSelectSession(session)}
                className="p-2 mb-1 rounded cursor-pointer flex justify-between items-center"
                style={{
                  backgroundColor: currentSessionId === session.id ? 'var(--bgColor-accent-muted)' : 'transparent',
                }}
                onMouseEnter={(e) => {
                  if (currentSessionId !== session.id) {
                    e.currentTarget.style.backgroundColor = 'var(--bgColor-inset)';
                  }
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.backgroundColor = currentSessionId === session.id ? 'var(--bgColor-accent-muted)' : 'transparent';
                }}
              >
                <div className="flex-1 overflow-hidden">
                  <Text className="font-semibold block overflow-hidden text-ellipsis whitespace-nowrap" style={{ fontSize: '0.875rem' }}>
                    {session.title || 'New Chat'}
                  </Text>
                  <Text style={{ color: 'var(--fgColor-muted)', fontSize: '0.75rem' }}>
                    {session.message_count} messages
                  </Text>
                </div>
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
              </div>
            ))
          )}
        </div>
      </div>

      {/* Main chat area */}
      <div className="flex-1 flex flex-col">
        {/* Header */}
        <div className="p-3" style={{ borderBottom: '1px solid var(--borderColor-default)' }}>
          <div className="flex items-center gap-2">
            <CopilotIcon size={24} />
            <div>
              <Text className="font-bold" style={{ fontSize: '1rem' }}>Copilot Assistant</Text>
              <Text className="block" style={{ color: 'var(--fgColor-muted)', fontSize: '0.875rem' }}>
                AI-powered migration planning and execution
              </Text>
            </div>
          </div>
        </div>

        {/* Messages */}
        <div className="flex-1 overflow-y-auto p-3">
          {messages.length === 0 ? (
            <div className="text-center mt-8">
              <CopilotIcon size={48} />
              <Text className="block font-bold mt-3" style={{ fontSize: '1rem' }}>
                How can I help with your migration?
              </Text>
              <Text className="block mt-2 mx-auto" style={{ color: 'var(--fgColor-muted)', maxWidth: 500 }}>
                I can analyze repositories, find dependencies, create batches, plan migration waves, and more.
                Try asking something like:
              </Text>
              <div className="mt-3 flex flex-col gap-2 items-center">
                {[
                  'Find repositories suitable for a pilot migration',
                  'What are the dependencies for my-org/my-repo?',
                  'Create a batch with low-complexity JavaScript repos',
                  'Plan migration waves to minimize downtime',
                ].map((suggestion) => (
                  <Button
                    key={suggestion}
                    size="small"
                    onClick={() => setMessage(suggestion)}
                    className="border"
                    style={{ maxWidth: 400, borderColor: 'var(--borderColor-default)' }}
                  >
                    {suggestion}
                  </Button>
                ))}
              </div>
            </div>
          ) : (
            <>
              {messages.map((msg) => (
                <div
                  key={msg.id}
                  className="flex gap-2 mb-3"
                  style={{
                    flexDirection: msg.role === 'user' ? 'row-reverse' : 'row',
                  }}
                >
                  <div
                    className="flex items-center justify-center rounded-full"
                    style={{
                      width: 32,
                      height: 32,
                      flexShrink: 0,
                      backgroundColor: msg.role === 'user' ? 'var(--bgColor-accent-emphasis)' : 'var(--bgColor-success-emphasis)',
                    }}
                  >
                    {msg.role === 'user' ? (
                      <PersonIcon size={16} fill="white" />
                    ) : (
                      <CopilotIcon size={16} fill="white" />
                    )}
                  </div>
                  <div
                    className="p-3 rounded"
                    style={{
                      maxWidth: '70%',
                      backgroundColor: msg.role === 'user' ? 'var(--bgColor-accent-muted)' : 'var(--bgColor-muted)',
                    }}
                  >
                    <Text style={{ whiteSpace: 'pre-wrap' }}>{msg.content}</Text>
                    {msg.tool_calls && msg.tool_calls.length > 0 && (
                      <div className="mt-2 pt-2" style={{ borderTop: '1px solid var(--borderColor-default)' }}>
                        <Text className="font-bold" style={{ fontSize: '0.75rem', color: 'var(--fgColor-muted)' }}>
                          Tools Used:
                        </Text>
                        {msg.tool_calls.map((tool) => (
                          <Text key={tool.id} className="block" style={{ fontSize: '0.75rem', color: 'var(--fgColor-muted)' }}>
                            {tool.name} ({tool.status})
                          </Text>
                        ))}
                      </div>
                    )}
                  </div>
                </div>
              ))}
              {sendMessageMutation.isPending && (
                <div className="flex gap-2 mb-3">
                  <div
                    className="flex items-center justify-center rounded-full"
                    style={{
                      width: 32,
                      height: 32,
                      flexShrink: 0,
                      backgroundColor: 'var(--bgColor-success-emphasis)',
                    }}
                  >
                    <CopilotIcon size={16} fill="white" />
                  </div>
                  <div className="p-3 rounded" style={{ backgroundColor: 'var(--bgColor-muted)' }}>
                    <Spinner size="small" />
                  </div>
                </div>
              )}
              <div ref={messagesEndRef} />
            </>
          )}
        </div>

        {/* Input */}
        <div className="p-3" style={{ borderTop: '1px solid var(--borderColor-default)' }}>
          {sendMessageMutation.isError && (
            <Flash variant="danger" className="mb-2">
              Failed to send message. Please try again.
            </Flash>
          )}
          <div className="flex gap-2">
            <TextInput
              value={message}
              onChange={(e) => setMessage(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder="Ask about repositories, dependencies, batches, migration planning..."
              className="flex-1"
              disabled={sendMessageMutation.isPending}
            />
            <Button
              onClick={handleSend}
              disabled={!message.trim() || sendMessageMutation.isPending}
              variant="primary"
            >
              <PaperAirplaneIcon />
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}

export default CopilotAssistant;
