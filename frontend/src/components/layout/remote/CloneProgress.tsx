import React from 'react';
import { Loader2, AlertCircle, CheckCircle, XCircle, RefreshCw } from 'lucide-react';

export type CloneStatus = 'idle' | 'fetching_info' | 'cloning' | 'complete' | 'error';

interface CloneProgressProps {
    status: CloneStatus;
    estimatedSeconds?: number;
    elapsedSeconds: number;
    repoInfo?: {
        name: string;
        sizeDisplay: string;
        message: string;
    };
    errorMessage?: string;
    onRetry?: () => void;
    onCancel?: () => void;
}

/**
 * CloneProgress - Displays progress during repository clone operations.
 * Shows estimated time, progress bar, and error handling UI.
 */
const CloneProgress: React.FC<CloneProgressProps> = ({
    status,
    estimatedSeconds = 0,
    elapsedSeconds,
    repoInfo,
    errorMessage,
    onRetry,
    onCancel,
}) => {
    if (status === 'idle') return null;

    const progress = estimatedSeconds > 0
        ? Math.min(100, (elapsedSeconds / estimatedSeconds) * 100)
        : 0;

    const formatTime = (seconds: number): string => {
        if (seconds < 60) return `${Math.round(seconds)}s`;
        const mins = Math.floor(seconds / 60);
        const secs = Math.round(seconds % 60);
        return `${mins}m ${secs}s`;
    };

    const remainingSeconds = Math.max(0, estimatedSeconds - elapsedSeconds);

    return (
        <div style={containerStyle}>
            {/* Status Icon */}
            <div style={iconContainerStyle}>
                {status === 'fetching_info' && (
                    <Loader2 size={20} style={{ animation: 'spin 1s linear infinite' }} />
                )}
                {status === 'cloning' && (
                    <Loader2 size={20} style={{ animation: 'spin 1s linear infinite' }} />
                )}
                {status === 'complete' && (
                    <CheckCircle size={20} color="var(--color-success)" />
                )}
                {status === 'error' && (
                    <AlertCircle size={20} color="var(--color-error)" />
                )}
            </div>

            {/* Content */}
            <div style={contentStyle}>
                {/* Status Text */}
                <div style={statusTextStyle}>
                    {status === 'fetching_info' && 'Fetching repository info...'}
                    {status === 'cloning' && (
                        <>
                            Cloning repository...
                            {repoInfo && (
                                <span style={infoTextStyle}> ({repoInfo.sizeDisplay})</span>
                            )}
                        </>
                    )}
                    {status === 'complete' && 'Clone complete!'}
                    {status === 'error' && 'Clone failed'}
                </div>

                {/* Progress Bar (only when cloning) */}
                {status === 'cloning' && estimatedSeconds > 0 && (
                    <div style={progressContainerStyle}>
                        <div style={progressBarStyle}>
                            <div
                                style={{
                                    ...progressFillStyle,
                                    width: `${progress}%`,
                                }}
                            />
                        </div>
                        <div style={timeTextStyle}>
                            {remainingSeconds > 0
                                ? `~${formatTime(remainingSeconds)} remaining`
                                : 'Almost done...'
                            }
                        </div>
                    </div>
                )}

                {/* Error Message */}
                {status === 'error' && errorMessage && (
                    <div style={errorMessageStyle}>
                        {errorMessage}
                    </div>
                )}

                {/* Action Buttons */}
                {status === 'error' && (
                    <div style={buttonContainerStyle}>
                        {onRetry && (
                            <button onClick={onRetry} style={retryButtonStyle}>
                                <RefreshCw size={14} />
                                Retry
                            </button>
                        )}
                        {onCancel && (
                            <button onClick={onCancel} style={cancelButtonStyle}>
                                <XCircle size={14} />
                                Cancel
                            </button>
                        )}
                    </div>
                )}
            </div>
        </div>
    );
};

// --- Styles ---

const containerStyle: React.CSSProperties = {
    display: 'flex',
    alignItems: 'flex-start',
    gap: '12px',
    padding: '12px 16px',
    background: 'var(--bg-tertiary)',
    borderRadius: '8px',
    marginTop: '12px',
};

const iconContainerStyle: React.CSSProperties = {
    color: 'var(--accent-primary)',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    flexShrink: 0,
};

const contentStyle: React.CSSProperties = {
    flex: 1,
    minWidth: 0,
};

const statusTextStyle: React.CSSProperties = {
    fontSize: '13px',
    fontWeight: 500,
    color: 'var(--text-primary)',
};

const infoTextStyle: React.CSSProperties = {
    color: 'var(--text-tertiary)',
    fontWeight: 400,
};

const progressContainerStyle: React.CSSProperties = {
    marginTop: '8px',
};

const progressBarStyle: React.CSSProperties = {
    height: '6px',
    background: 'var(--bg-secondary)',
    borderRadius: '3px',
    overflow: 'hidden',
};

const progressFillStyle: React.CSSProperties = {
    height: '100%',
    background: 'var(--accent-primary)',
    borderRadius: '3px',
    transition: 'width 0.3s ease-out',
};

const timeTextStyle: React.CSSProperties = {
    fontSize: '11px',
    color: 'var(--text-tertiary)',
    marginTop: '4px',
};

const errorMessageStyle: React.CSSProperties = {
    fontSize: '12px',
    color: '#d32f2f', // Red-700 for visibility
    fontWeight: 500,
    marginTop: '6px',
    lineHeight: 1.4,
    wordBreak: 'break-word',
    whiteSpace: 'pre-wrap',
};

const buttonContainerStyle: React.CSSProperties = {
    display: 'flex',
    gap: '8px',
    marginTop: '10px',
};

const retryButtonStyle: React.CSSProperties = {
    display: 'flex',
    alignItems: 'center',
    gap: '4px',
    padding: '6px 12px',
    fontSize: '12px',
    fontWeight: 500,
    background: 'var(--accent-primary)',
    color: 'white',
    border: 'none',
    borderRadius: '4px',
    cursor: 'pointer',
};

const cancelButtonStyle: React.CSSProperties = {
    display: 'flex',
    alignItems: 'center',
    gap: '4px',
    padding: '6px 12px',
    fontSize: '12px',
    fontWeight: 500,
    background: 'transparent',
    color: 'var(--text-secondary)',
    border: '1px solid var(--border-subtle)',
    borderRadius: '4px',
    cursor: 'pointer',
};

export default CloneProgress;
