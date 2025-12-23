import React, { useState } from 'react';
import { Cloud, Database } from 'lucide-react';
import { headerStyle, inputStyle, cancelButtonStyle, submitButtonStyle } from './remoteStyles';

interface RemoteHeaderProps {
    remoteUrl: string;
    projectName: string;
    isEditMode: boolean;
    isSettingUp: boolean;
    setupUrl: string;
    onSetupUrlChange: (url: string) => void;
    onEditRemote: () => void;
    onCancelEdit: () => void;
    onSubmit: (e: React.FormEvent) => void;
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
    onCancelEdit,
    onSubmit,
}) => {
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

    const displayTitle = remoteUrl ? projectName : 'NO REMOTE CONFIGURED';

    if (isEditMode) {
        return (
            <div style={headerStyle}>
                <form onSubmit={onSubmit} style={{ display: 'flex', gap: '8px', alignItems: 'center', width: '100%' }}>
                    <input
                        type="text"
                        placeholder="https://github.com/..."
                        value={setupUrl}
                        onChange={(e) => onSetupUrlChange(e.target.value)}
                        style={inputStyle}
                        autoFocus
                    />
                    <button type="button" onClick={onCancelEdit} style={cancelButtonStyle}>
                        Cancel
                    </button>
                    <button
                        type="submit"
                        disabled={isSettingUp || !setupUrl}
                        style={{ ...submitButtonStyle, opacity: isSettingUp ? 0.7 : 1 }}
                    >
                        {isSettingUp ? 'UPDATING...' : 'UPDATE'}
                    </button>
                </form>
            </div>
        );
    }

    return (
        <div style={headerStyle}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                <div style={{ display: 'flex', flexDirection: 'column', gap: '2px', overflow: 'hidden' }}>
                    <div style={{ fontWeight: 700, fontSize: '0.9rem', display: 'flex', alignItems: 'center', gap: '6px', color: 'var(--text-primary)' }}>
                        {remoteUrl ? (
                            <div style={{ position: 'relative', width: '24px', height: '24px', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                                <Cloud size={20} strokeWidth={2} style={{ position: 'absolute', top: 0, left: 0, color: 'var(--text-primary)' }} />
                                <Database size={10} strokeWidth={2} fill="var(--bg-secondary)" style={{ position: 'absolute', bottom: 2, right: 2, color: 'var(--text-primary)', background: 'var(--bg-secondary)', borderRadius: '50%' }} />
                            </div>
                        ) : null}
                        <span>{displayTitle}</span>
                        {remoteUrl && (
                            <span style={{
                                fontSize: '0.65rem',
                                background: '#238636',
                                color: 'white',
                                padding: '1px 6px',
                                borderRadius: '10px',
                                fontWeight: 600
                            }}>
                                origin
                            </span>
                        )}
                    </div>
                    {remoteUrl ? (
                        <div style={{ marginTop: '6px' }}>
                            {/* Label above URL box */}
                            <div style={{
                                fontSize: '11px',
                                color: 'var(--text-tertiary)',
                                marginBottom: '4px'
                            }}>
                                Clone using the web URL.
                            </div>
                            {/* GitHub-style URL input box */}
                            <div style={{
                                display: 'flex',
                                alignItems: 'center',
                                border: '1px solid var(--border-subtle)',
                                borderRadius: '6px',
                                background: 'var(--bg-primary)',
                                overflow: 'hidden'
                            }}>
                                <input
                                    type="text"
                                    readOnly
                                    value={remoteUrl}
                                    style={{
                                        flex: 1,
                                        padding: '8px 12px',
                                        fontSize: '12px',
                                        fontFamily: 'monospace',
                                        background: 'transparent',
                                        border: 'none',
                                        color: 'var(--text-primary)',
                                        outline: 'none',
                                        cursor: 'text'
                                    }}
                                    onClick={(e) => (e.target as HTMLInputElement).select()}
                                />
                                <button
                                    onClick={handleCopyUrl}
                                    title={isCopied ? 'Copied!' : 'Copy URL to clipboard'}
                                    style={{
                                        padding: '8px 12px',
                                        background: isCopied ? 'var(--accent-primary)' : 'var(--bg-secondary)',
                                        border: 'none',
                                        borderLeft: '1px solid var(--border-subtle)',
                                        cursor: 'pointer',
                                        display: 'flex',
                                        alignItems: 'center',
                                        justifyContent: 'center',
                                        color: isCopied ? 'white' : 'var(--text-secondary)',
                                        transition: 'background-color 0.15s, color 0.15s'
                                    }}
                                >
                                    {isCopied ? '✓' : '📋'}
                                </button>
                            </div>
                        </div>
                    ) : (
                        <div style={{
                            fontSize: '12px',
                            color: 'var(--text-tertiary)',
                            marginTop: '4px'
                        }}>
                            Connect a GitHub repository to visualize remote history.
                        </div>
                    )}
                </div>

                {/* Only show Configure button if remote is set. If empty, the main pane button handles it. */}
                {remoteUrl && (
                    <button
                        onClick={onEditRemote}
                        style={{
                            padding: '3px 10px',
                            fontSize: '12px',
                            background: 'transparent',
                            border: '1px solid var(--border-subtle)',
                            borderRadius: '4px',
                            color: 'var(--text-secondary)',
                            cursor: 'pointer',
                            fontWeight: 600,
                            whiteSpace: 'nowrap'
                        }}
                    >
                        Configure
                    </button>
                )}
            </div>
        </div>
    );
};

export default RemoteHeader;
