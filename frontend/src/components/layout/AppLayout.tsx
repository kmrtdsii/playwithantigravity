import { useState } from 'react';
import './AppLayout.css';
import { useGit } from '../../context/GitAPIContext';

import GitGraphViz from '../visualization/GitGraphViz';
import GitReferenceList from '../visualization/GitReferenceList';

import RemoteRepoView from './RemoteRepoView';
import DeveloperTabs from './DeveloperTabs';
import BottomPanel from './BottomPanel';
import { Resizer } from '../common';
import AddDeveloperModal from './AddDeveloperModal'; // New Import

import type { SelectedObject } from '../../types/layoutTypes';
import { useTheme } from '../../context/ThemeContext';
import { useResizablePanes } from '../../hooks/useResizablePanes'; // New Hook

type ViewMode = 'graph' | 'branches' | 'tags';

const AppLayout = () => {
    const {
        state, showAllCommits, toggleShowAllCommits,
        developers, activeDeveloper, switchDeveloper, addDeveloper
    } = useGit();

    const { theme, toggleTheme } = useTheme();

    const [selectedObject, setSelectedObject] = useState<SelectedObject | null>(null);
    const [viewMode, setViewMode] = useState<ViewMode>('graph');

    // --- Layout State (Refactored) ---
    const {
        leftPaneWidth,
        vizHeight,
        remoteGraphHeight,
        containerRef,
        centerContentRef,
        stackContainerRef,
        leftContentRef,
        startResizeLeft,
        startResizeCenterVert,
        startResizeLeftVert
    } = useResizablePanes();

    // Modal State
    const [isAddDevModalOpen, setIsAddDevModalOpen] = useState(false);

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

            {/* --- Modals (Refactored) --- */}
            <AddDeveloperModal
                isOpen={isAddDevModalOpen}
                onClose={() => setIsAddDevModalOpen(false)}
                onAddDeveloper={addDeveloper}
            />
        </div>
    );
};

export default AppLayout;
