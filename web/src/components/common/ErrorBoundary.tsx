import { Component, ReactNode } from 'react';
import { Blankslate } from '@primer/react/experimental';
import { AlertIcon } from '@primer/octicons-react';

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
}

interface State {
  hasError: boolean;
  error?: Error;
}

export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = { hasError: false };
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    console.error('Error caught by boundary:', error, errorInfo);
  }

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) {
        return this.props.fallback;
      }

      return (
        <Blankslate>
          <Blankslate.Visual>
            <AlertIcon size={48} className="text-gh-danger-fg" />
          </Blankslate.Visual>
          <Blankslate.Heading>Something went wrong</Blankslate.Heading>
          <Blankslate.Description>
            {this.state.error?.message || 'An unexpected error occurred. Please try refreshing the page.'}
          </Blankslate.Description>
          <Blankslate.PrimaryAction
            onClick={() => this.setState({ hasError: false, error: undefined })}
          >
            Try Again
          </Blankslate.PrimaryAction>
          <button
            onClick={() => window.location.reload()}
            className="px-4 py-2 text-sm font-medium text-gh-text-secondary hover:text-gh-text-primary transition-colors"
          >
            Refresh Page
          </button>
        </Blankslate>
      );
    }

    return this.props.children;
  }
}

