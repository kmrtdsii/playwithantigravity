import { useState, useRef, useEffect, useCallback, useMemo } from 'react';
import './AppLayout.css';
import { useGit } from '../../context/GitAPIContext';
import GitTerminal from '../terminal/GitTerminal';
import GitGraphViz from '../visualization/GitGraphViz';
import GitReferenceList from '../visualization/GitReferenceList';
import BranchingStrategies from '../visualization/BranchingStrategies';
import FileExplorer from './FileExplorer';
import RemoteRepoView from './RemoteRepoView';
import DeveloperTabs from './DeveloperTabs';
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
        developers, activeDeveloper, switchDeveloper, addDeveloper
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
    const [leftPaneWidth, setLeftPaneWidth] = useState(33); // Percentage
    const [vizHeight, setVizHeight] = useState(500); // Height of Top Graph in Center
    const [remoteGraphHeight, setRemoteGraphHeight] = useState(500); // Height of Top Graph in Left (Synced with Center)

    const containerRef = useRef<HTMLDivElement>(null);
    const centerContentRef = useRef<HTMLDivElement>(null);
    const leftContentRef = useRef<HTMLDivElement>(null);

    // Resize Refs
    const isResizingLeft = useRef(false); // Between Left & Main
    const isResizingCenterVertical = useRef(false); // Center Pane Split
    const isResizingLeftVertical = useRef(false); // Left Pane Split

    // --- Resize Handlers ---

    const startResizeLeft = useCallback(() => { isResizingLeft.current = true; document.body.style.cursor = 'col-resize'; document.body.style.userSelect = 'none'; }, []);
    const startResizeCenterVert = useCallback(() => { isResizingCenterVertical.current = true; document.body.style.cursor = 'row-resize'; document.body.style.userSelect = 'none'; }, []);
    const startResizeLeftVert = useCallback(() => { isResizingLeftVertical.current = true; document.body.style.cursor = 'row-resize'; document.body.style.userSelect = 'none'; }, []);

    const stopResizing = useCallback(() => {
        isResizingLeft.current = false;
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

        // 2. Vertical Resize (Center Pane)
        if (isResizingCenterVertical.current && centerContentRef.current) {
            const rect = centerContentRef.current.getBoundingClientRect();
            const newH = e.clientY - rect.top;
            if (newH > 100 && newH < rect.height - 100) setVizHeight(newH);
        }

        // 3. Vertical Resize (Left Pane)
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
        <div className="layout-container" ref={containerRef} style={{ display: 'flex', width: '100vw', height: '100vh', overflow: 'hidden', background: '#0d1117' }}>

            {/* --- COLUMN 1: REMOTE SERVER --- */}
            <aside
                className="left-pane"
                style={{ width: `${leftPaneWidth}%`, display: 'flex', flexDirection: 'column', borderRight: '1px solid var(--border-subtle)' }}
                ref={leftContentRef}
            >
                <div className="pane-content" style={{ display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
                    <RemoteRepoView
                        topHeight={remoteGraphHeight}
                        onResizeStart={startResizeLeftVert}
                    />
                </div>
            </aside>

            {/* Main Resizer (Left vs Local) */}
            <div
                className="resizer-vertical"
                onMouseDown={startResizeLeft}
                style={{ cursor: 'col-resize', width: '4px', background: 'var(--border-subtle)', flexShrink: 0, zIndex: 20 }}
            />

            {/* --- COLUMN 2: LOCAL WORKSPACE (Merged Center & Right) --- */}
            <div style={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column' }}>

                {/* ROW 1: User Tabs (Alice / Bob) */}
                <DeveloperTabs
                    developers={developers}
                    activeDeveloper={activeDeveloper}
                    onSwitchDeveloper={switchDeveloper}
                    onAddDeveloper={() => {
                        const name = prompt('Name?');
                        if (name) addDeveloper(name);
                    }}
                />

                {/* ROW 2: View Toggles (Graph, Branches...) & Global Controls */}
                <div style={{
                    height: '40px',
                    background: '#1e1e1e', // Darker to separate from Tabs
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    padding: '0 16px',
                    borderBottom: '1px solid var(--border-subtle)'
                }}>
                    {/* View Modes */}
                    <div style={{ display: 'flex', gap: '8px' }}>
                        {modes.map(mode => (
                            <button
                                key={mode}
                                onClick={() => setViewMode(mode)}
                                style={{
                                    background: viewMode === mode ? 'var(--accent-primary)' : 'rgba(255,255,255,0.05)',
                                    color: viewMode === mode ? 'white' : 'var(--text-secondary)',
                                    border: '1px solid transparent',
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

                    {/* Right Side Controls */}
                    <div style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
                        <label style={{ display: 'flex', alignItems: 'center', cursor: 'pointer', gap: '6px', fontSize: '10px', color: 'var(--text-secondary)' }}>
                            <input
                                type="checkbox"
                                checked={showAllCommits}
                                onChange={toggleShowAllCommits}
                                style={{ accentColor: 'var(--accent-primary)' }}
                            />
                            SHOW ALL
                        </label>
                    </div>
                </div>

                {/* ROW 3: Stacked Content */}
                <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minHeight: 0 }}>

                    {/* Top: Graph / Visualization */}
                    <div ref={centerContentRef} style={{ height: vizHeight, minHeight: '100px', display: 'flex', flexDirection: 'column', borderBottom: '1px solid var(--border-subtle)' }}>
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

                    {/* Resizer Vert (Graph vs Bottom) */}
                    <div
                        className="resizer"
                        onMouseDown={startResizeCenterVert}
                        style={{ height: '4px', cursor: 'row-resize', background: 'var(--border-subtle)', width: '100%', zIndex: 10, flexShrink: 0 }}
                    />

                    {/* Bottom Area: Explorer | Terminal */}
                    <div style={{ flex: 1, minHeight: 0, display: 'flex' }}>

                        {/* File Explorer (Left side of Bottom) */}
                        <div style={{ width: '40%', display: 'flex', flexDirection: 'column', borderRight: '1px solid var(--border-subtle)' }}>
                            <FileExplorer onSelect={(fileObj: SelectedObject) => handleObjectSelect(fileObj)} />
                        </div>

                        {/* Terminal (Right side of Bottom) */}
                        <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
                            <GitTerminal />
                        </div>
                    </div>
                </div>

            </div>

        </div>
    );
};

export default AppLayout;
