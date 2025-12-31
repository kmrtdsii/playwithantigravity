import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Settings, Cloud, Copy, Check, Database } from 'lucide-react';


interface RemoteHeaderProps {
    remoteUrl: string;
    projectName: string;
    // Legacy/Unused props kept for compatibility
    isEditMode: boolean;
    isSettingUp: boolean;
    setupUrl: string;
    onSetupUrlChange: (url: string) => void;
    // Actions
    onEditRemote: () => void; // Used for "Settings" action
    onDisconnect?: () => void; // Kept in interface but unused in component
    onCancelEdit: () => void;
    onSubmit: (e: React.FormEvent) => void;
    // Multi-remote Props
    remotes?: string[];
    activeRemote?: string;
    onSelectRemote?: (name: string) => void;
    // New prop to hide settings button when already in settings view
    isSettingsOpen?: boolean;
}

/**
 * Header component for the Remote Repository panel.
 * Simplified design: No tabs, just title, URL copier, and Settings button.
 */
const RemoteHeader: React.FC<RemoteHeaderProps> = ({
    remoteUrl,
    projectName,
    onEditRemote, // This will now open the settings view
    isSettingsOpen = false,
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
            console.error('Failed to copy!', err);
        }
    };

    const displayTitle = remoteUrl ? projectName : (t('remote.empty.title') || 'Remote Repository');

    return (
        <div style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            height: '44px', // Slightly taller for single row comfort
            padding: '0 12px',
            borderBottom: '1px solid var(--border-subtle)',
            background: 'var(--bg-secondary)',
            gap: '12px'
        }}>
            {/* Left: Project Name */}
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px', flexShrink: 0 }}>
                <div style={{ position: 'relative', display: 'flex', alignItems: 'center' }}>
                    <Cloud
                        size={18}
                        className={remoteUrl ? "text-accent-primary" : "text-tertiary"}
                        style={{ opacity: remoteUrl ? 1 : 0.5 }}
                    />
                    <Database
                        size={12}
                        style={{
                            position: 'absolute',
                            right: -4,
                            bottom: -2,
                            color: remoteUrl ? 'var(--text-accent)' : 'var(--text-tertiary)',
                            background: 'var(--bg-secondary)',
                            borderRadius: '50%'
                        }}
                    />
                </div>
                <span style={{
                    fontSize: '13px',
                    fontWeight: 600,
                    color: 'var(--text-primary)',
                    whiteSpace: 'nowrap'
                }}>
                    {displayTitle}
                </span>
            </div>

            {/* Middle: URL Input */}
            {remoteUrl ? (
                <div style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: '4px',
                    flex: 1,
                    maxWidth: '600px', // Prevent stretching too wide on large screens
                    minWidth: 0
                }}>
                    <input
                        type="text"
                        readOnly
                        value={remoteUrl}
                        style={{
                            flex: 1,
                            background: 'var(--bg-tertiary)',
                            border: '1px solid var(--border-subtle)',
                            borderRadius: '4px',
                            padding: '4px 8px',
                            fontSize: '11px',
                            fontFamily: 'monospace',
                            color: 'var(--text-secondary)',
                            outline: 'none',
                            minWidth: 0
                        }}
                    />
                    <button
                        onClick={handleCopyUrl}
                        title={t('remote.header.copyUrl')}
                        style={{
                            display: 'flex', alignItems: 'center',
                            background: 'transparent', border: 'none',
                            padding: '4px', cursor: 'pointer',
                            color: isCopied ? 'var(--accent-primary)' : 'var(--text-tertiary)',
                            flexShrink: 0
                        }}
                    >
                        {isCopied ? <Check size={14} /> : <Copy size={14} />}
                    </button>
                </div>
            ) : (
                <div style={{ flex: 1 }} />
            )}

            {/* Right Group: Build Info & Settings */}
            <div style={{ display: 'flex', alignItems: 'center', gap: '12px', flexShrink: 0 }}>
                {!remoteUrl && (
                    <div style={{
                        fontSize: '9px',
                        color: 'var(--text-secondary)',
                        opacity: 0.6,
                        fontFamily: 'monospace',
                        whiteSpace: 'nowrap'
                    }}>
                        {__BUILD_TIME__}
                    </div>
                )}

                {!isSettingsOpen && remoteUrl && (
                    <button
                        onClick={onEditRemote}
                        title={t('remote.header.settings')}
                        style={{
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            background: 'transparent',
                            border: 'none',
                            color: 'var(--text-secondary)',
                            cursor: 'pointer',
                            padding: '6px',
                            borderRadius: '4px',
                            transition: 'background-color 0.2s'
                        }}
                        className="hover:bg-bg-tertiary"
                    >
                        <Settings size={16} />
                    </button>
                )}
            </div>
        </div>
    );
};

export default RemoteHeader;
