import React from 'react';
import { useGit } from '../../context/GitAPIContext';

interface ObjectInspectorProps {
    selectedObject?: {
        type: 'commit' | 'file';
        id: string; // Commit Hash or File Path
        data?: any; // Additional data (message, author, content preview, status, view, etc.)
    } | null;
}

const ObjectInspector: React.FC<ObjectInspectorProps> = ({ selectedObject }) => {
    const { state } = useGit();

    // Default view: HEAD Info
    if (!selectedObject) {
        return (
            <div style={containerStyle}>
                <div style={headerStyle}>HEAD Inspector</div>
                <div style={contentStyle}>
                    <div style={itemStyle}>
                        <span style={labelStyle}>Current HEAD:</span>
                        <span style={valueStyle}>{state.HEAD.id || state.HEAD.ref || 'Detached'}</span>
                    </div>
                    <div style={itemStyle}>
                        <span style={labelStyle}>Branch:</span>
                        <span style={valueStyle}>{state.HEAD.type === 'branch' ? state.HEAD.ref : 'Detached'}</span>
                    </div>
                    <div style={{ marginTop: '20px', fontSize: '0.85rem', color: 'var(--text-secondary)' }}>
                        Select a commit from the graph or a file from the list to view details.
                    </div>
                </div>
            </div>
        );
    }

    // Commit View
    if (selectedObject.type === 'commit') {
        return (
            <div style={containerStyle}>
                <div style={headerStyle}>Commit Inspector</div>
                <div style={contentStyle}>
                    <div style={itemStyle}>
                        <span style={labelStyle}>Hash:</span>
                        <span style={{ ...valueStyle, fontFamily: 'monospace' }}>{selectedObject.id.substring(0, 7)}</span>
                    </div>
                    {selectedObject.data?.message && (
                        <div style={{ margin: '12px 0' }}>
                            <span style={labelStyle}>Message:</span>
                            <p style={{ marginTop: '4px', whiteSpace: 'pre-wrap', color: 'var(--text-primary)' }}>
                                {selectedObject.data.message}
                            </p>
                        </div>
                    )}
                    {selectedObject.data?.author && (
                        <div style={itemStyle}>
                            <span style={labelStyle}>Author:</span>
                            <span style={valueStyle}>{selectedObject.data.author}</span>
                        </div>
                    )}
                    {selectedObject.data?.date && (
                        <div style={itemStyle}>
                            <span style={labelStyle}>Date:</span>
                            <span style={valueStyle}>{selectedObject.data.date}</span>
                        </div>
                    )}
                </div>
            </div>
        );
    }

    // File View
    if (selectedObject.type === 'file') {
        const { id, data } = selectedObject;
        const view = data?.view; // 'staged' | 'worktree' | undefined


        // Get precise status code from state if possible
        const xy = state.fileStatuses[id] || '??';

        let inspectorTitle = 'File Inspector';
        if (view === 'staged') inspectorTitle = 'index Inspector (HEAD vs Index)';
        if (view === 'worktree') inspectorTitle = 'Worktree Inspector (Index vs Worktree)';

        const actionSuggestion = getActionSuggestion(xy, view);

        return (
            <div style={containerStyle}>
                <div style={headerStyle}>{inspectorTitle}</div>
                <div style={contentStyle}>
                    <div style={itemStyle}>
                        <span style={labelStyle}>Path:</span>
                        <span style={valueStyle}>{id}</span>
                    </div>

                    <div style={itemStyle}>
                        <span style={labelStyle}>Status Code:</span>
                        <span style={{ ...valueStyle, fontFamily: 'monospace', fontSize: '1.2rem', fontWeight: 'bold' }}>
                            {xy}
                        </span>
                    </div>

                    {actionSuggestion && (
                        <div style={{ marginTop: '16px', padding: '12px', background: 'var(--bg-secondary)', borderRadius: '6px', borderLeft: '4px solid var(--accent-primary)' }}>
                            <span style={{ display: 'block', fontSize: '0.75rem', fontWeight: 'bold', color: 'var(--accent-primary)', marginBottom: '4px' }}>
                                SUGGESTED ACTION
                            </span>
                            <span style={{ fontSize: '0.9rem', color: 'var(--text-primary)' }}>
                                {actionSuggestion}
                            </span>
                        </div>
                    )}

                    <div style={{ marginTop: '20px', fontStyle: 'italic', color: 'var(--text-tertiary)' }}>
                        File content preview not yet implemented.
                    </div>
                </div>
            </div>
        );
    }

    return null;
};

// Helper for Action Guide
const getActionSuggestion = (xy: string, view?: string) => {
    const x = xy[0];
    const y = xy[1];

    if (xy === '??') return "Untracked file. Run `git add <file>` to track it.";
    if (xy === '!!') return "Ignored file.";

    if (view === 'staged') {
        // Focusing on X (Index)
        if (x === 'M') return "Staged change. Run `git commit` to record it.";
        if (x === 'A') return "Staged new file. Run `git commit` to record it.";
        if (x === 'D') return "Staged deletion. Run `git commit` to record it.";
        if (x === ' ') return "No staged changes.";
    }

    if (view === 'worktree') {
        // Focusing on Y (Worktree)
        if (y === 'M') return "Modified in worktree. Run `git add <file>` to stage changes.";
        if (y === 'D') return "Deleted in worktree. Run `git add <file>` to stage deletion.";
        if (y === ' ') return "Clean in worktree.";
    }

    // Default general advice if no view specific
    if (y === 'M') return "Has unstaged changes. Run `git add` to stage.";
    if (x === 'M' || x === 'A') return "Has staged changes. Ready to commit.";

    return "Check file status.";
};

// Styles
const containerStyle: React.CSSProperties = {
    padding: '16px',
    height: '100%',
    overflowY: 'auto',
    display: 'flex',
    flexDirection: 'column',
};

const headerStyle: React.CSSProperties = {
    fontSize: '0.9rem',
    textTransform: 'uppercase',
    fontWeight: 700,
    color: 'var(--accent-primary)',
    borderBottom: '1px solid var(--border-subtle)',
    paddingBottom: '12px',
    marginBottom: '16px',
    letterSpacing: '0.05em'
};

const contentStyle: React.CSSProperties = {
    display: 'flex',
    flexDirection: 'column',
    gap: '12px'
};

const itemStyle: React.CSSProperties = {
    display: 'flex',
    flexDirection: 'column',
    gap: '4px'
};

const labelStyle: React.CSSProperties = {
    fontSize: '0.75rem',
    color: 'var(--text-tertiary)',
    textTransform: 'uppercase',
    fontWeight: 600
};

const valueStyle: React.CSSProperties = {
    fontSize: '0.9rem',
    color: 'var(--text-secondary)',
    wordBreak: 'break-all'
};

export default ObjectInspector;
