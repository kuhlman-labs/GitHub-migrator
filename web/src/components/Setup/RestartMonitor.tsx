import { useState, useEffect } from 'react';
import { Text, Spinner, Flash } from '@primer/react';
import { CheckIcon, AlertIcon } from '@primer/octicons-react';

interface RestartMonitorProps {
  onServerOnline: () => void;
}

export function RestartMonitor({ onServerOnline }: RestartMonitorProps) {
  const [status, setStatus] = useState<'restarting' | 'checking' | 'online' | 'manual'>('restarting');
  const [dots, setDots] = useState('');

  useEffect(() => {
    // Animate dots while waiting
    const dotsInterval = setInterval(() => {
      setDots((prev) => (prev.length >= 3 ? '' : prev + '.'));
    }, 500);

    // Wait 5 seconds before starting health checks (give server time to shutdown)
    const initialDelay = setTimeout(() => {
      setStatus('checking');
      checkHealth();
    }, 5000);

    return () => {
      clearInterval(dotsInterval);
      clearTimeout(initialDelay);
    };
  }, []);

  const checkHealth = async () => {
    let attempts = 0;
    const maxAttempts = 30; // 30 attempts * 2 seconds = 60 seconds max

    const interval = setInterval(async () => {
      try {
        const response = await fetch('/health');
        if (response.ok) {
          clearInterval(interval);
          setStatus('online');
          setTimeout(() => {
            onServerOnline();
          }, 1500);
        }
      } catch (error) {
        // Server not ready yet, keep trying
      }

      attempts++;
      if (attempts >= maxAttempts) {
        clearInterval(interval);
        setStatus('manual'); // Server didn't come back - needs manual restart
      }
    }, 2000);
  };

  return (
    <div
      className="fixed inset-0 flex items-center justify-center z-50"
      style={{ backgroundColor: 'rgba(0, 0, 0, 0.7)' }}
    >
      <div
        className="p-8 rounded-lg border min-w-96 text-center"
        style={{
          backgroundColor: 'var(--bgColor-default)',
          borderColor: 'var(--borderColor-default)',
        }}
      >
        {status === 'restarting' && (
          <>
            <Spinner size="large" className="mb-4" />
            <Text className="text-2xl font-bold block mb-3">
              Server Restarting{dots}
            </Text>
            <Text style={{ color: 'var(--fgColor-muted)' }}>
              Please wait while the server restarts with new configuration
            </Text>
          </>
        )}

        {status === 'checking' && (
          <>
            <Spinner size="large" className="mb-4" />
            <Text className="text-2xl font-bold block mb-3">
              Checking Server Status{dots}
            </Text>
            <Text style={{ color: 'var(--fgColor-muted)' }}>
              Waiting for server to come back online
            </Text>
          </>
        )}

        {status === 'online' && (
          <>
            <div
              className="w-16 h-16 rounded-full flex items-center justify-center mx-auto mb-4"
              style={{ backgroundColor: 'var(--bgColor-success-emphasis)' }}
            >
              <CheckIcon size={32} fill="white" />
            </div>
            <Text className="text-2xl font-bold block mb-3">
              Server Online!
            </Text>
            <Flash variant="success" className="text-left">
              Configuration has been applied successfully. Redirecting to dashboard...
            </Flash>
          </>
        )}

        {status === 'manual' && (
          <>
            <div className="mb-4" style={{ color: 'var(--fgColor-attention)' }}>
              <AlertIcon size={48} />
            </div>
            <Text className="text-2xl font-bold block mb-3">
              Server Restart Required
            </Text>
            <Flash variant="warning" className="text-left mb-4">
              Configuration has been applied successfully, but the server needs to be restarted manually.
            </Flash>
            <div className="text-left p-4 rounded" style={{ backgroundColor: 'var(--bgColor-muted)' }}>
              <Text className="font-bold mb-2 block">To restart the server:</Text>
              <code className="block p-2 rounded text-sm" style={{ backgroundColor: 'var(--bgColor-inset)' }}>
                make run-server
              </code>
              <Text className="text-xs mt-3" style={{ color: 'var(--fgColor-muted)' }}>
                In production with Docker/systemd, the server will restart automatically.
              </Text>
            </div>
          </>
        )}
      </div>
    </div>
  );
}
