import { Text, Heading, Flash, Button } from '@primer/react';
import { CheckCircleIcon } from '@primer/octicons-react';

export function RestartInstructions() {
  return (
    <div
      className="fixed inset-0 flex items-center justify-center z-50"
      style={{ backgroundColor: 'rgba(0, 0, 0, 0.7)' }}
    >
      <div
        className="p-8 rounded-lg border max-w-2xl"
        style={{
          backgroundColor: 'var(--bgColor-default)',
          borderColor: 'var(--borderColor-default)',
        }}
      >
        <div className="text-center mb-6">
          <div
            className="w-16 h-16 rounded-full flex items-center justify-center mx-auto mb-4"
            style={{ backgroundColor: 'var(--bgColor-success-emphasis)' }}
          >
            <CheckCircleIcon size={32} fill="white" />
          </div>
          <Heading as="h2" className="text-2xl mb-3">
            Configuration Saved Successfully!
          </Heading>
        </div>

        <Flash variant="success" className="mb-6">
          Your configuration has been saved to the <code>.env</code> file and the setup is marked as complete.
        </Flash>

        <div
          className="p-4 rounded-lg border mb-6"
          style={{
            backgroundColor: 'var(--bgColor-muted)',
            borderColor: 'var(--borderColor-default)',
          }}
        >
          <Heading as="h3" className="text-lg mb-3">
            Next Steps
          </Heading>
          <Text className="block mb-4">
            To apply the configuration changes, please restart the server:
          </Text>

          <div className="mb-4">
            <Text className="font-bold block mb-2">Development:</Text>
            <code
              className="block p-3 rounded text-sm"
              style={{
                backgroundColor: 'var(--bgColor-inset)',
                fontFamily: 'monospace',
              }}
            >
              make run-server
            </code>
          </div>

          <div className="mb-4">
            <Text className="font-bold block mb-2">Docker:</Text>
            <code
              className="block p-3 rounded text-sm"
              style={{
                backgroundColor: 'var(--bgColor-inset)',
                fontFamily: 'monospace',
              }}
            >
              docker-compose restart
            </code>
          </div>

          <div>
            <Text className="font-bold block mb-2">Production (systemd):</Text>
            <code
              className="block p-3 rounded text-sm"
              style={{
                backgroundColor: 'var(--bgColor-inset)',
                fontFamily: 'monospace',
              }}
            >
              systemctl restart github-migrator
            </code>
          </div>
        </div>

        <Text className="text-center" style={{ color: 'var(--fgColor-muted)' }}>
          Once the server restarts, you'll be able to access the dashboard.
        </Text>

        <div className="text-center mt-6">
          <Button
            onClick={() => window.location.href = '/'}
            variant="primary"
          >
            Go to Dashboard
          </Button>
        </div>
      </div>
    </div>
  );
}

