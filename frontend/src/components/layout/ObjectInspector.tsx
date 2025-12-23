import React from 'react';
import { useGit } from '../../context/GitAPIContext';

interface SelectedObject {
    type: 'commit' | 'file';
    id: string;
    data?: CommitData | FileData;
}

interface CommitData {
    message?: string;
    author?: string;
    timestamp?: string;
}

interface FileData {
    view?: 'staged' | 'worktree';
    content?: string;
}

interface ObjectInspectorProps {
    selectedObject?: SelectedObject | null;
}

// --- Sub-Components (moved outside of main component) ---

interface HeadInspectorProps {
    headId: string | null | undefined;
    headRef: string | null | undefined;
    headType: 'branch' | 'commit' | 'none';
}

const HeadInspector: React.FC<HeadInspectorProps> = ({ headId, headRef, headType }) => (
    <div>
        <div style={headerStyle}>HEAD Inspector</div>
        <div style={contentStyle}>
            <div style={itemStyle}>
                <span style={labelStyle}>Current HEAD:</span>
                <span style={valueStyle}>{headId || headRef || 'Detached'}</span>
            </div>
            <div style={itemStyle}>
                <span style={labelStyle}>Branch:</span>
                <span style={valueStyle}>{headType === 'branch' ? headRef : 'Detached'}</span>
            </div>
        </div>
    </div>
);

interface CommitInspectorProps {
    commit: { id: string; data?: CommitData };
}

const CommitInspector: React.FC<CommitInspectorProps> = ({ commit }) => (
    <div>
        <div style={headerStyle}>Commit Inspector</div>
        <div style={contentStyle}>
            <div style={itemStyle}>
                <span style={labelStyle}>Hash:</span>
                <span style={{ ...valueStyle, fontFamily: 'monospace' }}>{commit.id.substring(0, 7)}</span>
            </div>
            {commit.data?.message && (
                <div style={{ margin: '12px 0' }}>
                    <span style={labelStyle}>Message:</span>
                    <p style={{ marginTop: '4px', whiteSpace: 'pre-wrap', color: 'var(--text-primary)' }}>
                        {commit.data.message}
                    </p>
                </div>
            )}
            {commit.data?.author && (
                <div style={itemStyle}>
                    <span style={labelStyle}>Author:</span>
                    <span style={valueStyle}>{commit.data.author}</span>
                </div>
            )}
            {commit.data?.timestamp && (
                <div style={itemStyle}>
                    <span style={labelStyle}>Date:</span>
                    <span style={valueStyle}>{new Date(commit.data.timestamp).toLocaleString()}</span>
                </div>
            )}
        </div>
    </div>
);

interface FileInspectorProps {
    file: { id: string; data?: FileData };
    fileStatuses: Record<string, string>;
}

const FileInspector: React.FC<FileInspectorProps> = ({ file, fileStatuses }) => {
    const { id, data } = file;
    const view = data?.view;
    const xy = fileStatuses[id] || '??';
    let inspectorTitle = 'File Inspector';
    if (view === 'staged') inspectorTitle = 'index Inspector (HEAD vs Index)';
    if (view === 'worktree') inspectorTitle = 'Worktree Inspector (Index vs Worktree)';
    const actionSuggestion = getActionSuggestion(xy, view);

    return (
        <div>
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
};

// --- Main Component ---

const ObjectInspector: React.FC<ObjectInspectorProps> = ({ selectedObject }) => {
    const { state } = useGit();

    return (
        <div style={containerStyle}>
            <HeadInspector
                headId={state.HEAD.id}
                headRef={state.HEAD.ref}
                headType={state.HEAD.type}
            />

            <div style={{ height: '1px', background: 'var(--border-subtle)', margin: '24px 0' }}></div>

            {!selectedObject && (
                <div style={{ marginTop: '0px', fontSize: '0.85rem', color: 'var(--text-secondary)', fontStyle: 'italic' }}>
                    Select a commit from the graph or a file from the list to view details.
                </div>
            )}

            {selectedObject?.type === 'commit' && (
                <CommitInspector commit={{ id: selectedObject.id, data: selectedObject.data as CommitData }} />
            )}
            {selectedObject?.type === 'file' && (
                <FileInspector
                    file={{ id: selectedObject.id, data: selectedObject.data as FileData }}
                    fileStatuses={state.fileStatuses}
                />
            )}
        </div>
    );
};

// --- Helper Functions ---

/**
 * Returns an action suggestion based on file status code.
 */
const getActionSuggestion = (xy: string, view?: string): string | null => {
    const x = xy[0];
    const y = xy[1];

    if (xy === '??') return "Untracked file. Run `git add <file>` to track it.";
    if (xy === '!!') return "Ignored file.";

    if (view === 'staged') {
        if (x === 'M') return "Staged change. Run `git commit` to record it.";
        if (x === 'A') return "Staged new file. Run `git commit` to record it.";
        if (x === 'D') return "Staged deletion. Run `git commit` to record it.";
        if (x === ' ') return "No staged changes.";
    }

    if (view === 'worktree') {
        if (y === 'M') return "Modified in worktree. Run `git add <file>` to stage changes.";
        if (y === 'D') return "Deleted in worktree. Run `git add <file>` to stage deletion.";
        if (y === ' ') return "Clean in worktree.";
    }

    if (y === 'M') return "Has unstaged changes. Run `git add` to stage.";
    if (x === 'M' || x === 'A') return "Has staged changes. Ready to commit.";

    return "Check file status.";
};

// --- Styles ---

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
