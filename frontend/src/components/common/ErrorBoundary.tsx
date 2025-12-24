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
                    padding: '24px',
                    background: 'var(--bg-secondary, #1a1a2e)',
                    border: '1px solid var(--border-subtle, #333)',
                    borderRadius: '8px',
                    color: 'var(--text-primary, #fff)',
                    fontFamily: 'system-ui, -apple-system, sans-serif',
                    maxWidth: '800px',
                    margin: '40px auto',
                    boxShadow: '0 4px 20px rgba(0,0,0,0.3)'
                }}>
                    <h3 style={{
                        margin: '0 0 16px 0',
                        color: '#ff7b72', // GitHub Error Red
                        fontSize: '1.25rem',
                        display: 'flex',
                        alignItems: 'center',
                        gap: '8px'
                    }}>
                        <span style={{ fontSize: '1.5rem' }}>⚠️</span>
                        Application Error
                    </h3>

                    <div style={{
                        background: 'rgba(255,123,114,0.1)',
                        border: '1px solid rgba(255,123,114,0.2)',
                        borderRadius: '6px',
                        padding: '16px',
                        marginBottom: '16px'
                    }}>
                        <p style={{
                            margin: '0',
                            fontSize: '1rem',
                            fontWeight: 500,
                            color: 'var(--text-primary, #e6edf3)'
                        }}>
                            {this.state.error?.message || 'An unexpected error occurred'}
                        </p>
                    </div>

                    <details style={{ marginBottom: '20px' }}>
                        <summary style={{
                            cursor: 'pointer',
                            color: 'var(--text-secondary, #8b949e)',
                            fontSize: '0.9rem',
                            userSelect: 'none'
                        }}>
                            View Stack Trace
                        </summary>
                        <pre style={{
                            marginTop: '10px',
                            padding: '12px',
                            background: '#0d1117',
                            border: '1px solid #30363d',
                            borderRadius: '6px',
                            color: '#c9d1d9',
                            fontSize: '0.75rem',
                            overflowX: 'auto',
                            lineHeight: '1.45',
                            whiteSpace: 'pre-wrap'
                        }}>
                            {this.state.error?.stack}
                        </pre>
                    </details>

                    <button
                        onClick={() => this.setState({ hasError: false, error: null })}
                        style={{
                            padding: '8px 16px',
                            background: 'var(--accent-primary, #3b82f6)',
                            color: 'white',
                            border: 'none',
                            borderRadius: '6px',
                            cursor: 'pointer',
                            fontSize: '0.9rem',
                            fontWeight: 600,
                            transition: 'background 0.2s'
                        }}
                    >
                        Try Again
                    </button>

                    <div style={{ marginTop: '16px', fontSize: '0.8rem', color: 'var(--text-tertiary, #484f58)' }}>
                        Tip: Open the Developer Tools console for full logs.
                    </div>
                </div>
            );
        }

        return this.props.children;
    }
}

export default ErrorBoundary;
