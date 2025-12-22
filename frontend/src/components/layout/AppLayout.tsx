import { useState, useRef, useEffect, useCallback, useMemo } from 'react';
import './AppLayout.css';
import { useGit } from '../../context/GitAPIContext';
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
        state, showAllCommits, toggleShowAllCommits,
        developers, activeDeveloper, switchDeveloper
    } = useGit();

    // Theme removed from UI

    const [selectedObject, setSelectedObject] = useState<SelectedObject | null>(null);
    const [viewMode, setViewMode] = useState<ViewMode>('graph');


    const localState: GitState = useMemo(() => {
        return {
            ...state,
            remoteBranches: {} // Hide remote-tracking branches in local view
        };
    }, [state]);

    // --- Layout State ---
    // [Left Pane] | [Main Area (Shared Header + Split Content)]
    const [leftPaneWidth, setLeftPaneWidth] = useState(33); // Percentage
    // Within Main Area: [Center] | [Right]
    const [centerPaneWidth, setCenterPaneWidth] = useState(50); // Percentage of Main Area

    const [vizHeight, setVizHeight] = useState(500); // Height of Top Graph in Center
    const [remoteGraphHeight, setRemoteGraphHeight] = useState(500); // Height of Top Graph in Left (Synced with Center)

    const containerRef = useRef<HTMLDivElement>(null);
    const mainAreaRef = useRef<HTMLDivElement>(null);
    const centerContentRef = useRef<HTMLDivElement>(null);
    const leftContentRef = useRef<HTMLDivElement>(null);

    // Resize Refs
    const isResizingLeft = useRef(false); // Between Left & Main
    const isResizingRight = useRef(false); // Between Center & Right (inside Main)
    const isResizingCenterVertical = useRef(false); // Center Pane Split
    const isResizingLeftVertical = useRef(false); // Left Pane Split

    // --- Resize Handlers ---

    const startResizeLeft = useCallback(() => { isResizingLeft.current = true; document.body.style.cursor = 'col-resize'; document.body.style.userSelect = 'none'; }, []);
    const startResizeRight = useCallback(() => { isResizingRight.current = true; document.body.style.cursor = 'col-resize'; document.body.style.userSelect = 'none'; }, []);
    const startResizeCenterVert = useCallback(() => { isResizingCenterVertical.current = true; document.body.style.cursor = 'row-resize'; document.body.style.userSelect = 'none'; }, []);
    const startResizeLeftVert = useCallback(() => { isResizingLeftVertical.current = true; document.body.style.cursor = 'row-resize'; document.body.style.userSelect = 'none'; }, []);

    const stopResizing = useCallback(() => {
        isResizingLeft.current = false;
        isResizingRight.current = false;
        isResizingCenterVertical.current = false;
        isResizingLeftVertical.current = false;
        document.body.style.cursor = '';
        document.body.style.userSelect = '';
    }, []);

    const resize = useCallback((e: MouseEvent) => {
        if (!containerRef.current) return;
        const containerRect = containerRef.current.getBoundingClientRect();

        // 1. Resize Left Pane
        if (isResizingLeft.current) {
            const newLeftPct = ((e.clientX - containerRect.left) / containerRect.width) * 100;
            if (newLeftPct > 10 && newLeftPct < 90) {
                setLeftPaneWidth(newLeftPct);
            }
        }

        // 2. Resize Center vs Right (Inside Main Area)
        if (isResizingRight.current && mainAreaRef.current) {
            const mainRect = mainAreaRef.current.getBoundingClientRect();
            // Relative to Main Area
            const newCenterPct = ((e.clientX - mainRect.left) / mainRect.width) * 100;
            if (newCenterPct > 10 && newCenterPct < 90) {
                setCenterPaneWidth(newCenterPct);
            }
        }

        // 3. Vertical Resize (Center Pane)
        if (isResizingCenterVertical.current && centerContentRef.current) {
            const rect = centerContentRef.current.getBoundingClientRect();
            const newH = e.clientY - rect.top;
            if (newH > 100 && newH < rect.height - 100) setVizHeight(newH);
        }

        // 4. Vertical Resize (Left Pane)
        if (isResizingLeftVertical.current && leftContentRef.current) {
            const rect = leftContentRef.current.getBoundingClientRect();
            const newH = e.clientY - rect.top;
            if (newH > 100 && newH < rect.height - 100) setRemoteGraphHeight(newH);
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

    const modes: ViewMode[] = ['graph', 'branches', 'tags', 'strategies'];

    return (
        <div className="layout-container" ref={containerRef} style={{ display: 'flex', width: '100vw', height: '100vh', overflow: 'hidden' }}>

            {/* --- COLUMN 1: REMOTE SERVER --- */}
            <aside
                className="left-pane"
                style={{ width: `${leftPaneWidth}%`, display: 'flex', flexDirection: 'column', borderRight: '1px solid var(--border-subtle)' }}
                ref={leftContentRef}
            >
                {/* Header */}
                {/* Header Removed as per user request */}

                {/* Content Split: Graph (Top) / Operations (Bottom) */}
                <div className="pane-content" style={{ display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
                    <RemoteRepoView
                        topHeight={remoteGraphHeight}
                        onResizeStart={startResizeLeftVert}
                    />
                </div>
            </aside>

            {/* Main Resizer (Left vs Main) */}
            <div
                className="resizer-vertical"
                onMouseDown={startResizeLeft}
                style={{ cursor: 'col-resize', width: '4px', background: 'var(--border-subtle)', flexShrink: 0, zIndex: 20 }}
            />

            {/* --- MAIN AREA: SHARED HEADER + [Center | Right] --- */}
            <div ref={mainAreaRef} style={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column' }}>

                {/* SHARED HEADER */}
                <div className="pane-header" style={{ display: 'flex', justifyContent: 'space-between', borderBottom: '1px solid var(--border-subtle)', padding: '0 12px', height: '36px', alignItems: 'center', background: 'var(--bg-secondary)' }}>
                    {/* Controls Group */}
                    <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
                        {/* View Mode Selectors */}
                        <div style={{ display: 'flex', gap: '4px', background: 'var(--bg-tertiary)', padding: '2px', borderRadius: '6px', border: '1px solid var(--border-subtle)' }}>
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
                                        fontSize: '11px',
                                        cursor: 'pointer',
                                        fontWeight: 600,
                                        textTransform: 'capitalize'
                                    }}
                                >
                                    {mode}
                                </button>
                            ))}
                        </div>

                        {/* Developer Switcher */}
                        {developers.length > 0 && (
                            <div style={{ display: 'flex', gap: '2px', padding: '2px', background: 'rgba(0,0,0,0.1)', borderRadius: '6px' }}>
                                {developers.map(dev => (
                                    <button
                                        key={dev}
                                        onClick={() => switchDeveloper(dev)}
                                        style={{
                                            background: activeDeveloper === dev ? 'var(--bg-primary)' : 'transparent',
                                            color: activeDeveloper === dev ? 'var(--accent-primary)' : 'var(--text-tertiary)',
                                            border: 'none',
                                            borderRadius: '4px',
                                            padding: '4px 8px',
                                            fontSize: '10px',
                                            fontWeight: 700,
                                            cursor: 'pointer',
                                            display: 'flex',
                                            alignItems: 'center',
                                            gap: '4px',
                                            opacity: activeDeveloper === dev ? 1 : 0.6
                                        }}
                                    >
                                        👤 {dev.toUpperCase()}
                                    </button>
                                ))}
                            </div>
                        )}
                    </div>

                    {/* Right Side Controls & Banner */}
                    <div style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
                        {/* Show All Toggle */}
                        <label style={{ display: 'flex', alignItems: 'center', cursor: 'pointer', gap: '6px', fontSize: '10px', color: 'var(--text-secondary)', fontWeight: 500, textTransform: 'uppercase', letterSpacing: '0.05em' }}>
                            <input
                                type="checkbox"
                                checked={showAllCommits}
                                onChange={toggleShowAllCommits}
                                style={{ accentColor: 'var(--accent-primary)', cursor: 'pointer', width: '12px', height: '12px' }}
                            />
                            Show All
                        </label>

                        {/* Sandbox - REMOVED */}


                        {/* Terminal Label */}
                        <span style={{ fontSize: '10px', fontWeight: 800, color: 'var(--text-tertiary)', textTransform: 'uppercase', letterSpacing: '0.1em', paddingLeft: '12px', borderLeft: '1px solid var(--border-subtle)' }}>TERMINAL</span>
                    </div>
                </div>

                {/* Main Content Area (Split Center/Right) */}
                <div style={{ flex: 1, display: 'flex', minHeight: 0 }}>

                    {/* --- CENTER PANE (Local Graph & Files) --- */}
                    <main
                        style={{ width: `${centerPaneWidth}%`, display: 'flex', flexDirection: 'column', minWidth: 0, position: 'relative' }}
                    >
                        {/* Sandbox Banner - REMOVED */}

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

                    {/* Resizer 2 (Center vs Right) */}
                    <div
                        className="resizer-vertical"
                        onMouseDown={startResizeRight}
                        style={{ cursor: 'col-resize', width: '4px', background: 'var(--border-subtle)', flexShrink: 0, zIndex: 10 }}
                    />

                    {/* --- RIGHT PANE (Terminal) --- */}
                    <aside
                        style={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column', borderLeft: '1px solid var(--border-subtle)' }}
                    >
                        <div style={{ flex: 1, minHeight: 0, background: '#1e1e1e' }}>
                            <GitTerminal />
                        </div>
                    </aside>

                </div>

            </div>

        </div>
    );
};

export default AppLayout;
