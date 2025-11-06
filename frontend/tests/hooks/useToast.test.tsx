import { describe, it, expect, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useToast, toast } from '@/hooks/use-toast';

describe('useToast Hook', () => {
  beforeEach(() => {
    // Clear all toasts before each test
    const { result } = renderHook(() => useToast());
    act(() => {
      result.current.toasts.forEach((t) => {
        result.current.dismiss(t.id);
      });
    });
  });

  it('should initialize with empty toasts array', () => {
    const { result } = renderHook(() => useToast());
    expect(result.current.toasts).toEqual([]);
  });

  it('should add a toast', () => {
    const { result } = renderHook(() => useToast());

    act(() => {
      result.current.toast({
        title: 'Test Toast',
        description: 'This is a test',
      });
    });

    expect(result.current.toasts).toHaveLength(1);
    expect(result.current.toasts[0].title).toBe('Test Toast');
    expect(result.current.toasts[0].description).toBe('This is a test');
  });

  it('should limit toasts to TOAST_LIMIT (1)', () => {
    const { result } = renderHook(() => useToast());

    act(() => {
      result.current.toast({ title: 'Toast 1' });
      result.current.toast({ title: 'Toast 2' });
      result.current.toast({ title: 'Toast 3' });
    });

    // Only the most recent toast should remain
    expect(result.current.toasts).toHaveLength(1);
    expect(result.current.toasts[0].title).toBe('Toast 3');
  });

  it('should dismiss a specific toast', () => {
    const { result } = renderHook(() => useToast());

    let toastId: string;
    act(() => {
      const toast = result.current.toast({ title: 'Test Toast' });
      toastId = toast.id;
    });

    expect(result.current.toasts).toHaveLength(1);

    act(() => {
      result.current.dismiss(toastId!);
    });

    // Toast should be marked as closed (open: false)
    expect(result.current.toasts[0].open).toBe(false);
  });

  it('should dismiss all toasts when no ID provided', () => {
    const { result } = renderHook(() => useToast());

    act(() => {
      result.current.toast({ title: 'Toast 1' });
    });

    expect(result.current.toasts).toHaveLength(1);

    act(() => {
      result.current.dismiss();
    });

    // All toasts should be marked as closed
    result.current.toasts.forEach((toast) => {
      expect(toast.open).toBe(false);
    });
  });

  it('should generate unique IDs for each toast', () => {
    const { result } = renderHook(() => useToast());

    let id1: string;
    let id2: string;

    act(() => {
      const toast1 = result.current.toast({ title: 'Toast 1' });
      id1 = toast1.id;
    });

    act(() => {
      const toast2 = result.current.toast({ title: 'Toast 2' });
      id2 = toast2.id;
    });

    expect(id1).not.toBe(id2);
  });

  it('should return toast controller with dismiss and update methods', () => {
    const { result } = renderHook(() => useToast());

    let toastController: ReturnType<typeof toast>;
    act(() => {
      toastController = result.current.toast({ title: 'Test Toast' });
    });

    expect(toastController!).toHaveProperty('id');
    expect(toastController!).toHaveProperty('dismiss');
    expect(toastController!).toHaveProperty('update');
    expect(typeof toastController!.dismiss).toBe('function');
    expect(typeof toastController!.update).toBe('function');
  });

  it('should update toast content', () => {
    const { result } = renderHook(() => useToast());

    let toastController: ReturnType<typeof toast>;
    act(() => {
      toastController = result.current.toast({ 
        title: 'Original Title',
        description: 'Original Description' 
      });
    });

    act(() => {
      toastController!.update({ 
        title: 'Updated Title',
        description: 'Updated Description' 
      });
    });

    expect(result.current.toasts[0].title).toBe('Updated Title');
    expect(result.current.toasts[0].description).toBe('Updated Description');
  });

  it('should handle toast variants', () => {
    const { result } = renderHook(() => useToast());

    act(() => {
      result.current.toast({
        title: 'Success',
        variant: 'default',
      });
    });

    expect(result.current.toasts[0].variant).toBe('default');
  });

  it('should set toast as open by default', () => {
    const { result } = renderHook(() => useToast());

    act(() => {
      result.current.toast({ title: 'Test Toast' });
    });

    expect(result.current.toasts[0].open).toBe(true);
  });
});

describe('toast standalone function', () => {
  it('should create toast without hook', () => {
    const toastController = toast({ title: 'Standalone Toast' });

    expect(toastController).toHaveProperty('id');
    expect(toastController).toHaveProperty('dismiss');
    expect(toastController).toHaveProperty('update');
  });

  it('should dismiss toast via controller', () => {
    const { result } = renderHook(() => useToast());

    let toastController: ReturnType<typeof toast>;
    act(() => {
      toastController = toast({ title: 'Test Toast' });
    });

    act(() => {
      toastController!.dismiss();
    });

    // Verify toast was dismissed in the hook
    const dismissedToast = result.current.toasts.find(
      (t) => t.id === toastController!.id
    );
    expect(dismissedToast?.open).toBe(false);
  });
});
