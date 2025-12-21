import { useState, useRef, useEffect, useCallback } from 'react';
import './AppLayout.css';
import { useGit } from '../../context/GitAPIContext';
import GitTerminal from '../terminal/GitTerminal';
import GitGraphViz from '../visualization/GitGraphViz';
import FileExplorer from './FileExplorer';
import ObjectInspector from './ObjectInspector';

export interface SelectedObject {
    type: 'commit' | 'file';
    id: string; // Hash or Path
    data?: any;
}

const AppLayout = () => {
    const { showAllCommits, toggleShowAllCommits } = useGit();
    const [selectedObject, setSelectedObject] = useState<SelectedObject | null>(null);
    const [isLeftPaneOpen, setIsLeftPaneOpen] = useState(true);

    // Resizable Pane State
    const [vizHeight, setVizHeight] = useState(300); // Initial height in pixels
    const vizRef = useRef<HTMLDivElement>(null);
    const centerContentRef = useRef<HTMLDivElement>(null);
    const isResizing = useRef(false);

    const startResizing = useCallback(() => {
        isResizing.current = true;
        document.body.style.cursor = 'row-resize';
        document.body.style.userSelect = 'none'; // Prevent selection during drag
    }, []);

    const stopResizing = useCallback(() => {
        isResizing.current = false;
        document.body.style.cursor = '';
        document.body.style.userSelect = '';
    }, []);

    const resize = useCallback((e: MouseEvent) => {
        if (!isResizing.current || !centerContentRef.current) return;

        // Calculate relative height
        // We want the new height of the viz pane to be (MouseY - CenterPaneTop)
        const centerRect = centerContentRef.current.getBoundingClientRect();
        const newHeight = e.clientY - centerRect.top;

        // Min/Max constraints
        const minHeight = 100;
        const maxHeight = centerRect.height - 100; // Keep space for terminal

        if (newHeight >= minHeight && newHeight <= maxHeight) {
            setVizHeight(newHeight);
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
            {/* LEFT PANE: Explorer (1/4) */}
            <aside className={`left-pane ${!isLeftPaneOpen ? 'collapsed' : ''}`}>
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

            {/* CENTER PANE: Viz & Terminal (2/4) */}
            <main className="center-pane">
                {/* Unified Header for Center Pane */}
                <div className="pane-header" style={{ justifyContent: 'space-between' }}>
                    <span>Repository Visualization & Terminal</span>

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
                    </div>
                </div>

                <div className="center-content" ref={centerContentRef}>
                    {/* Upper: Visualization */}
                    <div
                        className="viz-pane"
                        style={{ height: vizHeight, flex: 'none', minHeight: 0 }}
                        ref={vizRef}
                    >
                        <GitGraphViz
                            onSelect={(commitData) => handleObjectSelect({ type: 'commit', id: commitData.id, data: commitData })}
                            selectedCommitId={selectedObject?.type === 'commit' ? selectedObject.id : undefined}
                        />
                    </div>

                    {/* Resizer Handle */}
                    <div className="resizer" onMouseDown={startResizing} />

                    {/* Lower: Terminal */}
                    <div className="terminal-pane" style={{ flex: 1, minHeight: 0 }}>
                        <GitTerminal />
                    </div>
                </div>
            </main>

            {/* RIGHT PANE: Object Inspector (1/4) */}
            <aside className="right-pane">
                <div className="pane-header">Object Inspector</div>
                <div className="pane-content">
                    <ObjectInspector selectedObject={selectedObject} />
                </div>
            </aside>
        </div>
    );
};

export default AppLayout;
