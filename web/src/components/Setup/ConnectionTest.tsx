import { useState } from 'react';
import { Button, Flash, Spinner, Text } from '@primer/react';
import { CheckIcon, XIcon, AlertIcon } from '@primer/octicons-react';
import type { ValidationResult } from '../../types';

interface ConnectionTestProps {
  onTest: () => Promise<ValidationResult>;
  label?: string;
  disabled?: boolean;
}

export function ConnectionTest({ onTest, label = 'Test Connection', disabled = false }: ConnectionTestProps) {
  const [testing, setTesting] = useState(false);
  const [result, setResult] = useState<ValidationResult | null>(null);

  const handleTest = async () => {
    setTesting(true);
    setResult(null);
    try {
      const validationResult = await onTest();
      setResult(validationResult);
    } catch (error) {
      setResult({
        valid: false,
        error: error instanceof Error ? error.message : 'Connection test failed',
      });
    } finally {
      setTesting(false);
    }
  };

  return (
    <div className="mt-4">
      <Button onClick={handleTest} disabled={disabled || testing} variant="default">
        {testing ? (
          <>
            <Spinner size="small" className="mr-2" />
            Testing...
          </>
        ) : (
          label
        )}
      </Button>

      {result && (
        <div className="mt-4">
          {result.valid ? (
            <div 
              className="flex items-start rounded-lg border p-4"
              style={{ 
                backgroundColor: 'var(--bgColor-success-muted)',
                borderColor: 'var(--borderColor-success-muted)'
              }}
            >
              <div style={{ color: 'var(--fgColor-success)' }}>
                <CheckIcon size={16} />
              </div>
              <div className="ml-3 flex-1">
                <Text className="font-bold">Connection successful!</Text>
                {result.details && Object.keys(result.details).length > 0 && (
                  <div className="mt-2 text-xs">
                    {Object.entries(result.details).map(([key, value]) => (
                      <Text key={key} className="block" style={{ color: 'var(--fgColor-muted)' }}>
                        {key}: {String(value)}
                      </Text>
                    ))}
                  </div>
                )}
              </div>
            </div>
          ) : (
            <Flash variant="danger" className="flex items-start">
              <XIcon size={16} />
              <div className="ml-3 flex-1">
                <Text className="font-bold">Connection failed</Text>
                {result.error && (
                  <Text className="block mt-2 text-xs">{result.error}</Text>
                )}
              </div>
            </Flash>
          )}

          {result.warnings && result.warnings.length > 0 && (
            <Flash variant="warning" className="flex items-start mt-3">
              <AlertIcon size={16} />
              <div className="ml-3 flex-1">
                <Text className="font-bold">Warnings</Text>
                {result.warnings.map((warning, index) => (
                  <Text key={index} className="block mt-2 text-xs">
                    {warning}
                  </Text>
                ))}
              </div>
            </Flash>
          )}
        </div>
      )}
    </div>
  );
}
