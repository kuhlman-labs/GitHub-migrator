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

  componentDidCatch(error: Error, errorInfo: any) {
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
          <Blankslate.SecondaryAction
            onClick={() => window.location.reload()}
          >
            Refresh Page
          </Blankslate.SecondaryAction>
        </Blankslate>
      );
    }

    return this.props.children;
  }
}

