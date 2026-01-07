import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Heading, Spinner } from '@primer/react';
import { MarkGithubIcon } from '@primer/octicons-react';
import { SetupWizard } from './SetupWizard';
import { useSetupStatus } from '../../hooks/useQueries';

const SETUP_COMPLETED_KEY = 'setup_completed_hint';
const POLLING_INTERVAL = 2000; // Poll every 2 seconds when waiting for backend

export function Setup() {
  const navigate = useNavigate();
  const [waitingForBackend, setWaitingForBackend] = useState(false);
  const [retryCount, setRetryCount] = useState(0);
  
  // Check if setup was previously completed (stored in localStorage as a hint)
  const wasSetupCompleted = localStorage.getItem(SETUP_COMPLETED_KEY) === 'true';
  
  // Use setup status query with polling when waiting for backend
  const { data: setupStatus, isError, refetch } = useSetupStatus();
  
  // If we know setup was completed before and we're getting errors, we're probably waiting for backend restart
  // This is a legitimate use of setState in effect to synchronize with external API state
  useEffect(() => {
    if (wasSetupCompleted && isError) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setWaitingForBackend(true);
    }
  }, [wasSetupCompleted, isError]);
  
  // Poll for backend reconnection when waiting
  useEffect(() => {
    if (!waitingForBackend) return;
    
    const intervalId = setInterval(async () => {
      try {
        setRetryCount(prev => prev + 1);
        const result = await refetch();
        
        // If we successfully fetched and setup is complete, redirect to dashboard
        if (result.data?.setup_completed) {
          localStorage.setItem(SETUP_COMPLETED_KEY, 'true');
          navigate('/', { replace: true });
        } else {
          // Backend is back but setup not complete, show wizard
          setWaitingForBackend(false);
        }
      } catch {
        // Still can't connect, keep polling
        console.debug('Waiting for backend to reconnect...');
      }
    }, POLLING_INTERVAL);
    
    return () => clearInterval(intervalId);
  }, [waitingForBackend, refetch, navigate]);
  
  // Store setup completion status in localStorage when we successfully load it
  useEffect(() => {
    if (setupStatus?.setup_completed) {
      localStorage.setItem(SETUP_COMPLETED_KEY, 'true');
    }
  }, [setupStatus]);
  
  // Show "waiting for backend" message if we think setup was completed
  if (waitingForBackend) {
    return (
      <div
        className="min-h-screen flex items-center justify-center"
        style={{ backgroundColor: 'var(--bgColor-inset)' }}
      >
        <div className="max-w-md mx-auto px-6 text-center">
          <div className="flex justify-center mb-6">
            <MarkGithubIcon size={64} />
          </div>
          <Heading as="h1" className="text-3xl mb-4">
            Connecting to Server
          </Heading>
          <Spinner size="large" className="mb-4" />
          <p className="text-sm mb-4" style={{ color: 'var(--fgColor-muted)' }}>
            Waiting for the backend server to become available...
          </p>
          <p className="text-xs" style={{ color: 'var(--fgColor-muted)' }}>
            Retry attempt: {retryCount}
          </p>
          <div 
            className="mt-6 text-left p-3 rounded-md border"
            style={{
              backgroundColor: 'var(--bgColor-accent-muted)',
              borderColor: 'var(--borderColor-accent-emphasis)',
            }}
          >
            <p className="text-sm">
              The server may be restarting. This page will automatically redirect once the connection is restored.
            </p>
          </div>
        </div>
      </div>
    );
  }
  
  return (
    <div
      className="min-h-screen py-8"
      style={{ backgroundColor: 'var(--bgColor-inset)' }}
    >
      <div className="max-w-5xl mx-auto px-6">
        {/* Header */}
        <div
          className="text-center mb-8 pb-6 border-b"
          style={{ borderColor: 'var(--borderColor-default)' }}
        >
          <div className="flex justify-center mb-4">
            <MarkGithubIcon size={64} />
          </div>
          <Heading as="h1" className="text-4xl mb-3">
            GitHub Migrator
          </Heading>
          <Heading
            as="h2"
            className="text-2xl font-normal"
            style={{ color: 'var(--fgColor-muted)' }}
          >
            Setup
          </Heading>
        </div>

        {/* Wizard */}
        <div
          className="rounded-lg border p-6"
          style={{
            backgroundColor: 'var(--bgColor-default)',
            borderColor: 'var(--borderColor-default)',
          }}
        >
          <SetupWizard />
        </div>
      </div>
    </div>
  );
}
