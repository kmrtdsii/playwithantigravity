import React from 'react';
import { Loader2, AlertCircle, CheckCircle, RefreshCw, XCircle } from 'lucide-react';
import { Button } from '../../common/Button';

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

    // Calculation logic
    const progress = estimatedSeconds > 0
        ? Math.min(100, (elapsedSeconds / estimatedSeconds) * 100)
        : 0;

    const remainingSeconds = Math.max(0, estimatedSeconds - elapsedSeconds);
    const formatTime = (seconds: number) => {
        if (seconds < 60) return `${Math.round(seconds)}s`;
        const mins = Math.floor(seconds / 60);
        const secs = Math.round(seconds % 60);
        return `${mins}m ${secs}s`;
    };

    return (
        <div style={{
            display: 'flex',
            alignItems: 'flex-start',
            gap: '12px',
            padding: '12px 16px',
            background: 'var(--bg-tertiary)',
            borderRadius: 'var(--radius-md)',
            marginTop: '12px',
            border: '1px solid var(--border-subtle)'
        }}>
            {/* Icon State */}
            <div style={{ color: 'var(--accent-primary)', flexShrink: 0, marginTop: '2px' }}>
                {(status === 'fetching_info' || status === 'cloning') && (
                    <Loader2 size={20} className="animate-spin" style={{ animation: 'spin 1s linear infinite' }} />
                )}
                {status === 'complete' && <CheckCircle size={20} color="var(--color-success)" />}
                {status === 'error' && <AlertCircle size={20} color="var(--color-error)" />}
            </div>

            {/* Content Body */}
            <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ fontSize: '13px', fontWeight: 500, color: 'var(--text-primary)' }}>
                    {status === 'fetching_info' && 'Connecting to remote...'}
                    {status === 'cloning' && (
                        <span>
                            Syncing remote repository...
                            {repoInfo && <span style={{ color: 'var(--text-tertiary)', fontWeight: 400 }}> ({repoInfo.sizeDisplay})</span>}
                        </span>
                    )}
                    {status === 'complete' && 'Remote synced!'}
                    {status === 'error' && 'Sync failed'}
                </div>

                {/* Progress Bar */}
                {status === 'cloning' && estimatedSeconds > 0 && (
                    <div style={{ marginTop: '8px' }}>
                        <div style={{ height: '6px', background: 'var(--bg-secondary)', borderRadius: '3px', overflow: 'hidden' }}>
                            <div style={{
                                height: '100%',
                                background: 'var(--accent-primary)',
                                borderRadius: '3px',
                                width: `${progress}%`,
                                transition: 'width 0.3s ease-out'
                            }} />
                        </div>
                        <div style={{ fontSize: '11px', color: 'var(--text-tertiary)', marginTop: '4px' }}>
                            {remainingSeconds > 0 ? `~${formatTime(remainingSeconds)} remaining` : 'Almost done...'}
                        </div>
                    </div>
                )}

                {/* Error Box */}
                {status === 'error' && errorMessage && (
                    <div style={{
                        fontSize: '12px',
                        color: 'var(--color-error)',
                        marginTop: '6px',
                        lineHeight: 1.4,
                        whiteSpace: 'pre-wrap'
                    }}>
                        {errorMessage}
                    </div>
                )}

                {/* Actions */}
                {status === 'error' && (
                    <div style={{ display: 'flex', gap: '8px', marginTop: '10px' }}>
                        {onRetry && (
                            <Button size="sm" variant="primary" onClick={onRetry}>
                                <RefreshCw size={14} /> Retry
                            </Button>
                        )}
                        {onCancel && (
                            <Button size="sm" variant="secondary" onClick={onCancel}>
                                <XCircle size={14} /> Cancel
                            </Button>
                        )}
                    </div>
                )}
            </div>
        </div>
    );
};

export default CloneProgress;
