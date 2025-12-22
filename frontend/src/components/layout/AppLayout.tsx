import { useState, useRef, useEffect, useCallback, useMemo } from 'react';
import './AppLayout.css';
import { useGit } from '../../context/GitAPIContext';
import { useTheme } from '../../context/ThemeContext';
import GitTerminal from '../terminal/GitTerminal';
import GitGraphViz from '../visualization/GitGraphViz';
import GitReferenceList from '../visualization/GitReferenceList';
import BranchingStrategies from '../visualization/BranchingStrategies';
import FileExplorer from './FileExplorer';
import RemoteRepoView from './RemoteRepoView';
import type { GitState } from '../../types/gitTypes';

export interface SelectedObject {
    type: 'commit' | 'file';
    id: string; // Hash or Path
    data?: any;
}

type ViewMode = 'graph' | 'branches' | 'tags' | 'strategies';

const AppLayout = () => {
    const {
        state, showAllCommits, toggleShowAllCommits, isSandbox, isForking,
        enterSandbox, exitSandbox, resetSandbox,
        developers, activeDeveloper, switchDeveloper
    } = useGit();
    const { theme, toggleTheme } = useTheme();
    const [selectedObject, setSelectedObject] = useState<SelectedObject | null>(null);
    const [viewMode, setViewMode] = useState<ViewMode>('graph');


    const localState: GitState = useMemo(() => {
        return {
            ...state,
            remoteBranches: {} // Hide remote-tracking branches in local view
        };
    }, [state]);

    // --- Layout State ---
    // 3 Columns: Left (Remote), Center (Local), Right (Terminal)
    // We track Left and Center widths; Right takes remaining.
    const [columnWidths, setColumnWidths] = useState({ left: 33, center: 33 }); // Percentages
    const [vizHeight, setVizHeight] = useState(500); // Height of Top Graph in Center
    const [remoteGraphHeight, setRemoteGraphHeight] = useState(300); // Height of Top Graph in Left

    const containerRef = useRef<HTMLDivElement>(null);
    const centerContentRef = useRef<HTMLDivElement>(null);
    const leftContentRef = useRef<HTMLDivElement>(null);

    // Resize Refs
    const isResizingCol1 = useRef(false); // Between Left & Center
    const isResizingCol2 = useRef(false); // Between Center & Right
    const isResizingCenterVertical = useRef(false); // Center Pane Split
    const isResizingLeftVertical = useRef(false); // Left Pane Split

    // --- Resize Handlers ---

    const startResizeCol1 = useCallback(() => { isResizingCol1.current = true; document.body.style.cursor = 'col-resize'; document.body.style.userSelect = 'none'; }, []);
    const startResizeCol2 = useCallback(() => { isResizingCol2.current = true; document.body.style.cursor = 'col-resize'; document.body.style.userSelect = 'none'; }, []);
    const startResizeCenterVert = useCallback(() => { isResizingCenterVertical.current = true; document.body.style.cursor = 'row-resize'; document.body.style.userSelect = 'none'; }, []);
    const startResizeLeftVert = useCallback(() => { isResizingLeftVertical.current = true; document.body.style.cursor = 'row-resize'; document.body.style.userSelect = 'none'; }, []);

    const stopResizing = useCallback(() => {
        isResizingCol1.current = false;
        isResizingCol2.current = false;
        isResizingCenterVertical.current = false;
        isResizingLeftVertical.current = false;
        document.body.style.cursor = '';
        document.body.style.userSelect = '';
    }, []);

    const resize = useCallback((e: MouseEvent) => {
        if (!containerRef.current) return;
        const containerRect = containerRef.current.getBoundingClientRect();

        // Column 1 Resize (Left | Center)
        if (isResizingCol1.current) {
            const newLeftPct = ((e.clientX - containerRect.left) / containerRect.width) * 100;
            if (newLeftPct > 10 && newLeftPct < (100 - columnWidths.center - 10)) {
                const deltaPct = newLeftPct - columnWidths.left;
                setColumnWidths(prev => ({
                    left: newLeftPct,
                    center: prev.center - deltaPct
                }));
            }
        }

        // Column 2 Resize (Center | Right)
        if (isResizingCol2.current) {
            const newLeftPlusCenterPct = ((e.clientX - containerRect.left) / containerRect.width) * 100;
            const newCenterPct = newLeftPlusCenterPct - columnWidths.left;

            if (newCenterPct > 10 && newLeftPlusCenterPct < 90) {
                setColumnWidths(prev => ({
                    ...prev,
                    center: newCenterPct
                }));
            }
        }

        // Vertical Resize (Center Pane)
        if (isResizingCenterVertical.current && centerContentRef.current) {
            const rect = centerContentRef.current.getBoundingClientRect();
            const newH = e.clientY - rect.top;
            if (newH > 100 && newH < rect.height - 100) setVizHeight(newH);
        }

        // Vertical Resize (Left Pane)
        if (isResizingLeftVertical.current && leftContentRef.current) {
            const rect = leftContentRef.current.getBoundingClientRect();
            const newH = e.clientY - rect.top;
            if (newH > 100 && newH < rect.height - 100) setRemoteGraphHeight(newH);
        }

    }, [columnWidths]);

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

    const modes: ViewMode[] = ['graph', 'branches', 'tags', 'strategies'];

    return (
        <div className="layout-container" ref={containerRef} style={{ display: 'flex', width: '100vw', height: '100vh', overflow: 'hidden' }}>

            {/* --- COLUMN 1: REMOTE SERVER --- */}
            <aside
                className="left-pane"
                style={{ width: `${columnWidths.left}%`, display: 'flex', flexDirection: 'column', borderRight: '1px solid var(--border-subtle)' }}
                ref={leftContentRef}
            >
                {/* Header */}
                <div className="pane-header">SERVER (REMOTE)</div>

                {/* Content Split: Graph (Top) / Operations (Bottom) */}
                <div className="pane-content" style={{ display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
                    <RemoteRepoView
                        topHeight={remoteGraphHeight}
                        onResizeStart={startResizeLeftVert}
                    />
                </div>
            </aside>

            {/* Resizer 1 */}
            <div
                className="resizer-vertical"
                onMouseDown={startResizeCol1}
                style={{ cursor: 'col-resize', width: '4px', background: 'var(--border-subtle)', flexShrink: 0, zIndex: 10 }}
            />

            {/* --- COLUMN 2: LOCAL REPOSITORY --- */}
            <main
                className="center-pane"
                style={{ width: `${columnWidths.center}%`, display: 'flex', flexDirection: 'column', minWidth: 0 }}
            >
                {/* Header */}
                <div className="pane-header" style={{ justifyContent: 'space-between' }}>

                    <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
                        {/* View Mode Selectors */}
                        <div style={{ display: 'flex', gap: '4px', background: 'var(--bg-secondary)', padding: '2px', borderRadius: '6px', border: '1px solid var(--border-subtle)' }}>
                            {modes.map(mode => (
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
                                        fontWeight: 500,
                                        textTransform: 'capitalize'
                                    }}
                                >
                                    {mode}
                                </button>
                            ))}
                        </div>

                        {/* Developer Switcher */}
                        {developers.length > 0 && (
                            <div style={{ display: 'flex', gap: '4px', padding: '2px', background: 'rgba(0,0,0,0.1)', borderRadius: '6px' }}>
                                {developers.map(dev => (
                                    <button
                                        key={dev}
                                        onClick={() => switchDeveloper(dev)}
                                        style={{
                                            background: activeDeveloper === dev ? 'var(--bg-primary)' : 'transparent',
                                            color: activeDeveloper === dev ? 'var(--accent-primary)' : 'var(--text-tertiary)',
                                            border: 'none',
                                            borderRadius: '4px',
                                            padding: '4px 10px',
                                            fontSize: '11px',
                                            fontWeight: 700,
                                            cursor: 'pointer',
                                            transition: 'all 0.2s'
                                        }}
                                    >
                                        👤 {dev.toUpperCase()}
                                    </button>
                                ))}
                            </div>
                        )}
                    </div>

                    {/* Right Side Controls */}
                    <div style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
                        {/* Show All Toggle */}
                        <label style={{ display: 'flex', alignItems: 'center', cursor: 'pointer', gap: '6px', fontSize: '11px', color: 'var(--text-secondary)' }}>
                            <input
                                type="checkbox"
                                checked={showAllCommits}
                                onChange={toggleShowAllCommits}
                                style={{ accentColor: 'var(--accent-primary)', cursor: 'pointer' }}
                            />
                            Show All
                        </label>

                        {/* Sandbox */}
                        <button
                            onClick={isSandbox ? exitSandbox : enterSandbox}
                            disabled={isForking}
                            className={`sandbox-toggle ${isSandbox ? 'active' : ''}`}
                            style={{
                                background: isSandbox ? '#f59f00' : 'transparent',
                                color: isSandbox ? 'white' : 'var(--text-secondary)',
                                border: isSandbox ? '1px solid #f59f00' : '1px solid var(--border-subtle)',
                                borderRadius: '4px',
                                padding: '4px 8px',
                                fontSize: '12px',
                                fontWeight: 600,
                                cursor: 'pointer',
                                display: 'flex',
                                alignItems: 'center',
                                gap: '6px'
                            }}
                        >
                            {isForking ? '⏳...' : (isSandbox ? 'EXIT ' : 'SANDBOX')}
                        </button>
                        <button onClick={toggleTheme} style={{ background: 'none', border: 'none', cursor: 'pointer' }}>{theme === 'dark' ? '☀️' : '🌙'}</button>
                    </div>
                </div>

                {/* Sandbox Banner */}
                {isSandbox && (
                    <div style={{
                        background: '#f59f00',
                        color: 'white',
                        padding: '4px 12px',
                        fontSize: '12px',
                        fontWeight: 600,
                        textAlign: 'center',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        gap: '8px',
                        borderBottom: '1px solid rgba(0,0,0,0.1)'
                    }}>
                        <span>🏝️ SANDBOX MODE</span>
                        <button onClick={resetSandbox} disabled={isForking} style={{ background: 'rgba(255,255,255,0.2)', border: 'none', color: 'white', padding: '2px 8px', borderRadius: '4px', fontSize: '10px', cursor: 'pointer' }}>
                            {isForking ? 'RESETTING...' : 'RESET'}
                        </button>
                    </div>
                )}

                {/* Content Split: Graph (Top) / Explore (Bottom) */}
                <div className="center-content" ref={centerContentRef} style={{ display: 'flex', flexDirection: 'column', flex: 1, overflow: 'hidden' }}>
                    {/* Top: Graph */}
                    <div style={{ height: vizHeight, flex: 'none', position: 'relative', overflow: 'hidden', borderBottom: '1px solid var(--border-subtle)' }}>
                        {state.HEAD && state.HEAD.type !== 'none' || viewMode === 'strategies' ? (
                            viewMode === 'graph' ? (
                                <GitGraphViz
                                    title="LOCAL GRAPH"
                                    state={localState}
                                    onSelect={(commitData) => handleObjectSelect({ type: 'commit', id: commitData.id, data: commitData })}
                                    selectedCommitId={selectedObject?.type === 'commit' ? selectedObject.id : undefined}
                                />
                            ) : viewMode === 'strategies' ? (
                                <BranchingStrategies />
                            ) : (
                                <GitReferenceList
                                    type={viewMode === 'branches' ? 'branches' : 'tags'}
                                    onSelect={(commitData) => handleObjectSelect({ type: 'commit', id: commitData.id, data: commitData })}
                                    selectedCommitId={selectedObject?.type === 'commit' ? selectedObject.id : undefined}
                                />
                            )
                        ) : (
                            <div style={{ height: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center', color: 'var(--text-tertiary)' }}>No Repository Loaded</div>
                        )}
                    </div>

                    {/* Resizer Center Viz */}
                    <div className="resizer" onMouseDown={startResizeCenterVert} style={{ height: '4px', cursor: 'row-resize', background: 'var(--border-subtle)', width: '100%', zIndex: 10 }} />

                    {/* Bottom: Explore (File Explorer) */}
                    <div style={{ flex: 1, minHeight: 0, overflow: 'hidden' }}>
                        <FileExplorer onSelect={(fileObj: SelectedObject) => handleObjectSelect(fileObj)} />
                    </div>
                </div>
            </main>

            {/* Resizer 2 */}
            <div
                className="resizer-vertical"
                onMouseDown={startResizeCol2}
                style={{ cursor: 'col-resize', width: '4px', background: 'var(--border-subtle)', flexShrink: 0, zIndex: 10 }}
            />

            {/* --- COLUMN 3: TERMINAL --- */}
            <aside
                className="right-pane"
                style={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column', borderLeft: '1px solid var(--border-subtle)' }}
            >
                <div className="pane-header">TERMINAL</div>
                <div style={{ flex: 1, minHeight: 0, background: '#1e1e1e' }}>
                    <GitTerminal />
                </div>
            </aside>

        </div>
    );
};

export default AppLayout;
