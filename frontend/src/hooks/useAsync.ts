import { useState, useCallback } from 'react';

interface AsyncState<T> {
    /** The data returned from the async operation */
    data: T | null;
    /** Whether the operation is in progress */
    isLoading: boolean;
    /** Error message if the operation failed */
    error: string | null;
}

interface UseAsyncReturn<T> extends AsyncState<T> {
    /** Execute the async function */
    execute: (...args: unknown[]) => Promise<T | undefined>;
    /** Clear the current error */
    clearError: () => void;
    /** Reset all state */
    reset: () => void;
}

/**
 * Custom hook for handling async operations with consistent error handling.
 * 
 * @param asyncFn - The async function to wrap
 * @returns Object with data, loading state, error, and control functions
 * 
 * @example
 * ```tsx
 * const { execute, isLoading, error, data } = useAsync(fetchData);
 * 
 * const handleClick = async () => {
 *   const result = await execute();
 *   if (result) {
 *     console.log('Success:', result);
 *   }
 * };
 * ```
 */
export function useAsync<T>(
    asyncFn: (...args: unknown[]) => Promise<T>
): UseAsyncReturn<T> {
    const [state, setState] = useState<AsyncState<T>>({
        data: null,
        isLoading: false,
        error: null,
    });

    const execute = useCallback(async (...args: unknown[]): Promise<T | undefined> => {
        setState(prev => ({ ...prev, isLoading: true, error: null }));

        try {
            const result = await asyncFn(...args);
            setState({ data: result, isLoading: false, error: null });
            return result;
        } catch (err) {
            const errorMessage = err instanceof Error ? err.message : 'An error occurred';
            setState(prev => ({ ...prev, isLoading: false, error: errorMessage }));
            console.error('useAsync error:', err);
            return undefined;
        }
    }, [asyncFn]);

    const clearError = useCallback(() => {
        setState(prev => ({ ...prev, error: null }));
    }, []);

    const reset = useCallback(() => {
        setState({ data: null, isLoading: false, error: null });
    }, []);

    return {
        ...state,
        execute,
        clearError,
        reset,
    };
}

/**
 * Simplified hook for async operations that don't need return data.
 * Useful for fire-and-forget operations like form submissions.
 */
export function useAsyncAction(
    asyncFn: (...args: unknown[]) => Promise<void>
): {
    execute: (...args: unknown[]) => Promise<boolean>;
    isLoading: boolean;
    error: string | null;
    clearError: () => void;
} {
    const { execute: baseExecute, isLoading, error, clearError } = useAsync(asyncFn);

    const execute = useCallback(async (...args: unknown[]): Promise<boolean> => {
        const result = await baseExecute(...args);
        return result !== undefined;
    }, [baseExecute]);

    return { execute, isLoading, error, clearError };
}
