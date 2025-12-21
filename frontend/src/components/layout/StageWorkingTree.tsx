import { } from 'react';
import { useGit } from '../../context/GitAPIContext';
import './AppLayout.css'; // Re-use layout styles or create specific ones if needed

// Interface for the prop
interface StageWorkingTreeProps {
    onSelect?: (file: { type: 'file', id: string, data: any }) => void;
}

const StageWorkingTree: React.FC<StageWorkingTreeProps> = ({ onSelect }) => {
    const { state } = useGit();

    // De-structure state for easier access
    const { staging, modified, untracked, fileStatuses, files } = state;



    const FileCard = ({ name, status }: { name: string, status: 'untracked' | 'staged' | 'modified' }) => {
        let statusColor = '#999';
        if (status === 'staged') statusColor = '#27c93f'; // Green
        if (status === 'modified') statusColor = '#ffbd2e'; // Yellow
        if (status === 'untracked') statusColor = '#ff5f56'; // Red

        const xy = fileStatuses[name] || '??';
        const x = xy[0] || '?';
        const y = xy[1] || '?';

        return (
            <div
                style={{
                    padding: '8px 12px',
                    marginBottom: '8px',
                    background: 'var(--bg-tertiary)',
                    borderRadius: '6px',
                    borderLeft: `4px solid ${statusColor}`,
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center',
                    fontSize: '0.9rem',
                    userSelect: 'none'
                }}
            >
                <span
                    onClick={() => onSelect && onSelect({ type: 'file', id: name, data: { status } })}
                    style={{ color: 'var(--text-primary)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', cursor: 'pointer' }}
                >
                    {name}
                </span>

                <div style={{ display: 'flex', gap: '2px', marginLeft: '8px' }}>
                    {/* X Status Badge - Left Click */}
                    <span
                        onClick={(e) => {
                            e.stopPropagation();
                            onSelect && onSelect({ type: 'file', id: name, data: { status, view: 'staged' } })
                        }}
                        style={{
                            fontSize: '0.75rem',
                            color: x !== ' ' && x !== '?' ? '#27c93f' : '#666',
                            background: x !== ' ' && x !== '?' ? 'rgba(39, 201, 63, 0.1)' : 'transparent',
                            padding: '2px 4px',
                            borderRadius: '2px 0 0 2px',
                            border: '1px solid #333',
                            borderRight: 'none',
                            cursor: 'pointer',
                            minWidth: '16px',
                            textAlign: 'center'
                        }}
                        title="Index Status (Click to compare HEAD vs Index)"
                    >
                        {x}
                    </span>
                    {/* Y Status Badge - Right Click */}
                    <span
                        onClick={(e) => {
                            e.stopPropagation();
                            onSelect && onSelect({ type: 'file', id: name, data: { status, view: 'worktree' } })
                        }}
                        style={{
                            fontSize: '0.75rem',
                            color: y !== ' ' ? '#ffbd2e' : '#666',
                            background: y !== ' ' ? 'rgba(255, 189, 46, 0.1)' : 'transparent',
                            padding: '2px 4px',
                            borderRadius: '0 2px 2px 0',
                            border: '1px solid #333',
                            cursor: 'pointer',
                            minWidth: '16px',
                            textAlign: 'center'
                        }}
                        title="Worktree Status (Click to compare Index vs Worktree)"
                    >
                        {y}
                    </span>
                </div>
            </div>
        );
    };

    // Loading state
    if (!state.initialized) {
        return (
            <div style={{ padding: '16px', display: 'flex', flex: 1, alignItems: 'center', justifyContent: 'center', color: 'var(--text-secondary)', minHeight: '100px' }}>
                Loading...
            </div>
        );
    }

    return (
        <div style={{ padding: '16px', flex: 1, minHeight: 0, overflowY: 'auto' }}>

            {/* Staged Area */}
            <div style={{
                marginBottom: '24px',
                border: '1px solid rgba(39, 201, 63, 0.3)',
                background: 'rgba(39, 201, 63, 0.05)',
                borderRadius: '8px',
                padding: '12px'
            }}>
                <h3 style={{
                    fontSize: '0.85rem',
                    textTransform: 'uppercase',
                    color: '#27c93f',
                    marginBottom: '10px',
                    letterSpacing: '0.05em',
                    display: 'flex',
                    justifyContent: 'space-between'
                }}>
                    Staged Changes
                    <span style={{ fontSize: '0.75rem', opacity: 0.7 }}>{staging.length}</span>
                </h3>
                {staging.length === 0 && <div style={{ color: 'var(--text-secondary)', fontSize: '0.85rem', fontStyle: 'italic' }}>No staged changes</div>}
                {staging.map(f => <FileCard key={f} name={f} status="staged" />)}
            </div>

            {/* Modified Area */}
            <div style={{
                marginBottom: '24px',
                border: '1px solid rgba(255, 189, 46, 0.3)',
                background: 'rgba(255, 189, 46, 0.05)',
                borderRadius: '8px',
                padding: '12px'
            }}>
                <h3 style={{
                    fontSize: '0.85rem',
                    textTransform: 'uppercase',
                    color: '#ffbd2e',
                    marginBottom: '10px',
                    letterSpacing: '0.05em',
                    display: 'flex',
                    justifyContent: 'space-between'
                }}>
                    Modified (Working Tree)
                    <span style={{ fontSize: '0.75rem', opacity: 0.7 }}>{modified.length}</span>
                </h3>
                {modified.length === 0 && <div style={{ color: 'var(--text-secondary)', fontSize: '0.85rem', fontStyle: 'italic' }}>Clean working tree</div>}
                {modified.map(f => <FileCard key={f} name={f} status="modified" />)}
            </div>

            {/* Untracked Area */}
            <div style={{
                marginBottom: '24px',
                border: '1px solid rgba(255, 95, 86, 0.3)',
                background: 'rgba(255, 95, 86, 0.05)',
                borderRadius: '8px',
                padding: '12px'
            }}>
                <h3 style={{
                    fontSize: '0.85rem',
                    textTransform: 'uppercase',
                    color: '#ff5f56',
                    marginBottom: '10px',
                    letterSpacing: '0.05em',
                    display: 'flex',
                    justifyContent: 'space-between'
                }}>
                    Untracked
                    <span style={{ fontSize: '0.75rem', opacity: 0.7 }}>{untracked.length}</span>
                </h3>
                {untracked.length === 0 && <div style={{ color: 'var(--text-secondary)', fontSize: '0.85rem', fontStyle: 'italic' }}>No untracked files</div>}
                {untracked.map(f => <FileCard key={f} name={f} status="untracked" />)}
            </div>

            {/* All Files Area - Minimal display */}
            <div style={{
                marginTop: '32px',
                borderTop: '1px solid var(--border-subtle)',
                paddingTop: '16px'
            }}>
                <h3 style={{
                    fontSize: '0.85rem',
                    textTransform: 'uppercase',
                    color: 'var(--text-tertiary)',
                    marginBottom: '10px',
                    letterSpacing: '0.05em'
                }}>All Files ({files.length})</h3>
                <div style={{ display: 'grid', gridTemplateColumns: '1fr', gap: '4px' }}>
                    {files.map(f => (
                        <div key={f} style={{
                            fontSize: '0.85rem',
                            color: 'var(--text-secondary)',
                            padding: '4px 8px',
                            background: 'var(--bg-secondary)',
                            borderRadius: '4px'
                        }}>
                            {f}
                        </div>
                    ))}
                </div>
            </div>
        </div>
    );
};

export default StageWorkingTree;
