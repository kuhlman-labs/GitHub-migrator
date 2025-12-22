import { Spinner } from '@primer/react';

export function LoadingSpinner() {
  return (
    <div className="flex justify-center items-center py-12" role="status" aria-live="polite" data-testid="loading-spinner">
      <Spinner size="large" aria-label="Loading content" />
    </div>
  );
}

