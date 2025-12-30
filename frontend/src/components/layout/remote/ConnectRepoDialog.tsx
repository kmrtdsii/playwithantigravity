import React, { useRef, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button } from '../../common/Button';
import { Link2 } from 'lucide-react';
import { useRemoteClone } from '../../../hooks/useRemoteClone';
import CloneProgress from './CloneProgress';

interface ConnectRepoDialogProps {
    isOpen: boolean;
    onClose: () => void;
    onSuccess?: () => void;
    existingRemotes?: Array<{ name: string, url: string }>;
}

const ConnectRepoDialog: React.FC<ConnectRepoDialogProps> = ({ isOpen, onClose, onSuccess, existingRemotes = [] }) => {
    const { t } = useTranslation('common');
    const dialogRef = useRef<HTMLDialogElement>(null);
    const inputRef = useRef<HTMLInputElement>(null);
    const [repoUrl, setRepoUrl] = useState('');
    const [error, setError] = useState<string | null>(null);

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

    // Check availability
    useEffect(() => {
        if (!repoUrl.trim()) {
            setError(null);
            return;
        }

        // Check if URL matches existing
        const input = repoUrl.trim();
        const isDuplicate = existingRemotes.some(r => r.url === input);

        if (isDuplicate) {
            setError('既に登録済みのリモートURLです。');
            return;
        }

        // Also check by name if possible? e.g. extraction. 
        // For now preventing strict URL duplication is enough as requested "same domain is duplicated".

        setError(null);
    }, [repoUrl, existingRemotes]);

    // Dialog Control
    useEffect(() => {
        const dialog = dialogRef.current;
        if (!dialog) return;

        if (isOpen) {
            dialog.showModal();
            setRepoUrl('');
            setError(null);
            cancelClone(); // Reset any previous state
            // Focus input after a short delay to ensure modal is open
            setTimeout(() => {
                inputRef.current?.focus();
            }, 50);
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
                    {/* Close button removed as requested */}
                </div>

                <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
                    <div>
                        <input
                            ref={inputRef}
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
                        {error && (
                            <div style={{ fontSize: '12px', color: 'var(--text-danger)', marginTop: '4px' }}>
                                {error}
                            </div>
                        )}
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
                            hideCancelButton={true}
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
                            disabled={!repoUrl.trim() || !!error || isCloning || cloneStatus === 'complete'}
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
