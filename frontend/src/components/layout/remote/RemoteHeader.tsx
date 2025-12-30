import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Settings, Cloud, Copy, Check, Database } from 'lucide-react';
import {
    toolbarRowStyle
} from './remoteStyles';

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
            flexDirection: 'column',
            borderBottom: '1px solid var(--border-subtle)',
            background: 'var(--bg-secondary)', // Same as developer tabs
        }}>
            {/* ROW 1: Title & Main Actions (36px) - Matches DeveloperTabs */}
            <div style={{ ...toolbarRowStyle, padding: '0 12px', justifyContent: 'space-between', height: remoteUrl ? '36px' : '40px', borderBottom: 'none' }}>
                {/* Left: Project Name */}
                <div style={{ display: 'flex', alignItems: 'center', gap: '8px', minWidth: 0, flex: 1 }}>
                    <div style={{ position: 'relative', display: 'flex', alignItems: 'center' }}>
                        <Cloud
                            size={16}
                            className={remoteUrl ? "text-accent-primary" : "text-tertiary"}
                            style={{ opacity: remoteUrl ? 1 : 0.5 }}
                        />
                        <Database
                            size={12}
                            style={{
                                position: 'absolute',
                                right: -4,
                                bottom: -2,
                                color: remoteUrl ? 'var(--accent-primary)' : 'var(--text-tertiary)',
                                background: 'var(--bg-secondary)',
                                borderRadius: '50%'
                            }}
                        />
                    </div>
                    <span style={{
                        fontSize: '13px',
                        fontWeight: 600,
                        color: 'var(--text-primary)',
                        whiteSpace: 'nowrap',
                        overflow: 'hidden',
                        textOverflow: 'ellipsis'
                    }}>
                        {displayTitle}
                    </span>
                </div>

                {/* Settings button - Right aligned */}
                {remoteUrl && !isSettingsOpen && (
                    <button
                        onClick={onEditRemote}
                        title="リポジトリ設定"
                        style={{
                            display: 'flex',
                            alignItems: 'center',
                            gap: '4px',
                            background: 'transparent',
                            border: 'none',
                            color: 'var(--text-secondary)',
                            fontSize: '11px',
                            cursor: 'pointer',
                            padding: '4px 8px',
                            borderRadius: '4px',
                            flexShrink: 0 // Prevent shrinking
                        }}
                        className="hover:bg-bg-tertiary transition-colors"
                    >
                        <Settings size={14} />
                        <span>リモートを設定</span>
                    </button>
                )}
            </div>

            {/* ROW 2: URL & Info (40px) - Matches Local View Toggles */}
            {remoteUrl && (
                <div style={{
                    height: '40px',
                    display: 'flex',
                    alignItems: 'center',
                    padding: '0 12px',
                    background: 'var(--bg-toolbar)', // Distinct background for submenu-like feel
                    // borderTop: '1px solid var(--border-subtle)', // Removed divider as requested
                    gap: '12px'
                }}>
                    <span style={{
                        fontSize: '11px',
                        fontWeight: 600,
                        color: 'var(--text-secondary)',
                        whiteSpace: 'nowrap'
                    }}>
                        リモートURL
                    </span>
                    <div style={{
                        flex: 1,
                        display: 'flex',
                        alignItems: 'center',
                        gap: '8px',
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
                                width: '100%'
                            }}
                        />
                        <button
                            onClick={handleCopyUrl}
                            title="Copy URL"
                            style={{
                                display: 'flex', alignItems: 'center',
                                background: 'transparent', border: 'none',
                                padding: '4px', cursor: 'pointer',
                                color: isCopied ? 'var(--accent-primary)' : 'var(--text-tertiary)'
                            }}
                        >
                            {isCopied ? <Check size={14} /> : <Copy size={14} />}
                        </button>
                    </div>
                </div>
            )}
        </div>
    );
};

export default RemoteHeader;
