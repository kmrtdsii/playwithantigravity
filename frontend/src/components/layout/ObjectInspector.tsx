import React from 'react';
import { useGit } from '../../context/GitAPIContext';

interface ObjectInspectorProps {
    selectedObject?: {
        type: 'commit' | 'file';
        id: string; // Commit Hash or File Path
        data?: any; // Additional data (message, author, content preview, etc.)
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

    // File View (Future enhancement: show diff or content)
    if (selectedObject.type === 'file') {
        return (
            <div style={containerStyle}>
                <div style={headerStyle}>File Inspector</div>
                <div style={contentStyle}>
                    <div style={itemStyle}>
                        <span style={labelStyle}>Path:</span>
                        <span style={valueStyle}>{selectedObject.id}</span>
                    </div>
                    {selectedObject.data?.status && (
                        <div style={itemStyle}>
                            <span style={labelStyle}>Status:</span>
                            <span style={valueStyle}>{selectedObject.data.status}</span>
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
