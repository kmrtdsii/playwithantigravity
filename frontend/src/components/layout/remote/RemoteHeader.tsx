import React, { useState } from 'react';
import { Cloud, Database, Copy, Check } from 'lucide-react';
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
                        onFocus={(e) => e.target.select()}
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
                    {remoteUrl && (
                        <span style={{
                            fontSize: 'var(--text-xs)',
                            background: 'var(--accent-primary)',
                            color: 'white',
                            padding: '1px 6px',
                            borderRadius: '10px',
                            fontWeight: 600
                        }}>
                            origin
                        </span>
                    )}
                </div>

                {/* Configure button */}
                {remoteUrl && (
                    <button
                        onClick={onEditRemote}
                        style={{
                            padding: '3px 10px',
                            fontSize: 'var(--text-sm)',
                            background: 'transparent',
                            border: '1px solid var(--border-subtle)',
                            borderRadius: 'var(--radius-sm)',
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

            {/* Full-width URL row */}
            {remoteUrl ? (
                <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-2)' }}>
                    <span style={{
                        fontSize: 'var(--text-xs)',
                        color: 'var(--text-tertiary)',
                        whiteSpace: 'nowrap'
                    }}>
                        Clone using the web URL.
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
                    <button
                        onClick={handleCopyUrl}
                        title={isCopied ? 'Copied!' : 'Copy URL to clipboard'}
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
                </div>
            ) : (
                <div style={{
                    fontSize: 'var(--text-sm)',
                    color: 'var(--text-tertiary)',
                    marginTop: 'var(--space-1)'
                }}>
                    Connect a GitHub repository to visualize remote history.
                </div>
            )}
        </div>
    );
};

export default RemoteHeader;
