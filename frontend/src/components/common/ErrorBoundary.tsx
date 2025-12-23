import { Component } from 'react';
import type { ErrorInfo, ReactNode } from 'react';

interface ErrorBoundaryProps {
    children: ReactNode;
    /** Fallback UI to show when an error occurs */
    fallback?: ReactNode;
    /** Called when an error is caught */
    onError?: (error: Error, errorInfo: ErrorInfo) => void;
}

interface ErrorBoundaryState {
    hasError: boolean;
    error: Error | null;
}

/**
 * React Error Boundary component.
 * 
 * Catches JavaScript errors in child components and displays
 * a fallback UI instead of crashing the entire application.
 * 
 * @example
 * ```tsx
 * <ErrorBoundary fallback={<div>Something went wrong</div>}>
 *   <MyComponent />
 * </ErrorBoundary>
 * ```
 */
class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
    constructor(props: ErrorBoundaryProps) {
        super(props);
        this.state = { hasError: false, error: null };
    }

    static getDerivedStateFromError(error: Error): ErrorBoundaryState {
        return { hasError: true, error };
    }

    componentDidCatch(error: Error, errorInfo: ErrorInfo): void {
        console.error('ErrorBoundary caught an error:', error, errorInfo);
        this.props.onError?.(error, errorInfo);
    }

    render(): ReactNode {
        if (this.state.hasError) {
            if (this.props.fallback) {
                return this.props.fallback;
            }

            return (
                <div style={{
                    padding: '20px',
                    background: 'var(--bg-secondary, #1a1a2e)',
                    border: '1px solid var(--border-subtle, #333)',
                    borderRadius: '8px',
                    color: 'var(--text-primary, #fff)',
                    fontFamily: 'system-ui, sans-serif'
                }}>
                    <h3 style={{
                        margin: '0 0 12px 0',
                        color: '#ff7b72',
                        fontSize: '1rem'
                    }}>
                        Something went wrong
                    </h3>
                    <p style={{
                        margin: '0 0 12px 0',
                        fontSize: '0.875rem',
                        color: 'var(--text-secondary, #aaa)'
                    }}>
                        {this.state.error?.message || 'An unexpected error occurred'}
                    </p>
                    <button
                        onClick={() => this.setState({ hasError: false, error: null })}
                        style={{
                            padding: '6px 12px',
                            background: 'var(--accent-primary, #3b82f6)',
                            color: 'white',
                            border: 'none',
                            borderRadius: '4px',
                            cursor: 'pointer',
                            fontSize: '0.8rem'
                        }}
                    >
                        Try Again
                    </button>
                </div>
            );
        }

        return this.props.children;
    }
}

export default ErrorBoundary;
