import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useDialogState, useMultiDialogState } from './useDialogState';

describe('useDialogState', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('should initialize with closed state', () => {
    const { result } = renderHook(() => useDialogState());

    expect(result.current.isOpen).toBe(false);
    expect(result.current.data).toBeNull();
  });

  it('should open dialog without data', () => {
    const { result } = renderHook(() => useDialogState());

    act(() => {
      result.current.open();
    });

    expect(result.current.isOpen).toBe(true);
    expect(result.current.data).toBeNull();
  });

  it('should open dialog with data', () => {
    const { result } = renderHook(() => useDialogState<{ id: number }>());

    act(() => {
      result.current.open({ id: 123 });
    });

    expect(result.current.isOpen).toBe(true);
    expect(result.current.data).toEqual({ id: 123 });
  });

  it('should close dialog', () => {
    const { result } = renderHook(() => useDialogState());

    act(() => {
      result.current.open();
    });

    expect(result.current.isOpen).toBe(true);

    act(() => {
      result.current.close();
    });

    expect(result.current.isOpen).toBe(false);
  });

  it('should toggle dialog state', () => {
    const { result } = renderHook(() => useDialogState());

    expect(result.current.isOpen).toBe(false);

    act(() => {
      result.current.toggle();
    });

    expect(result.current.isOpen).toBe(true);

    act(() => {
      result.current.toggle();
    });

    expect(result.current.isOpen).toBe(false);
  });

  it('should preserve data when closing', () => {
    const { result } = renderHook(() => useDialogState<{ id: number }>());

    act(() => {
      result.current.open({ id: 123 });
    });

    expect(result.current.data).toEqual({ id: 123 });

    act(() => {
      result.current.close();
    });

    // Data is preserved after close (for animation purposes)
    expect(result.current.data).toEqual({ id: 123 });
  });

  it('should update data when opening with new data', () => {
    const { result } = renderHook(() => useDialogState<{ id: number }>());

    act(() => {
      result.current.open({ id: 1 });
    });

    expect(result.current.data).toEqual({ id: 1 });

    act(() => {
      result.current.open({ id: 2 });
    });

    expect(result.current.data).toEqual({ id: 2 });
  });

  it('should have a returnFocusRef', () => {
    const { result } = renderHook(() => useDialogState());

    expect(result.current.returnFocusRef).toBeDefined();
    expect(result.current.returnFocusRef.current).toBeNull();
  });
});

describe('useMultiDialogState', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('should initialize with all dialogs closed', () => {
    const { result } = renderHook(() =>
      useMultiDialogState({
        delete: null,
        edit: null,
      })
    );

    expect(result.current.isOpen('delete')).toBe(false);
    expect(result.current.isOpen('edit')).toBe(false);
  });

  it('should open a specific dialog', () => {
    const { result } = renderHook(() =>
      useMultiDialogState({
        delete: null,
        edit: null,
      })
    );

    act(() => {
      result.current.open('delete');
    });

    expect(result.current.isOpen('delete')).toBe(true);
    expect(result.current.isOpen('edit')).toBe(false);
  });

  it('should open dialog with data', () => {
    const { result } = renderHook(() =>
      useMultiDialogState<{
        delete: { id: number };
        edit: { name: string };
      }>({
        delete: null,
        edit: null,
      })
    );

    act(() => {
      result.current.open('delete', { id: 123 });
    });

    expect(result.current.isOpen('delete')).toBe(true);
    expect(result.current.data('delete')).toEqual({ id: 123 });
  });

  it('should close a specific dialog', () => {
    const { result } = renderHook(() =>
      useMultiDialogState({
        delete: null,
        edit: null,
      })
    );

    act(() => {
      result.current.open('delete');
      result.current.open('edit');
    });

    expect(result.current.isOpen('delete')).toBe(true);
    expect(result.current.isOpen('edit')).toBe(true);

    act(() => {
      result.current.close('delete');
    });

    expect(result.current.isOpen('delete')).toBe(false);
    expect(result.current.isOpen('edit')).toBe(true);
  });

  it('should close all dialogs', () => {
    const { result } = renderHook(() =>
      useMultiDialogState({
        delete: null,
        edit: null,
        create: null,
      })
    );

    act(() => {
      result.current.open('delete');
      result.current.open('edit');
      result.current.open('create');
    });

    expect(result.current.isOpen('delete')).toBe(true);
    expect(result.current.isOpen('edit')).toBe(true);
    expect(result.current.isOpen('create')).toBe(true);

    act(() => {
      result.current.closeAll();
    });

    expect(result.current.isOpen('delete')).toBe(false);
    expect(result.current.isOpen('edit')).toBe(false);
    expect(result.current.isOpen('create')).toBe(false);
  });

  it('should return null for dialog data when not set', () => {
    const { result } = renderHook(() =>
      useMultiDialogState<{
        delete: { id: number };
      }>({
        delete: null,
      })
    );

    expect(result.current.data('delete')).toBeNull();
  });

  it('should allow multiple dialogs open simultaneously', () => {
    const { result } = renderHook(() =>
      useMultiDialogState({
        dialog1: null,
        dialog2: null,
        dialog3: null,
      })
    );

    act(() => {
      result.current.open('dialog1');
      result.current.open('dialog2');
    });

    expect(result.current.isOpen('dialog1')).toBe(true);
    expect(result.current.isOpen('dialog2')).toBe(true);
    expect(result.current.isOpen('dialog3')).toBe(false);
  });

  it('should clear data after close with delay', () => {
    const { result } = renderHook(() =>
      useMultiDialogState<{
        edit: { id: number };
      }>({
        edit: null,
      })
    );

    act(() => {
      result.current.open('edit', { id: 123 });
    });

    expect(result.current.data('edit')).toEqual({ id: 123 });

    act(() => {
      result.current.close('edit');
    });

    // Data is still present immediately after close
    expect(result.current.data('edit')).toEqual({ id: 123 });

    // Advance timers to clear data
    act(() => {
      vi.advanceTimersByTime(350);
    });

    expect(result.current.data('edit')).toBeNull();
  });
});

