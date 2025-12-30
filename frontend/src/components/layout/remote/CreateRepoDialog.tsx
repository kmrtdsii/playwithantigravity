import React, { useRef, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button } from '../../common/Button';
import { FolderPlus } from 'lucide-react';
import { useRemoteClone } from '../../../hooks/useRemoteClone';
import CloneProgress from './CloneProgress';

interface CreateRepoDialogProps {
    isOpen: boolean;
    onClose: () => void;
    onSuccess?: () => void;
    existingRemotes?: Array<{ name: string, url: string }>;
}

const CreateRepoDialog: React.FC<CreateRepoDialogProps> = ({ isOpen, onClose, onSuccess, existingRemotes = [] }) => {
    const { t } = useTranslation('common');
    const dialogRef = useRef<HTMLDialogElement>(null);
    const [repoName, setRepoName] = useState('');
    const [error, setError] = useState<string | null>(null);

    // Use the hook for logic
    const {
        cloneStatus,
        performCreate,
        cancelClone,
        errorMessage,
        elapsedSeconds
    } = useRemoteClone();

    // Check availability
    useEffect(() => {
        // Skip validation if we are currently creating or have completed creation
        if (cloneStatus === 'creating' || cloneStatus === 'complete') {
            return;
        }

        if (!repoName.trim()) {
            setError(null);
            return;
        }

        // Check if name implies a URL we already have? 
        // For creation, we usually just need to check if the generated name "my-project" conflicts with a local remote name.
        // Or if we are creating a repo with name X, and we already have a remote named X.
        const isDuplicateLocalName = existingRemotes.some(r => r.name === repoName);
        if (isDuplicateLocalName) {
            setError('既に同じ名前のリモートリポジトリが存在します。');
            return;
        }

        setError(null);
    }, [repoName, existingRemotes, cloneStatus]);

    // Dialog Control
    useEffect(() => {
        const dialog = dialogRef.current;
        if (!dialog) return;

        if (isOpen) {
            dialog.showModal();
            setRepoName('');
            setError(null);
            cancelClone(); // Reset any previous state
        } else {
            dialog.close();
        }
    }, [isOpen, cancelClone]);

    // ... (keep default close handlers)

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
        if (repoName.trim() && !error) {
            performCreate(repoName);
        }
    };

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
                maxWidth: '450px',
                boxShadow: '0 10px 25px rgba(0,0,0,0.5)',
                backdropFilter: 'blur(5px)',
                overflow: 'hidden'
            }}
            className="backdrop:bg-black/50 backdrop:backdrop-blur-sm"
        >
            <div style={{ padding: '20px' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px' }}>
                    <h2 style={{ fontSize: '18px', fontWeight: 600, display: 'flex', alignItems: 'center', gap: '8px', margin: 0 }}>
                        <FolderPlus size={20} className="text-accent-primary" />
                        {t('remote.empty.createTitle')}
                    </h2>
                </div>

                <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
                    <div>
                        <input
                            type="text"
                            value={repoName}
                            onChange={(e) => setRepoName(e.target.value)}
                            placeholder={t('remote.empty.createPlaceholder') || 'my-awesome-project'}
                            disabled={cloneStatus === 'creating' || cloneStatus === 'complete'}
                            style={{
                                width: '100%',
                                padding: '10px 12px',
                                background: 'var(--bg-tertiary)',
                                border: '1px solid var(--border-subtle)', // Should we change border color on error?
                                borderColor: error ? 'var(--text-danger)' : 'var(--border-subtle)',
                                borderRadius: 'var(--radius-md)',
                                color: 'var(--text-primary)',
                                fontSize: '14px',
                                outline: 'none'
                            }}
                            autoFocus
                        />
                        {error ? (
                            <div style={{ fontSize: '12px', color: 'var(--text-danger)', marginTop: '4px' }}>
                                {t('remote.empty.duplicateName')}
                            </div>
                        ) : (
                            <div style={{ fontSize: '12px', color: 'var(--text-tertiary)', marginTop: '4px' }}>
                                {t('remote.empty.validation')}
                            </div>
                        )}
                    </div>

                    {(cloneStatus === 'creating' || cloneStatus === 'error' || cloneStatus === 'complete') && (
                        <CloneProgress
                            status={cloneStatus}
                            elapsedSeconds={elapsedSeconds}
                            errorMessage={errorMessage}
                            onRetry={() => performCreate(repoName)}
                            onCancel={onClose}
                            successMessage={t('remote.status.created') || '作成しました'}
                            hideCancelButton
                            customErrorTitle={t('remote.status.createFailed')}
                        />
                    )}

                    <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '8px', marginTop: '10px' }}>
                        <Button
                            type="button"
                            variant="secondary"
                            onClick={onClose}
                            disabled={cloneStatus === 'creating'}
                        >
                            {t('remote.cancel')}
                        </Button>
                        <Button
                            type="submit"
                            variant="primary"
                            disabled={!repoName.trim() || !!error || cloneStatus === 'creating' || cloneStatus === 'complete'}
                        >
                            {cloneStatus === 'creating' ? t('remote.empty.creating') : t('remote.empty.createButton')}
                        </Button>
                    </div>
                </form>
            </div>
        </dialog>
    );
};

export default CreateRepoDialog;
