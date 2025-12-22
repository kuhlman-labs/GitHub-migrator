import { useState, useCallback, useRef } from 'react';

interface DialogState<T = undefined> {
  /** Whether the dialog is open */
  isOpen: boolean;
  /** Data associated with the dialog (e.g., item being edited/deleted) */
  data: T | null;
}

interface UseDialogStateReturn<T = undefined> {
  /** Whether the dialog is open */
  isOpen: boolean;
  /** Data associated with the dialog */
  data: T | null;
  /** Open the dialog, optionally with data */
  open: (data?: T) => void;
  /** Close the dialog */
  close: () => void;
  /** Toggle the dialog state */
  toggle: () => void;
  /** Ref to return focus to when dialog closes */
  returnFocusRef: React.RefObject<HTMLElement | null>;
}

/**
 * A hook for managing dialog/modal state with optional data payload.
 * 
 * @example
 * ```tsx
 * // Simple dialog (no data)
 * const confirmDialog = useDialogState();
 * 
 * // Dialog with data (e.g., editing an item)
 * const editDialog = useDialogState<User>();
 * 
 * // Usage
 * <Button onClick={editDialog.open.bind(null, user)}>Edit</Button>
 * 
 * <ConfirmationDialog
 *   isOpen={confirmDialog.isOpen}
 *   onCancel={confirmDialog.close}
 *   onConfirm={() => {
 *     handleDelete(editDialog.data);
 *     editDialog.close();
 *   }}
 *   returnFocusRef={confirmDialog.returnFocusRef}
 * />
 * ```
 */
export function useDialogState<T = undefined>(): UseDialogStateReturn<T> {
  const [state, setState] = useState<DialogState<T>>({
    isOpen: false,
    data: null,
  });

  const returnFocusRef = useRef<HTMLElement>(null);

  const open = useCallback((data?: T) => {
    // Store the currently focused element for returning focus later
    if (document.activeElement instanceof HTMLElement) {
      returnFocusRef.current = document.activeElement;
    }
    setState({
      isOpen: true,
      data: data ?? null,
    });
  }, []);

  const close = useCallback(() => {
    setState((prev) => ({
      ...prev,
      isOpen: false,
    }));
    // Return focus after a short delay to allow dialog to close
    setTimeout(() => {
      if (returnFocusRef.current) {
        returnFocusRef.current.focus();
      }
    }, 0);
  }, []);

  const toggle = useCallback(() => {
    setState((prev) => ({
      ...prev,
      isOpen: !prev.isOpen,
    }));
  }, []);

  return {
    isOpen: state.isOpen,
    data: state.data,
    open,
    close,
    toggle,
    returnFocusRef,
  };
}

/**
 * A hook for managing multiple dialogs in a component.
 * Useful when you have several confirmation dialogs, edit dialogs, etc.
 * 
 * @example
 * ```tsx
 * const dialogs = useMultiDialogState({
 *   delete: false,
 *   edit: false,
 *   create: false,
 * });
 * 
 * dialogs.open('delete', item);
 * dialogs.isOpen('delete'); // true
 * dialogs.close('delete');
 * ```
 */
interface MultiDialogState<TDialogs extends Record<string, unknown>> {
  isOpen: (dialog: keyof TDialogs) => boolean;
  data: <K extends keyof TDialogs>(dialog: K) => TDialogs[K] | null;
  open: <K extends keyof TDialogs>(dialog: K, data?: TDialogs[K]) => void;
  close: (dialog: keyof TDialogs) => void;
  closeAll: () => void;
}

export function useMultiDialogState<TDialogs extends Record<string, unknown>>(
  dialogNames: { [K in keyof TDialogs]: null }
): MultiDialogState<TDialogs> {
  const [openDialogs, setOpenDialogs] = useState<Set<keyof TDialogs>>(new Set());
  const [dialogData, setDialogData] = useState<Partial<TDialogs>>({});

  const isOpen = useCallback((dialog: keyof TDialogs): boolean => {
    return openDialogs.has(dialog);
  }, [openDialogs]);

  const data = useCallback(<K extends keyof TDialogs>(dialog: K): TDialogs[K] | null => {
    return (dialogData[dialog] ?? null) as TDialogs[K] | null;
  }, [dialogData]);

  const open = useCallback(<K extends keyof TDialogs>(dialog: K, newData?: TDialogs[K]) => {
    setOpenDialogs((prev) => new Set([...prev, dialog]));
    if (newData !== undefined) {
      setDialogData((prev) => ({ ...prev, [dialog]: newData }));
    }
  }, []);

  const close = useCallback((dialog: keyof TDialogs) => {
    setOpenDialogs((prev) => {
      const next = new Set(prev);
      next.delete(dialog);
      return next;
    });
    // Clear data after a delay to prevent UI flicker
    setTimeout(() => {
      setDialogData((prev) => {
        const next = { ...prev };
        delete next[dialog];
        return next;
      });
    }, 300);
  }, []);

  const closeAll = useCallback(() => {
    setOpenDialogs(new Set());
    setTimeout(() => {
      setDialogData({});
    }, 300);
  }, []);

  // Validate dialog names on mount (development only)
  if (process.env.NODE_ENV === 'development') {
    const validNames = Object.keys(dialogNames);
    for (const dialog of openDialogs) {
      if (!validNames.includes(String(dialog))) {
        console.warn(`useMultiDialogState: Unknown dialog name "${String(dialog)}"`);
      }
    }
  }

  return {
    isOpen,
    data,
    open,
    close,
    closeAll,
  };
}

export type { DialogState, UseDialogStateReturn, MultiDialogState };

