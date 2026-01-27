import { useState, useRef, useEffect, useCallback } from 'react';
import { Link } from 'react-router-dom';
import { Text, TextInput, Button, Flash, Spinner, IconButton } from '@primer/react';
import { 
  CopilotIcon, 
  PaperAirplaneIcon, 
  TrashIcon, 
  PlusIcon, 
  PersonIcon, 
  StopIcon,
  CopyIcon,
  CheckIcon,
  SyncIcon,
  SidebarCollapseIcon,
  SidebarExpandIcon,
  ToolsIcon,
  CheckCircleFillIcon,
  XCircleFillIcon,
  ClockIcon,
  ShieldLockIcon,
  GearIcon
} from '@primer/octicons-react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { copilotApi } from '../../services/api/copilot';
import type { CopilotMessage, CopilotSession, ToolCall } from '../../types/copilot';
import { PageHeader } from '../common/PageHeader';
import { useToast } from '../../contexts/ToastContext';
import { useAuth } from '../../contexts/AuthContext';

// Breakpoint for mobile responsiveness
const MOBILE_BREAKPOINT = 768;

// Generate unique IDs for messages to avoid React key collisions
let messageIdCounter = 0;
function generateMessageId(): number {
  return Date.now() * 1000 + (messageIdCounter++ % 1000);
}

