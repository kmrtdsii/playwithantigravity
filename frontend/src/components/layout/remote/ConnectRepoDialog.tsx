import React, { useRef, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button } from '../../common/Button';
import { Link2, X } from 'lucide-react';
import { useRemoteClone } from '../../../hooks/useRemoteClone';
import CloneProgress from './CloneProgress';

interface ConnectRepoDialogProps {
    isOpen: boolean;
    onClose: () => void;
    onSuccess?: () => void;
}

const ConnectRepoDialog: React.FC<ConnectRepoDialogProps> = ({ isOpen, onClose, onSuccess }) => {
    const { t } = useTranslation('common');
    const dialogRef = useRef<HTMLDialogElement>(null);
    const [repoUrl, setRepoUrl] = useState('');

    // Use the hook for logic
    const {
        cloneStatus,
        performClone,
        cancelClone,
        errorMessage,
        elapsedSeconds,
        estimatedSeconds,
        repoInfo
    } = useRemoteClone();

    // Dialog Control
    useEffect(() => {
        const dialog = dialogRef.current;
        if (!dialog) return;

        if (isOpen) {
            dialog.showModal();
            setRepoUrl('');
            cancelClone(); // Reset any previous state
        } else {
            dialog.close();
        }
    }, [isOpen, cancelClone]);

    // Handle ESC key or click outside to close
    useEffect(() => {
        const dialog = dialogRef.current;
        if (!dialog) return;

        const handleCancel = (e: Event) => {
            e.preventDefault();
            if (cloneStatus !== 'cloning' && cloneStatus !== 'fetching_info') {
                onClose();
            }
        };

        const handleClick = (e: MouseEvent) => {
            if (e.target === dialog && cloneStatus !== 'cloning' && cloneStatus !== 'fetching_info') {
                onClose();
            }
        };

        dialog.addEventListener('cancel', handleCancel);
        dialog.addEventListener('click', handleClick);

        return () => {
            dialog.removeEventListener('cancel', handleCancel);
            dialog.removeEventListener('click', handleClick);
        };
    }, [onClose, cloneStatus]);

    // Success Handler
    useEffect(() => {
        if (cloneStatus === 'complete') {
            const timer = setTimeout(() => {
                onClose();
                if (onSuccess) onSuccess();
            }, 1000);
            return () => clearTimeout(timer);
        }
    }, [cloneStatus, onClose, onSuccess]);

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault();
        if (repoUrl.trim()) {
            performClone(repoUrl);
        }
    };

    const isCloning = cloneStatus === 'cloning' || cloneStatus === 'fetching_info';

    return (
        <dialog
            ref={dialogRef}
            style={{
                background: 'var(--bg-secondary)',
                color: 'var(--text-primary)',
                border: '1px solid var(--border-subtle)',
                borderRadius: 'var(--radius-lg)',
                padding: 0,
                width: '100%',
                maxWidth: '500px',
                boxShadow: '0 10px 25px rgba(0,0,0,0.5)',
                backdropFilter: 'blur(5px)'
            }}
            className="backdrop:bg-black/50 backdrop:backdrop-blur-sm"
        >
            <div style={{ padding: '20px' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px' }}>
                    <h2 style={{ fontSize: '18px', fontWeight: 600, display: 'flex', alignItems: 'center', gap: '8px', margin: 0 }}>
                        <Link2 size={20} className="text-accent-primary" />
                        {t('remote.empty.connect')}
                    </h2>
                    <button
                        onClick={onClose}
                        disabled={isCloning}
                        style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'var(--text-tertiary)' }}
                    >
                        <X size={20} />
                    </button>
                </div>

                <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
                    <div>
                        <input
                            type="text"
                            value={repoUrl}
                            onChange={(e) => setRepoUrl(e.target.value)}
                            placeholder="https://github.com/owner/repo.git or git@..."
                            disabled={isCloning || cloneStatus === 'complete'}
                            style={{
                                width: '100%',
                                padding: '10px 12px',
                                background: 'var(--bg-tertiary)',
                                border: '1px solid var(--border-subtle)',
                                borderRadius: 'var(--radius-md)',
                                color: 'var(--text-primary)',
                                fontSize: '14px',
                                outline: 'none'
                            }}
                            autoFocus
                        />
                    </div>

                    {(cloneStatus !== 'idle') && (
                        <CloneProgress
                            status={cloneStatus}
                            elapsedSeconds={elapsedSeconds}
                            estimatedSeconds={estimatedSeconds}
                            repoInfo={repoInfo}
                            errorMessage={errorMessage}
                            onRetry={() => performClone(repoUrl)}
                            onCancel={cancelClone}
                        />
                    )}

                    <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '8px', marginTop: '10px' }}>
                        <Button
                            type="button"
                            variant="secondary"
                            onClick={onClose}
                            disabled={isCloning}
                        >
                            {t('remote.cancel')}
                        </Button>
                        <Button
                            type="submit"
                            variant="primary"
                            disabled={!repoUrl.trim() || isCloning || cloneStatus === 'complete'}
                        >
                            {isCloning ? t('remote.status.connecting') : t('remote.empty.connectButton')}
                        </Button>
                    </div>
                </form>
            </div>
        </dialog>
    );
};

export default ConnectRepoDialog;
