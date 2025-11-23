import { Spinner } from '@primer/react';

export function LoadingSpinner() {
  return (
    <div className="flex justify-center items-center py-12">
      <Spinner size="large" />
    </div>
  );
}

