import { useState, useRef, useEffect, useCallback } from 'react';
import './AppLayout.css';
import { useGit } from '../../context/GitAPIContext';
import { useTheme } from '../../context/ThemeContext';
import GitTerminal from '../terminal/GitTerminal';
import GitGraphViz from '../visualization/GitGraphViz';
import GitReferenceList from '../visualization/GitReferenceList';
import FileExplorer from './FileExplorer';
import ObjectInspector from './ObjectInspector';
import ObjectGraph from '../ObjectGraph';

export interface SelectedObject {
    type: 'commit' | 'file';
    id: string; // Hash or Path
    data?: any;
}

type ViewMode = 'graph' | 'branches' | 'tags';

const AppLayout = () => {
    const { state, showAllCommits, toggleShowAllCommits } = useGit();
    const { theme, toggleTheme } = useTheme();
    const [selectedObject, setSelectedObject] = useState<SelectedObject | null>(null);
    const [isLeftPaneOpen, setIsLeftPaneOpen] = useState(true);
    const [isInspectOpen, setIsInspectOpen] = useState(false);
    const [viewMode, setViewMode] = useState<ViewMode>('graph');

    // Resizable Pane State (Vertical - Side Panes)
    const [leftPaneWidth, setLeftPaneWidth] = useState(250);
    const [rightPaneWidth, setRightPaneWidth] = useState(250);
    const isResizingLeft = useRef(false);
    const isResizingRight = useRef(false);

    // Resizable Pane State (Horizontal - Center Split)
    const [vizHeight, setVizHeight] = useState(300); // Initial height in pixels
    const vizRef = useRef<HTMLDivElement>(null);
    const centerContentRef = useRef<HTMLDivElement>(null);
    const isResizingViz = useRef(false);

    const startResizingViz = useCallback(() => {
        isResizingViz.current = true;
        document.body.style.cursor = 'row-resize';
        document.body.style.userSelect = 'none';
    }, []);

    const startResizingLeft = useCallback(() => {
        isResizingLeft.current = true;
        document.body.style.cursor = 'col-resize';
        document.body.style.userSelect = 'none';
    }, []);

    const startResizingRight = useCallback(() => {
        isResizingRight.current = true;
        document.body.style.cursor = 'col-resize';
        document.body.style.userSelect = 'none';
    }, []);

    const stopResizing = useCallback(() => {
        isResizingViz.current = false;
        isResizingLeft.current = false;
        isResizingRight.current = false;
        document.body.style.cursor = '';
        document.body.style.userSelect = '';
    }, []);

    const resize = useCallback((e: MouseEvent) => {
        // Horizontal Resize (Center Viz)
        if (isResizingViz.current && centerContentRef.current) {
            const centerRect = centerContentRef.current.getBoundingClientRect();
            const newHeight = e.clientY - centerRect.top;
            const minHeight = 100;
            const maxHeight = centerRect.height - 100;

            if (newHeight >= minHeight && newHeight <= maxHeight) {
                setVizHeight(newHeight);
            }
        }

        // Vertical Resize (Left Pane)
        if (isResizingLeft.current) {
            const newWidth = e.clientX;
            if (newWidth >= 150 && newWidth <= 600) {
                setLeftPaneWidth(newWidth);
            }
        }

        // Vertical Resize (Right Pane)
        if (isResizingRight.current) {
            const newWidth = window.innerWidth - e.clientX;
            if (newWidth >= 150 && newWidth <= 600) {
                setRightPaneWidth(newWidth);
            }
        }
    }, []);

    useEffect(() => {
        window.addEventListener('mousemove', resize);
        window.addEventListener('mouseup', stopResizing);
        return () => {
            window.removeEventListener('mousemove', resize);
            window.removeEventListener('mouseup', stopResizing);
        };
    }, [resize, stopResizing]);

    const handleObjectSelect = (obj: SelectedObject) => {
        setSelectedObject(obj);
    };

    return (
        <div className="layout-container">
            {/* LEFT PANE: Explorer */}
            <aside
                className={`left-pane ${!isLeftPaneOpen ? 'collapsed' : ''}`}
                style={{ width: isLeftPaneOpen ? leftPaneWidth : undefined, minWidth: isLeftPaneOpen ? undefined : '40px' }}
            >
                <div
                    className="pane-header"
                    style={{ justifyContent: 'space-between', paddingLeft: isLeftPaneOpen ? '16px' : '8px', paddingRight: isLeftPaneOpen ? '16px' : '8px' }}
                    onClick={() => !isLeftPaneOpen && setIsLeftPaneOpen(true)} // Click header to expand if collapsed
                >
                    {isLeftPaneOpen && <span>EXPLORER</span>}
                    <button
                        onClick={(e) => {
                            e.stopPropagation();
                            setIsLeftPaneOpen(!isLeftPaneOpen);
                        }}
                        style={{
                            background: 'none',
                            border: 'none',
                            color: 'var(--text-tertiary)',
                            cursor: 'pointer',
                            padding: '4px',
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center'
                        }}
                    >
                        {isLeftPaneOpen ? '◀' : '▶'}
                    </button>
                </div>
                {isLeftPaneOpen && (
                    <div className="pane-content">
                        <FileExplorer onSelect={(fileObj: SelectedObject) => handleObjectSelect(fileObj)} />
                    </div>
                )}
            </aside>

            {/* Resizer Left */}
            <div className="resizer-vertical" onMouseDown={startResizingLeft} style={{ display: isLeftPaneOpen ? 'block' : 'none' }} />

            {/* CENTER PANE: Viz & Terminal */}
            <main className="center-pane">
                {/* Unified Header for Center Pane */}
                <div className="pane-header" style={{ justifyContent: 'space-between' }}>
                    {/* View Switcher */}
                    <div style={{ display: 'flex', background: 'var(--bg-secondary)', borderRadius: '6px', padding: '2px' }}>
                        {(['graph', 'branches', 'tags'] as ViewMode[]).map((mode) => (
                            <button
                                key={mode}
                                onClick={() => setViewMode(mode)}
                                style={{
                                    background: viewMode === mode ? 'var(--accent-primary)' : 'transparent',
                                    color: viewMode === mode ? 'white' : 'var(--text-secondary)',
                                    border: 'none',
                                    borderRadius: '4px',
                                    padding: '4px 12px',
                                    fontSize: '12px',
                                    cursor: 'pointer',
                                    fontWeight: 500
                                }}
                            >
                                {mode}
                            </button>
                        ))}
                    </div>

                    <div style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
                        {/* Toggle Button */}
                        <label style={{ display: 'flex', alignItems: 'center', cursor: 'pointer', gap: '8px', fontSize: '0.8rem', color: 'var(--text-secondary)' }}>
                            <input
                                type="checkbox"
                                checked={showAllCommits}
                                onChange={toggleShowAllCommits}
                                style={{
                                    accentColor: 'var(--accent-primary)',
                                    cursor: 'pointer'
                                }}
                            />
                            Show All Commits
                        </label>

                        {/* Traffic Lights - Premium Feel */}
                        <div style={{ display: 'flex', gap: '8px' }}>
                            <div style={{ width: 10, height: 10, borderRadius: '50%', background: '#ff5f56' }} />
                            <div style={{ width: 10, height: 10, borderRadius: '50%', background: '#ffbd2e' }} />
                            <div style={{ width: 10, height: 10, borderRadius: '50%', background: '#27c93f' }} />
                        </div>

                        {/* Theme Toggle */}
                        <button
                            onClick={toggleTheme}
                            style={{
                                background: 'transparent',
                                border: '1px solid var(--border-subtle)',
                                color: 'var(--text-secondary)',
                                padding: '4px 8px',
                                fontSize: '12px',
                                cursor: 'pointer',
                                display: 'flex',
                                alignItems: 'center',
                                justifyContent: 'center'
                            }}
                            title={`Switch to ${theme === 'dark' ? 'Light' : 'Dark'} Mode`}
                        >
                            {theme === 'dark' ? '☀️' : '🌙'}
                        </button>
                    </div>
                </div>

                <div className="center-content" ref={centerContentRef}>
                    {/* Upper: Visualization */}
                    <div
                        className="viz-pane"
                        style={{ height: vizHeight, flex: 'none', minHeight: 0 }}
                        ref={vizRef}
                    >
                        {state.HEAD && state.HEAD.type !== 'none' ? (
                            viewMode === 'graph' ? (
                                <GitGraphViz
                                    onSelect={(commitData) => handleObjectSelect({ type: 'commit', id: commitData.id, data: commitData })}
                                    selectedCommitId={selectedObject?.type === 'commit' ? selectedObject.id : undefined}
                                />
                            ) : (
                                <GitReferenceList
                                    type={viewMode === 'branches' ? 'branches' : 'tags'}
                                    onSelect={(commitData) => handleObjectSelect({ type: 'commit', id: commitData.id, data: commitData })}
                                    selectedCommitId={selectedObject?.type === 'commit' ? selectedObject.id : undefined}
                                />
                            )
                        ) : (
                            <div style={{
                                height: '100%',
                                display: 'flex',
                                alignItems: 'center',
                                justifyContent: 'center',
                                color: 'var(--text-tertiary)',
                                flexDirection: 'column',
                                gap: '12px'
                            }}>
                                <span style={{ fontSize: '24px', opacity: 0.5 }}>📦</span>
                                <div>Select a project from Workspaces to view Git Graph</div>
                            </div>
                        )}
                    </div>

                    {/* Resizer Handle (Horizontal) */}
                    <div className="resizer" onMouseDown={startResizingViz} />

                    {/* Lower: Terminal */}
                    <div className="terminal-pane" style={{ flex: 1, minHeight: 0 }}>
                        <GitTerminal />
                    </div>
                </div>
            </main>

            {/* Resizer Right */}
            <div className="resizer-vertical" onMouseDown={startResizingRight} />

            {/* RIGHT PANE: Object Inspector */}
            <aside
                className="right-pane"
                style={{ width: rightPaneWidth }}
            >
                <div className="pane-header">Object Inspector</div>
                <div className="pane-content">
                    <ObjectInspector
                        selectedObject={selectedObject}
                        onInspect={() => setIsInspectOpen(true)}
                    />
                </div>
            </aside>

            {/* X-Ray Modal */}
            {isInspectOpen && selectedObject?.type === 'commit' && (
                <ObjectGraph
                    commitId={selectedObject.id}
                    objects={state.objects || {}}
                    onClose={() => setIsInspectOpen(false)}
                />
            )}
        </div>
    );
};

export default AppLayout;
