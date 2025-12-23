import React from 'react';
import type { CloneStatus } from './CloneProgress';
import { actionButtonStyle } from './remoteStyles';

interface EmptyStateProps {
    isEditMode: boolean;
    cloneStatus?: CloneStatus;
    onConnect: () => void;
}

const EmptyState: React.FC<EmptyStateProps> = ({ isEditMode, cloneStatus, onConnect }) => (
    <div style={{
        height: '100%',
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        color: 'var(--text-tertiary)',
        gap: '12px',
        padding: '20px',
        textAlign: 'center'
    }}>
        {!isEditMode && cloneStatus === 'idle' && (
            <>
                <div style={{ fontSize: '24px', opacity: 0.3 }}>🌐</div>
                <div style={{ fontSize: '0.85rem' }}>No Remote Configured</div>
                <button
                    onClick={onConnect}
                    style={{
                        ...actionButtonStyle,
                        background: 'var(--bg-tertiary)',
                        color: 'var(--text-primary)',
                        border: '1px solid var(--border-subtle)',
                        fontSize: '14px',
                        padding: '10px 20px',
                        marginTop: '10px'
                    }}
                >
                    Connect Repository
                </button>
            </>
        )}
        {(cloneStatus === 'fetching_info' || cloneStatus === 'cloning') && (
            <div style={{ fontSize: '0.85rem', color: 'var(--text-secondary)' }}>
                Connecting to repository...
            </div>
        )}
    </div>
);

export default EmptyState;
