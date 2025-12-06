/* eslint-disable react-refresh/only-export-components */
import { createContext, useContext, useState, ReactNode, useCallback } from 'react';
import { Flash } from '@primer/react';
import { XIcon } from '@primer/octicons-react';

type ToastVariant = 'default' | 'success' | 'warning' | 'danger';

interface Toast {
  id: string;
  message: string;
  variant: ToastVariant;
}

interface ToastContextType {
  showToast: (message: string, variant?: ToastVariant) => void;
  showSuccess: (message: string) => void;
  showError: (message: string) => void;
  showWarning: (message: string) => void;
}

const ToastContext = createContext<ToastContextType | undefined>(undefined);

export function useToast() {
  const context = useContext(ToastContext);
  if (!context) {
    throw new Error('useToast must be used within ToastProvider');
  }
  return context;
}

interface ToastProviderProps {
  children: ReactNode;
}

export function ToastProvider({ children }: ToastProviderProps) {
  const [toasts, setToasts] = useState<Toast[]>([]);

  const showToast = useCallback((message: string, variant: ToastVariant = 'default') => {
    const id = Math.random().toString(36).substring(7);
    const newToast: Toast = { id, message, variant };
    
    setToasts((prev) => [...prev, newToast]);
    
    // Auto-dismiss after 5 seconds
    setTimeout(() => {
      setToasts((prev) => prev.filter((t) => t.id !== id));
    }, 5000);
  }, []);

  const showSuccess = useCallback((message: string) => {
    showToast(message, 'success');
  }, [showToast]);

  const showError = useCallback((message: string) => {
    showToast(message, 'danger');
  }, [showToast]);

  const showWarning = useCallback((message: string) => {
    showToast(message, 'warning');
  }, [showToast]);

  const dismissToast = (id: string) => {
    setToasts((prev) => prev.filter((t) => t.id !== id));
  };

  return (
    <ToastContext.Provider value={{ showToast, showSuccess, showError, showWarning }}>
      {children}
      
      {/* Toast Container */}
      <div 
        className="fixed top-4 right-4 z-50 flex flex-col gap-2"
        style={{ maxWidth: '400px' }}
        role="region"
        aria-label="Notifications"
        aria-live="polite"
      >
        {toasts.map((toast) => (
          <Flash key={toast.id} variant={toast.variant} className="shadow-lg">
            <div className="flex items-start justify-between gap-2">
              <div className="flex-1">{toast.message}</div>
              <button
                onClick={() => dismissToast(toast.id)}
                className="flex-shrink-0 p-1 hover:bg-black hover:bg-opacity-10 rounded transition-colors"
                aria-label="Dismiss notification"
              >
                <XIcon size={16} />
              </button>
            </div>
          </Flash>
        ))}
      </div>
    </ToastContext.Provider>
  );
}