export function CopilotAssistant() {
  const [message, setMessage] = useState('');
  const [currentSessionId, setCurrentSessionId] = useState<string | null>(null);
  const [messages, setMessages] = useState<CopilotMessage[]>([]);
  const [streamingContent, setStreamingContent] = useState<string>('');
  const [isStreaming, setIsStreaming] = useState(false);
  const [activeToolCalls, setActiveToolCalls] = useState<ToolCall[]>([]);
  const [shouldFetchHistory, setShouldFetchHistory] = useState(false);
  const [copiedMessageId, setCopiedMessageId] = useState<number | null>(null);
  const [failedMessageContent, setFailedMessageContent] = useState<string | null>(null);
  const [sidebarOpen, setSidebarOpen] = useState(true);
  const [isMobile, setIsMobile] = useState(false);
  const [focusedSessionIndex, setFocusedSessionIndex] = useState(-1);
  
  const fetchRequestIdRef = useRef(0);
  const streamAbortRef = useRef<(() => void) | null>(null);
  const messagesContainerRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const sessionListRef = useRef<HTMLDivElement>(null);
  const streamingContentRef = useRef<string>('');
  const messageAddedRef = useRef(false);
  const queryClient = useQueryClient();
  const { showError, showSuccess } = useToast();
  const { authEnabled, login } = useAuth();

  // Check for mobile viewport
  useEffect(() => {
    const checkMobile = () => {
      const mobile = window.innerWidth < MOBILE_BREAKPOINT;
      setIsMobile(mobile);
      if (mobile) {
        setSidebarOpen(false);
      }
    };
    
    checkMobile();
    window.addEventListener('resize', checkMobile);
    return () => window.removeEventListener('resize', checkMobile);
  }, []);

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

  // Delete session mutation
  const deleteSessionMutation = useMutation({
    mutationFn: copilotApi.deleteSession,
    onSuccess: (_data, deletedSessionId) => {
      if (currentSessionId === deletedSessionId) {
        setCurrentSessionId(null);
        setMessages([]);
      }
      queryClient.invalidateQueries({ queryKey: ['copilot-sessions'] });
      showSuccess('Chat deleted');
    },
    onError: () => {
      showError('Failed to delete chat');
    },
  });

  // Load session history
  useEffect(() => {
    if (currentSessionId && shouldFetchHistory) {
      const requestId = ++fetchRequestIdRef.current;
      
      copilotApi.getSessionHistory(currentSessionId).then(response => {
        if (requestId === fetchRequestIdRef.current) {
          setMessages(response.messages);
        }
      }).catch(() => {
        if (requestId === fetchRequestIdRef.current) {
          setCurrentSessionId(null);
          setMessages([]);
          showError('Failed to load chat history');
        }
      }).finally(() => {
        if (requestId === fetchRequestIdRef.current) {
          setShouldFetchHistory(false);
        }
      });
    }
  }, [currentSessionId, shouldFetchHistory, showError]);

  // Scroll to bottom when messages change
  useEffect(() => {
    const container = messagesContainerRef.current;
    if (container) {
      // Scroll the container directly to avoid page scroll
      container.scrollTo({
        top: container.scrollHeight,
        behavior: 'smooth'
      });
    }
  }, [messages, streamingContent]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (streamAbortRef.current) {
        streamAbortRef.current();
      }
    };
  }, []);

  // Copy message to clipboard
  const handleCopyMessage = useCallback(async (messageId: number, content: string) => {
    try {
      await navigator.clipboard.writeText(content);
      setCopiedMessageId(messageId);
      setTimeout(() => setCopiedMessageId(null), 2000);
    } catch {
      showError('Failed to copy to clipboard');
    }
  }, [showError]);

  // Retry failed message
  const handleRetry = useCallback(() => {
    if (failedMessageContent) {
      setMessage(failedMessageContent);
      setFailedMessageContent(null);
      // Remove the error message
      setMessages(prev => prev.filter(m => !m.content.startsWith('Error:')));
      // Focus input
      inputRef.current?.focus();
    }
  }, [failedMessageContent]);

  const handleSendStreaming = useCallback(() => {
    if (!message.trim() || isStreaming) return;

    const userMessageContent = message;
    setMessage('');
    setFailedMessageContent(null);

    const userMessage: CopilotMessage = {
      id: generateMessageId(),
      session_id: currentSessionId || '',
      role: 'user',
      content: userMessageContent,
      created_at: new Date().toISOString(),
    };
    setMessages(prev => [...prev, userMessage]);
    setIsStreaming(true);
    setStreamingContent('');
    streamingContentRef.current = '';
    messageAddedRef.current = false;
    setActiveToolCalls([]);

    // Capture tool calls at the start for closure safety
    let capturedToolCalls: ToolCall[] = [];

    const { abort } = copilotApi.streamMessage(
      userMessageContent,
      currentSessionId || undefined,
      {
        onSessionId: (sessionId) => {
          if (!currentSessionId) {
            setCurrentSessionId(sessionId);
          }
        },
        onDelta: (content) => {
          streamingContentRef.current += content;
          setStreamingContent(prev => prev + content);
        },
        onToolCall: (toolCall) => {
          if (toolCall) {
            capturedToolCalls = [...capturedToolCalls, toolCall];
            setActiveToolCalls(capturedToolCalls);
          }
        },
        onToolResult: (toolResult) => {
          if (toolResult) {
            capturedToolCalls = capturedToolCalls.map(tc => 
              tc.id === toolResult.tool_call_id 
                ? { ...tc, status: toolResult.success ? 'completed' : 'failed' as const }
                : tc
            );
            setActiveToolCalls(capturedToolCalls);
          }
        },
        onDone: (content) => {
          // Prevent duplicate message addition (React Strict Mode can call this twice)
          if (messageAddedRef.current) {
            return;
          }
          messageAddedRef.current = true;
          
          // Use the ref to get the final content
          const finalContent = content || streamingContentRef.current;
          const assistantMessage: CopilotMessage = {
            id: generateMessageId(),
            session_id: currentSessionId || '',
            role: 'assistant',
            content: finalContent,
            tool_calls: capturedToolCalls.length > 0 ? capturedToolCalls : undefined,
            created_at: new Date().toISOString(),
          };
          setMessages(prev => [...prev, assistantMessage]);
          setStreamingContent('');
          streamingContentRef.current = '';
          setActiveToolCalls([]);
          setIsStreaming(false);
          streamAbortRef.current = null;
          queryClient.invalidateQueries({ queryKey: ['copilot-sessions'] });
          // Return focus to input
          setTimeout(() => inputRef.current?.focus(), 100);
        },
        onError: (error) => {
          // Prevent duplicate error message addition
          if (messageAddedRef.current) {
            return;
          }
          messageAddedRef.current = true;
          
          console.error('Stream error:', error);
          showError(`Chat error: ${error}`);
          setFailedMessageContent(userMessageContent);
          
          const errorMessage: CopilotMessage = {
            id: generateMessageId(),
            session_id: currentSessionId || '',
            role: 'assistant',
            content: `Error: ${error}`,
            created_at: new Date().toISOString(),
          };
          setMessages(prev => [...prev, errorMessage]);
          setStreamingContent('');
          streamingContentRef.current = '';
          setActiveToolCalls([]);
          setIsStreaming(false);
          streamAbortRef.current = null;
        },
      }
    );

    streamAbortRef.current = abort;
  }, [message, currentSessionId, isStreaming, queryClient, showError]);

  const handleStopStreaming = useCallback(() => {
    if (streamAbortRef.current) {
      streamAbortRef.current();
      streamAbortRef.current = null;
    }
    
    if (streamingContent) {
      const assistantMessage: CopilotMessage = {
        id: generateMessageId(),
        session_id: currentSessionId || '',
        role: 'assistant',
        content: streamingContent + '\n\n[Stopped]',
        tool_calls: activeToolCalls.length > 0 ? activeToolCalls : undefined,
        created_at: new Date().toISOString(),
      };
      setMessages(prev => [...prev, assistantMessage]);
    }
    
    setStreamingContent('');
    setActiveToolCalls([]);
    setIsStreaming(false);
    showSuccess('Response stopped');
  }, [currentSessionId, streamingContent, activeToolCalls, showSuccess]);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSendStreaming();
    }
  };

  const handleNewChat = () => {
    if (streamAbortRef.current) {
      streamAbortRef.current();
      streamAbortRef.current = null;
    }
    fetchRequestIdRef.current++;
    setShouldFetchHistory(false);
    setCurrentSessionId(null);
    setMessages([]);
    setStreamingContent('');
    setActiveToolCalls([]);
    setIsStreaming(false);
    setFailedMessageContent(null);
    if (isMobile) {
      setSidebarOpen(false);
    }
    inputRef.current?.focus();
  };

  const handleSelectSession = useCallback((session: CopilotSession) => {
    if (streamAbortRef.current) {
      streamAbortRef.current();
      streamAbortRef.current = null;
    }
    setStreamingContent('');
    setActiveToolCalls([]);
    setIsStreaming(false);
    setFailedMessageContent(null);
    setShouldFetchHistory(true);
    setCurrentSessionId(session.id);
    if (isMobile) {
      setSidebarOpen(false);
    }
  }, [isMobile]);

  // Keyboard navigation for session list
  const handleSessionKeyDown = useCallback((e: React.KeyboardEvent, sessions: CopilotSession[]) => {
    if (!sessions.length) return;
    
    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault();
        setFocusedSessionIndex(prev => Math.min(prev + 1, sessions.length - 1));
        break;
      case 'ArrowUp':
        e.preventDefault();
        setFocusedSessionIndex(prev => Math.max(prev - 1, 0));
        break;
      case 'Enter':
      case ' ':
        e.preventDefault();
        if (focusedSessionIndex >= 0 && focusedSessionIndex < sessions.length) {
          handleSelectSession(sessions[focusedSessionIndex]);
        }
        break;
      case 'Escape':
        e.preventDefault();
        setFocusedSessionIndex(-1);
        inputRef.current?.focus();
        break;
    }
  }, [focusedSessionIndex, handleSelectSession]);

  // Render tool call with status icon
  const renderToolCall = (tool: ToolCall, isActive: boolean = false) => {
    const StatusIcon = {
      pending: ClockIcon,
      completed: CheckCircleFillIcon,
      failed: XCircleFillIcon,
    }[tool.status];

    const statusColor = {
      pending: 'var(--fgColor-attention)',
      completed: 'var(--fgColor-success)',
      failed: 'var(--fgColor-danger)',
    }[tool.status];

    return (
      <div 
        key={tool.id} 
        className="flex items-center gap-1 py-0.5"
        role="status"
        aria-label={`Tool ${tool.name} ${tool.status}`}
      >
        {isActive && tool.status === 'pending' ? (
          <Spinner size="small" />
        ) : (
          <StatusIcon size={12} fill={statusColor} />
        )}
        <Text style={{ fontSize: '0.75rem', color: statusColor }}>
          {tool.name}
        </Text>
      </div>
    );
  };

  // Loading state
  if (statusLoading) {
    return (
      <div 
        className="flex justify-center items-center"
        style={{ height: '60vh' }}
        role="status"
        aria-label="Loading Copilot"
      >
        <Spinner size="large" />
      </div>
    );
  }

  // Check if authentication is required but not configured/authenticated
  // This handles 401 errors from the API when auth is needed
  const isAuthError = statusError && 
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    ((statusError as any)?.response?.status === 401 || (statusError as any)?.status === 401);

  if (isAuthError) {
    return (
      <div className="p-4">
        <PageHeader
          title="Migration Copilot"
          description="AI-powered migration planning and execution"
        />
        <div className="mt-4" style={{ maxWidth: 600 }}>
          <Flash variant="warning">
            <div className="flex items-start gap-3">
              <ShieldLockIcon size={24} className="flex-shrink-0 mt-0.5" />
              <div>
                <Text className="font-bold block">Authentication Required</Text>
                <Text className="block mt-1">
                  Migration Copilot requires authentication to access your GitHub account and perform migrations.
                </Text>
                {authEnabled ? (
                  <div className="mt-3">
                    <Button onClick={login} variant="primary">
                      Sign in with GitHub
                    </Button>
                  </div>
                ) : (
                  <Text className="block mt-2 text-sm" style={{ color: 'var(--fgColor-muted)' }}>
                    Authentication is not enabled in this deployment. To use Migration Copilot, 
                    please configure GitHub OAuth authentication in the{' '}
                    <Link to="/settings" className="underline" style={{ color: 'var(--fgColor-accent)' }}>
                      <GearIcon size={12} className="inline mr-1" />
                      Settings
                    </Link>.
                  </Text>
                )}
              </div>
            </div>
          </Flash>
        </div>
      </div>
    );
  }

  // General error state (non-auth errors)
  if (statusError) {
    return (
      <div className="p-4">
        <PageHeader
          title="Migration Copilot"
          description="AI-powered migration planning and execution"
        />
        <div className="mt-4" style={{ maxWidth: 600 }}>
          <Flash variant="danger">
            Failed to load Copilot status. Please try again later.
          </Flash>
        </div>
      </div>
    );
  }

  // Unavailable state
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
    <div 
      className="flex"
      style={{ 
        height: 'calc(100vh - 120px)',
        position: 'relative',
      }}
    >
      {/* Mobile overlay */}
      {isMobile && sidebarOpen && (
        <div
          className="fixed inset-0"
          style={{ 
            backgroundColor: 'rgba(0, 0, 0, 0.5)',
            zIndex: 10 
          }}
          onClick={() => setSidebarOpen(false)}
          role="presentation"
        />
      )}

      {/* Sidebar - Session list */}
      <aside
        className="flex flex-col"
        style={{
          width: sidebarOpen ? (isMobile ? '85%' : 280) : 0,
          maxWidth: isMobile ? 320 : 280,
          borderRight: sidebarOpen ? '1px solid var(--borderColor-default)' : 'none',
          backgroundColor: 'var(--bgColor-muted)',
          overflow: 'hidden',
          transition: 'width 0.2s ease-in-out',
          position: isMobile ? 'fixed' : 'relative',
          left: 0,
          top: isMobile ? 60 : 0,
          bottom: 0,
          zIndex: isMobile ? 20 : 1,
        }}
        role="complementary"
        aria-label="Chat history sidebar"
      >
        <div className="p-3" style={{ borderBottom: '1px solid var(--borderColor-default)' }}>
          <Button 
            onClick={handleNewChat} 
            leadingVisual={PlusIcon} 
            className="w-full"
            aria-label="Start new chat"
          >
            New Chat
          </Button>
        </div>
        <div 
          ref={sessionListRef}
          className="flex-1 overflow-y-auto p-2"
          role="listbox"
          aria-label="Previous chats"
          tabIndex={0}
          onKeyDown={(e: React.KeyboardEvent<HTMLDivElement>) => handleSessionKeyDown(e, sessions)}
        >
          {sessions.length === 0 ? (
            <Text className="text-center block mt-4" style={{ color: 'var(--fgColor-muted)', fontSize: '0.875rem' }}>
              No previous chats
            </Text>
          ) : (
            sessions.map((session, index) => (
              <div
                key={session.id}
                onClick={() => handleSelectSession(session)}
                onKeyDown={(e: React.KeyboardEvent<HTMLDivElement>) => {
                  if (e.key === 'Enter' || e.key === ' ') {
                    e.preventDefault();
                    handleSelectSession(session);
                  }
                }}
                className="p-2 mb-1 rounded flex justify-between items-center"
                role="option"
                aria-selected={currentSessionId === session.id}
                tabIndex={focusedSessionIndex === index ? 0 : -1}
                style={{
                  cursor: 'pointer',
                  backgroundColor: currentSessionId === session.id 
                    ? 'var(--bgColor-accent-muted)' 
                    : focusedSessionIndex === index 
                    ? 'var(--bgColor-inset)' 
                    : 'transparent',
                  outline: focusedSessionIndex === index ? '2px solid var(--color-accent-emphasis)' : 'none',
                  outlineOffset: -2,
                }}
                onMouseEnter={(e) => {
                  if (currentSessionId !== session.id) {
                    e.currentTarget.style.backgroundColor = 'var(--bgColor-inset)';
                  }
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.backgroundColor = currentSessionId === session.id 
                    ? 'var(--bgColor-accent-muted)' 
                    : 'transparent';
                }}
              >
                <div className="flex-1 overflow-hidden">
                  <Text 
                    className="font-semibold block overflow-hidden text-ellipsis whitespace-nowrap" 
                    style={{ fontSize: '0.875rem' }}
                  >
                    {session.title || 'New Chat'}
                  </Text>
                  <Text style={{ color: 'var(--fgColor-muted)', fontSize: '0.75rem' }}>
                    {session.message_count} messages
                  </Text>
                </div>
                <IconButton
                  icon={TrashIcon}
                  aria-label={`Delete chat: ${session.title || 'New Chat'}`}
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
      </aside>

      {/* Main chat area */}
      <div className="flex-1 flex flex-col min-w-0">
        {/* Header */}
        <header className="p-3" style={{ borderBottom: '1px solid var(--borderColor-default)' }}>
          <div className="flex items-center gap-2">
            <IconButton
              icon={sidebarOpen ? SidebarCollapseIcon : SidebarExpandIcon}
              aria-label={sidebarOpen ? 'Hide sidebar' : 'Show sidebar'}
              variant="invisible"
              onClick={() => setSidebarOpen(!sidebarOpen)}
            />
            <CopilotIcon size={24} />
            <div>
              <Text className="font-bold" style={{ fontSize: '1rem' }}>Copilot Assistant</Text>
              <Text className="block" style={{ color: 'var(--fgColor-muted)', fontSize: '0.875rem' }}>
                AI-powered migration planning and execution
              </Text>
            </div>
          </div>
        </header>

        {/* Messages */}
        <div 
          ref={messagesContainerRef}
          className="flex-1 overflow-y-auto p-3"
          role="log"
          aria-label="Chat messages"
          aria-live="polite"
          aria-atomic="false"
        >
          {messages.length === 0 && !streamingContent ? (
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
                    onClick={() => {
                      setMessage(suggestion);
                      inputRef.current?.focus();
                    }}
                    className="border"
                    style={{ 
                      maxWidth: isMobile ? '100%' : 400, 
                      borderColor: 'var(--borderColor-default)',
                    }}
                  >
                    {suggestion}
                  </Button>
                ))}
              </div>
            </div>
          ) : (
            <>
              {messages.map((msg) => {
                const isError = msg.content.startsWith('Error:');
                
                return (
                  <article
                    key={msg.id}
                    className="flex gap-2 mb-3 group"
                    style={{
                      flexDirection: msg.role === 'user' ? 'row-reverse' : 'row',
                    }}
                    aria-label={`${msg.role === 'user' ? 'You' : 'Copilot'}`}
                  >
                    <div
                      className="flex items-center justify-center rounded-full"
                      style={{
                        width: 32,
                        height: 32,
                        flexShrink: 0,
                        backgroundColor: msg.role === 'user' 
                          ? 'var(--bgColor-accent-emphasis)' 
                          : isError 
                          ? 'var(--bgColor-danger-emphasis)'
                          : 'var(--bgColor-success-emphasis)',
                      }}
                    >
                      {msg.role === 'user' ? (
                        <PersonIcon size={16} fill="white" />
                      ) : (
                        <CopilotIcon size={16} fill="white" />
                      )}
                    </div>
                    <div
                      className="p-3 rounded relative"
                      style={{
                        maxWidth: isMobile ? '85%' : '70%',
                        backgroundColor: msg.role === 'user' 
                          ? 'var(--bgColor-accent-muted)' 
                          : isError
                          ? 'var(--bgColor-danger-muted)'
                          : 'var(--bgColor-muted)',
                      }}
                    >
                      <Text style={{ whiteSpace: 'pre-wrap' }}>{msg.content}</Text>
                      
                      {/* Tool calls */}
                      {msg.tool_calls && msg.tool_calls.length > 0 && (
                        <div className="mt-2 pt-2" style={{ borderTop: '1px solid var(--borderColor-default)' }}>
                          <div className="flex items-center gap-1 mb-1">
                            <ToolsIcon size={12} />
                            <Text className="font-bold" style={{ fontSize: '0.75rem', color: 'var(--fgColor-muted)' }}>
                              Tools Used:
                            </Text>
                          </div>
                          {msg.tool_calls.map((tool) => renderToolCall(tool))}
                        </div>
                      )}
                      
                      {/* Action buttons - show on hover */}
                      <div 
                        className="absolute top-1 right-1 flex gap-1 opacity-0 group-hover:opacity-100 transition-opacity"
                      >
                        {msg.role === 'assistant' && !isError && (
                          <IconButton
                            icon={copiedMessageId === msg.id ? CheckIcon : CopyIcon}
                            aria-label={copiedMessageId === msg.id ? 'Copied!' : 'Copy message'}
                            variant="invisible"
                            size="small"
                            onClick={() => handleCopyMessage(msg.id, msg.content)}
                          />
                        )}
                        {isError && failedMessageContent && (
                          <IconButton
                            icon={SyncIcon}
                            aria-label="Retry message"
                            variant="invisible"
                            size="small"
                            onClick={handleRetry}
                          />
                        )}
                      </div>
                    </div>
                  </article>
                );
              })}
              
              {/* Streaming response */}
              {isStreaming && (
                <div 
                  className="flex gap-2 mb-3"
                  role="status"
                  aria-label="Copilot is responding"
                >
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
                  <div 
                    className="p-3 rounded" 
                    style={{ 
                      maxWidth: isMobile ? '85%' : '70%', 
                      backgroundColor: 'var(--bgColor-muted)' 
                    }}
                  >
                    {streamingContent ? (
                      <Text style={{ whiteSpace: 'pre-wrap' }}>{streamingContent}</Text>
                    ) : (
                      <div className="flex items-center gap-2">
                        <Spinner size="small" />
                        <Text style={{ color: 'var(--fgColor-muted)', fontSize: '0.875rem' }}>
                          Thinking...
                        </Text>
                      </div>
                    )}
                    {activeToolCalls.length > 0 && (
                      <div className="mt-2 pt-2" style={{ borderTop: '1px solid var(--borderColor-default)' }}>
                        <div className="flex items-center gap-1 mb-1">
                          <ToolsIcon size={12} />
                          <Text className="font-bold" style={{ fontSize: '0.75rem', color: 'var(--fgColor-muted)' }}>
                            Tools in use:
                          </Text>
                        </div>
                        {activeToolCalls.map((tool) => renderToolCall(tool, true))}
                      </div>
                    )}
                  </div>
                </div>
              )}
            </>
          )}
        </div>

        {/* Input */}
        <footer className="p-3" style={{ borderTop: '1px solid var(--borderColor-default)' }}>
          <div className="flex gap-2">
            <TextInput
              ref={inputRef}
              value={message}
              onChange={(e) => setMessage(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder="Ask about repositories, dependencies, batches, migration planning..."
              className="flex-1"
              disabled={isStreaming}
              aria-label="Message input"
              aria-describedby="chat-input-hint"
            />
            <span id="chat-input-hint" className="sr-only">
              Press Enter to send, Shift+Enter for new line
            </span>
            {isStreaming ? (
              <Button
                onClick={handleStopStreaming}
                variant="danger"
                aria-label="Stop response"
              >
                <StopIcon />
              </Button>
            ) : (
              <Button
                onClick={handleSendStreaming}
                disabled={!message.trim()}
                variant="primary"
                aria-label="Send message"
              >
                <PaperAirplaneIcon />
              </Button>
            )}
          </div>
          {failedMessageContent && (
            <div className="mt-2">
              <Button 
                size="small" 
                leadingVisual={SyncIcon}
                onClick={handleRetry}
                aria-label="Retry last message"
              >
                Retry last message
              </Button>
            </div>
          )}
        </footer>
      </div>

      {/* Screen reader only CSS */}
      <style>{`
        .sr-only {
          position: absolute;
          width: 1px;
          height: 1px;
          padding: 0;
          margin: -1px;
          overflow: hidden;
          clip: rect(0, 0, 0, 0);
          white-space: nowrap;
          border: 0;
        }
      `}</style>
    </div>
  );
}

export default CopilotAssistant;
