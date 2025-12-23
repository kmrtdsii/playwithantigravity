import { useState, useRef, useCallback, useEffect } from 'react';
import './AppLayout.css';
import { useGit } from '../../context/GitAPIContext';

import GitGraphViz from '../visualization/GitGraphViz';
import GitReferenceList from '../visualization/GitReferenceList';

import RemoteRepoView from './RemoteRepoView';
import DeveloperTabs from './DeveloperTabs';
import BottomPanel from './BottomPanel';
import { Modal, Resizer } from '../common';

import type { SelectedObject } from '../../types/layoutTypes';
import { useTheme } from '../../context/ThemeContext';

type ViewMode = 'graph' | 'branches' | 'tags';

const AppLayout = () => {
    const {
        state, showAllCommits, toggleShowAllCommits,
        developers, activeDeveloper, switchDeveloper, addDeveloper
    } = useGit();

    const { theme, toggleTheme } = useTheme();

    const [selectedObject, setSelectedObject] = useState<SelectedObject | null>(null);
    const [viewMode, setViewMode] = useState<ViewMode>('graph');




    // --- Layout State ---
    const [leftPaneWidth, setLeftPaneWidth] = useState(33); // Percentage
    const [vizHeight, setVizHeight] = useState(500); // Height of Top Graph in Center
    const [remoteGraphHeight, setRemoteGraphHeight] = useState(500); // Height of Top Graph in Left

    // Modal State
    const [isAddDevModalOpen, setIsAddDevModalOpen] = useState(false);
    const [newDevName, setNewDevName] = useState('');

    const containerRef = useRef<HTMLDivElement>(null);
    const centerContentRef = useRef<HTMLDivElement>(null);
    const stackContainerRef = useRef<HTMLDivElement>(null); // Parent of graph + bottom
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

        // 2. Vertical Resize (Center Pane) - use stack container for full height
        if (isResizingCenterVertical.current && stackContainerRef.current) {
            const rect = stackContainerRef.current.getBoundingClientRect();
            const newH = e.clientY - rect.top;
            const minHeight = 100;
            const maxHeight = rect.height - 100; // Leave room for bottom panel
            if (newH >= minHeight && newH <= maxHeight) setVizHeight(newH);
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

    const modes: ViewMode[] = ['graph', 'branches', 'tags'];

    return (
        <div className="layout-container" ref={containerRef} style={{ display: 'flex', width: '100vw', height: '100vh', overflow: 'hidden', background: 'var(--bg-primary)' }}>

            {/* --- COLUMN 1: REMOTE SERVER --- */}
            <aside
                className="left-pane"
                style={{ width: `${leftPaneWidth}% `, display: 'flex', flexDirection: 'column', borderRight: '1px solid var(--border-subtle)' }}
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
            <Resizer orientation="vertical" onMouseDown={startResizeLeft} />

            {/* --- COLUMN 2: LOCAL WORKSPACE (Merged Center & Right) --- */}
            <div style={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column' }}>

                {/* ROW 1: User Tabs (Alice / Bob) */}
                <DeveloperTabs
                    developers={developers}
                    activeDeveloper={activeDeveloper}
                    onSwitchDeveloper={switchDeveloper}
                    onAddDeveloper={() => setIsAddDevModalOpen(true)}
                />

                {/* ROW 2: View Toggles (Graph, Branches...) & Global Controls */}
                <div style={{
                    height: '40px',
                    background: 'var(--bg-toolbar)', // Matches active tab
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
                                    background: viewMode === mode ? 'var(--accent-primary)' : 'var(--bg-button-inactive)',
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
                        {/* Theme Toggle */}
                        <button
                            onClick={toggleTheme}
                            style={{
                                background: 'transparent',
                                border: '1px solid var(--border-subtle)',
                                borderRadius: '4px',
                                padding: '4px 8px',
                                fontSize: '10px',
                                cursor: 'pointer',
                                color: 'var(--text-secondary)',
                                display: 'flex',
                                alignItems: 'center',
                                gap: '4px'
                            }}
                            title={theme === 'dark' ? 'Switch to Light Mode' : 'Switch to Dark Mode'}
                        >
                            {theme === 'dark' ? '☀️' : '🌙'}
                        </button>
                    </div>
                </div>

                {/* ROW 3: Stacked Content */}
                <div ref={stackContainerRef} style={{ flex: 1, display: 'flex', flexDirection: 'column', minHeight: 0 }}>

                    {/* Top: Graph / Visualization */}
                    <div ref={centerContentRef} style={{ height: vizHeight, minHeight: '100px', display: 'flex', flexDirection: 'column', borderBottom: '1px solid var(--border-subtle)' }}>
                        {state.HEAD && state.HEAD.type !== 'none' ? (
                            viewMode === 'graph' ? (
                                <GitGraphViz
                                    // state={state} // Use context state to show all branches including remotes
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
                            <div style={{ height: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center', color: 'var(--text-tertiary)' }}>No Repository Loaded</div>
                        )}
                    </div>

                    {/* Resizer Vert (Graph vs Bottom) */}
                    <Resizer orientation="horizontal" onMouseDown={startResizeCenterVert} />

                    {/* Bottom Area: Explorer | Terminal (Custom Resizable) */}
                    <BottomPanel onSelect={(fileObj: SelectedObject) => handleObjectSelect(fileObj)} />

                </div>

            </div>



            {/* --- Modals --- */}
            <Modal
                isOpen={isAddDevModalOpen}
                onClose={() => setIsAddDevModalOpen(false)}
                title="Add New Developer"
            >
                <form
                    onSubmit={(e) => {
                        e.preventDefault();
                        const formData = new FormData(e.currentTarget);
                        const name = formData.get('name') as string;
                        if (name && name.trim()) {
                            addDeveloper(name.trim());
                            setIsAddDevModalOpen(false);
                            setNewDevName('');
                        }
                    }}
                    style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}
                >
                    <input
                        name="name"
                        value={newDevName}
                        onChange={(e) => setNewDevName(e.target.value)}
                        placeholder="Enter developer name (e.g., Alice)"
                        autoFocus
                        style={{
                            padding: '8px 12px',
                            borderRadius: '4px',
                            border: '1px solid var(--border-subtle)',
                            background: 'var(--bg-primary)',
                            color: 'var(--text-primary)',
                            fontSize: '14px'
                        }}
                    />
                    <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '8px' }}>
                        <button
                            type="button"
                            onClick={() => setIsAddDevModalOpen(false)}
                            style={{
                                padding: '8px 16px',
                                borderRadius: '4px',
                                border: '1px solid var(--border-subtle)',
                                background: 'transparent',
                                color: 'var(--text-secondary)',
                                cursor: 'pointer'
                            }}
                        >
                            Cancel
                        </button>
                        <button
                            type="submit"
                            style={{
                                padding: '8px 16px',
                                borderRadius: '4px',
                                border: 'none',
                                background: 'var(--accent-primary)',
                                color: 'white',
                                cursor: 'pointer',
                                fontWeight: 500
                            }}
                        >
                            Add Developer
                        </button>
                    </div>
                </form>
            </Modal>
        </div>
    );
};

export default AppLayout;
