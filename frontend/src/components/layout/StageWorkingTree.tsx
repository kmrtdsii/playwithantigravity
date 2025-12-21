import { useMemo } from 'react';
import { useGit } from '../../context/GitAPIContext';
import './AppLayout.css'; // Re-use layout styles or create specific ones if needed

// Interface for the prop
interface StageWorkingTreeProps {
    onSelect?: (file: { type: 'file', id: string, data: any }) => void;
}

const StageWorkingTree: React.FC<StageWorkingTreeProps> = ({ onSelect }) => {
    const { state } = useGit();

    // De-structure state for easier access
    const { staging, modified } = state;

    // Helper to separate untracked files
    const untracked = useMemo(() => {
        return state.files || [];
    }, [state.files]);

    const FileCard = ({ name, status }: { name: string, status: 'untracked' | 'staged' | 'modified' }) => {
        let statusColor = '#999';
        if (status === 'staged') statusColor = '#27c93f'; // Green
        if (status === 'modified') statusColor = '#ffbd2e'; // Yellow
        if (status === 'untracked') statusColor = '#ff5f56'; // Red

        return (
            <div
                onClick={() => onSelect && onSelect({ type: 'file', id: name, data: { status } })}
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
                    cursor: 'pointer',
                    userSelect: 'none'
                }}
            >
                <span style={{ color: 'var(--text-primary)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {name}
                </span>
                <span style={{ fontSize: '0.75rem', color: statusColor, textTransform: 'uppercase', fontWeight: 600 }}>
                    {status}
                </span>
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
            {/* DEBUG INFO - REMOVE LATER */}
            <div style={{ padding: '8px', border: '1px dashed red', marginBottom: '16px', fontSize: '0.75rem', color: 'red' }}>
                DEBUG:
                Staged: {staging.length} |
                Modified: {modified.length} |
                Untracked: {untracked.length} |
                Files: {state.files?.length}
            </div>

            <div style={{ marginBottom: '24px' }}>
                <h3 style={{
                    fontSize: '0.85rem',
                    textTransform: 'uppercase',
                    color: 'var(--text-tertiary)',
                    marginBottom: '10px',
                    letterSpacing: '0.05em'
                }}>Staged Changes</h3>
                {staging.length === 0 && <div style={{ color: 'var(--text-secondary)', fontSize: '0.85rem', fontStyle: 'italic' }}>No staged changes</div>}
                {staging.map(f => <FileCard key={f} name={f} status="staged" />)}
            </div>

            <div style={{ marginBottom: '24px' }}>
                <h3 style={{
                    fontSize: '0.85rem',
                    textTransform: 'uppercase',
                    color: 'var(--text-tertiary)',
                    marginBottom: '10px',
                    letterSpacing: '0.05em'
                }}>Modified (Working Tree)</h3>
                {modified.length === 0 && <div style={{ color: 'var(--text-secondary)', fontSize: '0.85rem', fontStyle: 'italic' }}>Clean working tree</div>}
                {modified.map(f => <FileCard key={f} name={f} status="modified" />)}
            </div>

            <div style={{ marginBottom: '24px' }}>
                <h3 style={{
                    fontSize: '0.85rem',
                    textTransform: 'uppercase',
                    color: 'var(--text-tertiary)',
                    marginBottom: '10px',
                    letterSpacing: '0.05em'
                }}>Untracked</h3>
                {untracked.length === 0 && <div style={{ color: 'var(--text-secondary)', fontSize: '0.85rem', fontStyle: 'italic' }}>No untracked files</div>}
                {untracked.map(f => <FileCard key={f} name={f} status="untracked" />)}
            </div>
        </div>
    );
};

export default StageWorkingTree;
