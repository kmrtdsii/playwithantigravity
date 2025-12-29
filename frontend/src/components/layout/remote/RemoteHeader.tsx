import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Cloud, Database, Copy, Check, LogOut, Plus } from 'lucide-react';
import { headerStyle, inputStyle, cancelButtonStyle, submitButtonStyle } from './remoteStyles';

interface RemoteHeaderProps {
    remoteUrl: string;
    projectName: string;
    isEditMode: boolean;
    isSettingUp: boolean;
    setupUrl: string;
    onSetupUrlChange: (url: string) => void;
    onEditRemote: () => void;
    onDisconnect?: () => void;
    onCancelEdit: () => void;
    onSubmit: (e: React.FormEvent) => void;
    // Multi-remote Props
    remotes?: string[];
    activeRemote?: string;
    onSelectRemote?: (name: string) => void;
}

/**
 * Header component for the Remote Repository panel.
 * Displays repository info and handles URL editing.
 */
const RemoteHeader: React.FC<RemoteHeaderProps> = ({
    remoteUrl,
    projectName,
    isEditMode,
    isSettingUp,
    setupUrl,
    onSetupUrlChange,
    onEditRemote,
    onDisconnect,
    onCancelEdit,
    onSubmit,
    remotes = [],
    activeRemote,
    onSelectRemote,
}) => {
    const { t } = useTranslation('common');
    const [isCopied, setIsCopied] = useState(false);

    const handleCopyUrl = async () => {
        if (!remoteUrl) return;
        try {
            await navigator.clipboard.writeText(remoteUrl);
            setIsCopied(true);
            setTimeout(() => setIsCopied(false), 2000);
        } catch (err) {
            console.error('Failed to copy:', err);
        }
    };

    const displayTitle = remoteUrl ? projectName : t('remote.noConfigured');

    if (isEditMode) {
        return (
            <div style={headerStyle}>
                <form onSubmit={onSubmit} style={{ display: 'flex', gap: '8px', alignItems: 'center', width: '100%' }}>
                    <input
                        type="text"
                        placeholder="https://github.com/... or git@..."
                        value={setupUrl}
                        onChange={(e) => onSetupUrlChange(e.target.value)}
                        style={{ ...inputStyle, flex: 3 }}
                        autoFocus
                        onFocus={(e) => e.target.select()}
                        data-testid="remote-url-input"
                    />

                    <button type="button" onClick={onCancelEdit} style={cancelButtonStyle} data-testid="remote-cancel-btn">
                        {t('remote.cancel')}
                    </button>
                    <button
                        type="submit"
                        disabled={isSettingUp || !setupUrl}
                        style={{ ...submitButtonStyle, opacity: isSettingUp ? 0.7 : 1 }}
                        data-testid="remote-update-btn"
                    >
                        {isSettingUp ? t('remote.updating') : t('remote.update')}
                    </button>
                </form>
            </div>
        );
    }

    return (
        <div style={headerStyle}>
            {/* Multi-Remote Tabs */}
            {remotes.length > 0 && (
                <div style={{ display: 'flex', gap: '4px', marginBottom: '12px', borderBottom: '1px solid var(--border-subtle)', overflowX: 'auto' }}>
                    {remotes.map(r => (
                        <button
                            key={r}
                            onClick={() => onSelectRemote?.(r)}
                            style={{
                                padding: '6px 12px',
                                borderBottom: activeRemote === r ? '2px solid var(--accent-primary)' : '2px solid transparent',
                                fontWeight: activeRemote === r ? 600 : 400,
                                fontSize: 'var(--text-sm)',
                                cursor: 'pointer',
                                background: 'transparent',
                                borderTop: 'none',
                                borderLeft: 'none',
                                borderRight: 'none',
                                color: activeRemote === r ? 'var(--text-primary)' : 'var(--text-secondary)',
                                transition: 'all 0.2s'
                            }}
                        >
                            {r}
                        </button>
                    ))}
                    {/* New Remote Button (Triggers Edit/Add) */}
                    <button
                        onClick={onEditRemote}
                        title="Add Remote (Enter new URL)"
                        style={{
                            padding: '6px 12px',
                            background: 'transparent',
                            border: 'none',
                            cursor: 'pointer',
                            color: 'var(--text-tertiary)',
                            display: 'flex',
                            alignItems: 'center'
                        }}
                    >
                        <Plus size={14} />
                    </button>
                </div>
            )}

            {/* Title row with Configure button */}
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                <div style={{ fontWeight: 700, fontSize: 'var(--text-md)', display: 'flex', alignItems: 'center', gap: 'var(--space-2)', color: 'var(--text-primary)' }}>
                    {remoteUrl ? (
                        <div style={{ position: 'relative', width: '24px', height: '24px', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                            <Cloud size={20} strokeWidth={2} style={{ position: 'absolute', top: 0, left: 0, color: 'var(--text-primary)' }} />
                            <Database size={10} strokeWidth={2} fill="var(--bg-secondary)" style={{ position: 'absolute', bottom: 2, right: 2, color: 'var(--text-primary)', background: 'var(--bg-secondary)', borderRadius: '50%' }} />
                        </div>
                    ) : null}
                    <span>{displayTitle}</span>

                </div>

                {/* Configure button */}
                {remoteUrl && (
                    <button
                        onClick={remoteUrl.startsWith('remote://') ? onDisconnect : onEditRemote}
                        style={{
                            padding: '3px 10px',
                            fontSize: 'var(--text-sm)',
                            background: 'transparent',
                            border: '1px solid var(--border-subtle)',
                            borderRadius: 'var(--radius-sm)',
                            color: 'var(--text-secondary)',
                            cursor: 'pointer',
                            fontWeight: 600,
                            whiteSpace: 'nowrap',
                            display: 'flex',
                            alignItems: 'center',
                            gap: '4px'
                        }}
                        data-testid="remote-configure-btn"
                    >
                        {remoteUrl.startsWith('remote://') ? (
                            <>
                                <LogOut size={12} />
                                {t('remote.disconnect')}
                            </>
                        ) : (
                            t('remote.configure')
                        )}
                    </button>
                )}
            </div>

            {/* Full-width URL row */}
            {remoteUrl ? (
                <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-2)' }}>
                    <span style={{
                        fontSize: 'var(--text-xs)',
                        color: 'var(--text-tertiary)',
                        whiteSpace: 'nowrap'
                    }}>
                        {t('remote.cloneUrlLabel')}
                    </span>
                    <input
                        type="text"
                        readOnly
                        value={remoteUrl}
                        style={{
                            flex: 1,
                            minWidth: 0,
                            padding: 'var(--space-2) var(--space-3)',
                            fontSize: 'var(--text-sm)',
                            fontFamily: 'monospace',
                            background: 'var(--bg-tertiary)',
                            border: '1px solid var(--border-subtle)',
                            borderRadius: 'var(--radius-md)',
                            color: 'var(--text-primary)',
                            outline: 'none',
                            cursor: 'text'
                        }}
                        onClick={(e) => (e.target as HTMLInputElement).select()}
                    />
                    <div style={{ position: 'relative' }}>
                        <button
                            onClick={handleCopyUrl}
                            title={t('remote.copyUrl')}
                            style={{
                                padding: 'var(--space-1)',
                                background: 'transparent',
                                border: 'none',
                                cursor: 'pointer',
                                display: 'flex',
                                alignItems: 'center',
                                justifyContent: 'center',
                                color: isCopied ? 'var(--accent-primary)' : 'var(--text-secondary)',
                                transition: 'color 0.15s'
                            }}
                        >
                            {isCopied ? <Check size={16} /> : <Copy size={16} />}
                        </button>
                        {/* Copied! tooltip */}
                        {isCopied && (
                            <div style={{
                                position: 'absolute',
                                top: '100%',
                                right: 0,
                                marginTop: '4px',
                                padding: '4px 8px',
                                background: 'var(--accent-primary)',
                                color: 'white',
                                fontSize: 'var(--text-xs)',
                                fontWeight: 600,
                                borderRadius: 'var(--radius-sm)',
                                whiteSpace: 'nowrap',
                                zIndex: 100,
                                animation: 'fadeIn 0.15s ease-out'
                            }}>
                                {t('remote.copied')}
                            </div>
                        )}
                    </div>
                </div>
            ) : null}
        </div>
    );
};

export default RemoteHeader;
