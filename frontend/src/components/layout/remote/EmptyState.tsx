import React, { useState } from 'react';
import type { CloneStatus } from './CloneProgress';
import CreateRepoDialog from './CreateRepoDialog';
import ConnectRepoDialog from './ConnectRepoDialog';
import Modal from '../../common/Modal';
import { Button } from '../../common/Button';
import { Plus, Link2, Trash2, AlertTriangle } from 'lucide-react';
import { useTranslation } from 'react-i18next';

interface EmptyStateProps {
    isEditMode?: boolean;
    cloneStatus?: CloneStatus;
    onConnect?: () => void;
    recentRemotes?: Array<{ name: string, url: string }>;
    onSelectRemote?: (name: string) => void;
    onDeleteRemote?: (name: string) => void;
}

const EmptyState: React.FC<EmptyStateProps> = ({
    cloneStatus,
    recentRemotes = [],
    onSelectRemote,
    onDeleteRemote
}) => {
    const { t } = useTranslation('common');
    const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false);
    const [isConnectDialogOpen, setIsConnectDialogOpen] = useState(false);
    const [deleteTarget, setDeleteTarget] = useState<string | null>(null);

    // Style constants
    const containerStyle: React.CSSProperties = {
        height: '100%',
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        padding: '30px 20px',
        color: 'var(--text-tertiary)',
        background: 'var(--bg-primary)',
        overflowY: 'auto',
        overflowX: 'hidden'
    };

    const textStyle: React.CSSProperties = {
        fontSize: '0.9rem',
        color: 'var(--text-secondary)',
        marginBottom: '20px'
    };

    const actionButtonStyle: React.CSSProperties = {
        width: '100%',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        gap: '8px',
        padding: '10px 16px',
        borderRadius: '4px',
        fontSize: '13px',
        fontWeight: 600,
        cursor: 'pointer',
        transition: 'all 0.2s ease',
        height: '40px'
    };

    const separatorStyle: React.CSSProperties = {
        display: 'flex',
        alignItems: 'center',
        gap: '12px',
        width: '100%',
        fontSize: '12px',
        color: 'var(--text-tertiary)'
    };

    const lineStyle: React.CSSProperties = {
        flex: 1,
        height: '1px',
        background: 'var(--border-subtle)'
    };

    const handleDeleteClick = (name: string) => {
        setDeleteTarget(name);
    };

    const confirmDelete = () => {
        if (deleteTarget) {
            onDeleteRemote?.(deleteTarget);
            setDeleteTarget(null);
        }
    };

    return (
        <div style={containerStyle}>
            {cloneStatus === 'idle' && (
                <>
                    <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', maxWidth: '400px', width: '100%', marginBottom: '40px' }}>
                        <div style={textStyle}>
                            {t('remote.empty.description')}
                        </div>

                        <div style={{ display: 'flex', flexDirection: 'column', gap: '16px', width: '100%', maxWidth: '300px' }}>
                            {/* Connect Button */}
                            <button
                                onClick={() => setIsConnectDialogOpen(true)}
                                style={{
                                    ...actionButtonStyle,
                                    background: 'var(--bg-tertiary)',
                                    color: 'var(--text-primary)',
                                    border: '1px solid var(--border-subtle)',
                                }}
                            >
                                <Link2 size={16} />
                                {t('remote.empty.connect')}
                            </button>

                            {/* Separator */}
                            <div style={separatorStyle}>
                                <div style={lineStyle} />
                                <span>{t('remote.empty.or')}</span>
                                <div style={lineStyle} />
                            </div>

                            {/* Create Button */}
                            <button
                                onClick={() => setIsCreateDialogOpen(true)}
                                style={{
                                    ...actionButtonStyle,
                                    background: 'var(--accent-primary)',
                                    color: '#ffffff',
                                    border: 'none',
                                }}
                            >
                                <Plus size={16} />
                                {t('remote.empty.create')}
                            </button>
                        </div>
                    </div>

                    {/* Recent Remotes List */}
                    {recentRemotes.length > 0 && (
                        <div style={{ width: '100%', maxWidth: '300px', marginTop: '20px' }}>
                            <div style={{
                                fontSize: '12px',
                                color: 'var(--text-tertiary)',
                                marginBottom: '12px',
                                display: 'flex',
                                alignItems: 'center',
                                gap: '8px'
                            }}>
                                <span>{t('remote.empty.recentTitle')}</span>
                                <div style={lineStyle} />
                            </div>

                            <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                                {recentRemotes.map(remote => (
                                    <div
                                        key={remote.name}
                                        onClick={() => onSelectRemote?.(remote.name)}
                                        style={{
                                            display: 'flex',
                                            alignItems: 'center',
                                            justifyContent: 'space-between',
                                            padding: '12px',
                                            background: 'var(--bg-secondary)',
                                            border: '1px solid var(--border-subtle)',
                                            borderRadius: '6px',
                                            cursor: 'pointer', // Make entire card clickable
                                        }}
                                        className="hover:border-[var(--accent-primary)] transition-colors"
                                    >
                                        <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'flex-start', gap: '2px', overflow: 'hidden', flex: 1, minWidth: 0 }}>
                                            <div style={{ display: 'flex', alignItems: 'center', gap: '8px', width: '100%' }}>
                                                <div style={{
                                                    width: '8px',
                                                    height: '8px',
                                                    borderRadius: '50%',
                                                    background: 'var(--text-tertiary)',
                                                    flexShrink: 0
                                                }} />
                                                <span style={{ fontSize: '14px', fontWeight: 600, color: 'var(--text-primary)', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                                                    {remote.name}
                                                </span>
                                            </div>
                                            {/* Show remote URL if available */}
                                            {remote.url && (
                                                <span style={{
                                                    fontSize: '11px',
                                                    color: 'var(--text-tertiary)',
                                                    marginLeft: '16px', // Align with text start
                                                    whiteSpace: 'nowrap',
                                                    overflow: 'hidden',
                                                    textOverflow: 'ellipsis',
                                                    maxWidth: '100%'
                                                }} title={remote.url}>
                                                    {remote.url}
                                                </span>
                                            )}
                                        </div>

                                        <button
                                            onClick={(e) => {
                                                e.stopPropagation(); // Stop click from triggering selection
                                                handleDeleteClick(remote.name);
                                            }}
                                            title={t('remote.list.delete')}
                                            style={{
                                                background: 'transparent',
                                                border: 'none',
                                                color: 'var(--text-tertiary)',
                                                cursor: 'pointer',
                                                padding: '8px', // Increased touch area
                                                display: 'flex', alignItems: 'center',
                                                borderRadius: '4px',
                                                marginLeft: '8px'
                                            }}
                                            className="hover:text-red-400 hover:bg-[var(--bg-button-hover)]"
                                        >
                                            <Trash2 size={16} />
                                        </button>
                                    </div>
                                ))}
                            </div>
                        </div>
                    )}
                </>
            )}

            {(cloneStatus === 'fetching_info' || cloneStatus === 'cloning') && (
                <div style={{ fontSize: '0.85rem', color: 'var(--text-secondary)' }}>
                    {t('remote.empty.connecting')}
                </div>
            )}

            <CreateRepoDialog
                isOpen={isCreateDialogOpen}
                onClose={() => setIsCreateDialogOpen(false)}
                existingRemotes={recentRemotes}
            />
            <ConnectRepoDialog
                isOpen={isConnectDialogOpen}
                onClose={() => setIsConnectDialogOpen(false)}
                existingRemotes={recentRemotes}
            />

            {/* Delete Confirmation Modal */}
            {deleteTarget && (
                <Modal
                    isOpen={!!deleteTarget}
                    onClose={() => setDeleteTarget(null)}
                    title={t('remote.empty.deleteRemoteTitle')}
                    hideCloseButton
                >
                    <div style={{ display: 'flex', flexDirection: 'column', gap: '16px', padding: '8px 0' }}>
                        <div style={{ display: 'flex', gap: '12px', alignItems: 'start' }}>
                            <div style={{
                                padding: '8px',
                                background: 'rgba(239, 68, 68, 0.1)',
                                borderRadius: '50%',
                                color: 'var(--color-error)'
                            }}>
                                <AlertTriangle size={24} />
                            </div>
                            <div style={{ flex: 1 }}>
                                <div style={{ fontSize: '14px', lineHeight: '1.5', color: 'var(--text-primary)' }}>
                                    {t('remote.empty.deleteRemoteConfirm', { name: deleteTarget })}
                                </div>
                                <div style={{ marginTop: '8px', fontSize: '12px', color: 'var(--text-tertiary)' }}>
                                    {t('remote.empty.deleteRemoteDesc', { defaultValue: 'The actual repository and local files will be preserved.' })}
                                </div>
                            </div>
                        </div>

                        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '8px', marginTop: '12px' }}>
                            <Button
                                variant="secondary"
                                onClick={() => setDeleteTarget(null)}
                            >
                                {t('remote.cancel')}
                            </Button>
                            <Button
                                variant="primary" // Using primary with error color overriding
                                onClick={confirmDelete}
                                style={{
                                    background: 'var(--color-error)',
                                    borderColor: 'var(--color-error)',
                                    color: 'white'
                                }}
                            >
                                {t('remote.list.delete')}
                            </Button>
                        </div>
                    </div>
                </Modal>
            )}
        </div>
    );
};

export default EmptyState;
