import React, { useState } from 'react';
import './AppLayout.css';
import { useGit } from '../../context/GitAPIContext';
import { useTranslation } from 'react-i18next';

import GitGraphViz from '../visualization/GitGraphViz';
import GitReferenceList from '../visualization/GitReferenceList';

import RemoteRepoView from './RemoteRepoView';
import DeveloperTabs from './DeveloperTabs';
import BottomPanel from './BottomPanel';
import { Resizer } from '../common';
import AddDeveloperModal from './AddDeveloperModal';
import MissionPanel from './MissionPanel';
import { Sun, Moon } from 'lucide-react';

import type { SelectedObject } from '../../types/layoutTypes';
import { useTheme } from '../../context/ThemeContext';
import { useResizablePanes } from '../../hooks/useResizablePanes';
import { motion, AnimatePresence } from 'framer-motion';

import CommitDetails from '../visualization/CommitDetails';
import SearchBar from '../common/SearchBar';
import SkillRadar from '../visualization/SkillRadar';

type ViewMode = 'graph' | 'branches' | 'tags';

const AppLayout = () => {
    const { t } = useTranslation('common'); // Hook

    const {
        state, showAllCommits, toggleShowAllCommits,
        developers, activeDeveloper, switchDeveloper, addDeveloper, removeDeveloper
    } = useGit();

    const { theme, toggleTheme } = useTheme();

    const [selectedObject, setSelectedObject] = useState<SelectedObject | null>(null);
    const [viewMode, setViewMode] = useState<ViewMode>('graph');
    const [detailsPaneWidth, setDetailsPaneWidth] = useState(300);
    const [searchQuery, setSearchQuery] = useState('');

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
    const [isSkillRadarOpen, setIsSkillRadarOpen] = useState(false);

    const handleObjectSelect = (obj: SelectedObject) => {
        setSelectedObject(obj);
    };

    // Auto-close details when repo is closed/unloaded
    // eslint-disable-next-line react-hooks/exhaustive-deps
    React.useEffect(() => {
        if (state.HEAD && state.HEAD.type === 'none') {
            setSelectedObject(null);
        }
    }, [state.HEAD?.type]);

    const startResizeDetails = (e: React.MouseEvent) => {
        e.preventDefault();
        const startX = e.clientX;
        const startWidth = detailsPaneWidth;

        const handleMouseMove = (mm: MouseEvent) => {
            const delta = startX - mm.clientX; // Dragging left increases width (since it's a right pane)
            setDetailsPaneWidth(Math.max(200, Math.min(800, startWidth + delta)));
        };

        const handleMouseUp = () => {
            document.removeEventListener('mousemove', handleMouseMove);
            document.removeEventListener('mouseup', handleMouseUp);
            document.body.style.cursor = 'default';
        };

        document.addEventListener('mousemove', handleMouseMove);
        document.addEventListener('mouseup', handleMouseUp);
        document.body.style.cursor = 'col-resize';
    };

    const modes: ViewMode[] = ['graph', 'branches', 'tags'];

    return (
        <div className="layout-container" ref={containerRef} style={{ display: 'flex', width: '100vw', height: '100vh', overflow: 'hidden', background: 'var(--bg-primary)' }}>

            {/* --- COLUMN 1: REMOTE SERVER --- */}
            <aside
                className="left-pane"
                style={{ width: `${leftPaneWidth}% `, display: 'flex', flexDirection: 'column', borderRight: '1px solid var(--border-subtle)' }}
                ref={leftContentRef}
                data-testid="layout-left-pane"
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
            <div style={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column' }} data-testid="layout-workspace-pane">

                {/* ROW 1: User Tabs (Alice / Bob) */}
                <DeveloperTabs
                    developers={developers}
                    activeDeveloper={activeDeveloper}
                    onSwitchDeveloper={switchDeveloper}
                    onAddDeveloper={() => setIsAddDevModalOpen(true)}
                    onRemoveDeveloper={removeDeveloper}
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
                    <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
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
                                    fontWeight: 600
                                }}
                                data-testid={`view-mode-${mode}`}
                            >
                                {t(`viewMode.${mode}`)}
                            </button>
                        ))}

                        {/* Skill Radar Button */}
                        <button
                            onClick={() => setIsSkillRadarOpen(true)}
                            style={{
                                background: 'transparent',
                                border: '1px solid var(--accent-primary)',
                                color: 'var(--accent-primary)',
                                borderRadius: '4px',
                                padding: '4px 12px',
                                fontSize: '11px',
                                cursor: 'pointer',
                                fontWeight: 600,
                                marginLeft: '8px',
                                display: 'flex',
                                alignItems: 'center',
                                gap: '4px'
                            }}
                            title="Open Git Skill Tree"
                        >
                            <span>{t('app.skills')}</span>
                        </button>
                    </div>

                    {/* Search Bar */}
                    <div style={{ flex: 1, display: 'flex', justifyContent: 'center' }}>
                        <SearchBar value={searchQuery} onChange={setSearchQuery} placeholder={t('app.searchPlaceholder')} />
                    </div>

                    {/* Right Side Controls */}
                    <div style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
                        <label style={{ display: 'flex', alignItems: 'center', cursor: 'pointer', gap: '6px', fontSize: '10px', color: 'var(--text-secondary)' }} data-testid="show-all-toggle">
                            <input
                                type="checkbox"
                                checked={showAllCommits}
                                onChange={toggleShowAllCommits}
                                style={{ accentColor: 'var(--accent-primary)' }}
                            />
                            {t('common.showAll')}
                        </label>
                        {/* Theme Toggle - Segmented Switch */}
                        <div className="theme-toggle-group">
                            <button
                                onClick={() => theme === 'dark' && toggleTheme()}
                                className={`theme-toggle-option ${theme === 'light' ? 'active' : ''}`}
                                data-testid="theme-light-btn"
                                title={t('app.theme.light')}
                            >
                                <Sun size={12} strokeWidth={2.5} />
                                <span>{t('app.theme.light')}</span>
                            </button>
                            <button
                                onClick={() => theme === 'light' && toggleTheme()}
                                className={`theme-toggle-option ${theme === 'dark' ? 'active' : ''}`}
                                data-testid="theme-dark-btn"
                                title={t('app.theme.dark')}
                            >
                                <Moon size={12} strokeWidth={2.5} />
                                <span>{t('app.theme.dark')}</span>
                            </button>
                        </div>
                    </div>
                </div>

                {/* ROW 3: Stacked Content */}
                <div ref={stackContainerRef} style={{ flex: 1, display: 'flex', flexDirection: 'column', minHeight: 0 }}>

                    {/* Top: Graph / Visualization */}
                    <div ref={centerContentRef} style={{ height: vizHeight, minHeight: '100px', display: 'flex', borderBottom: '1px solid var(--border-subtle)', overflow: 'hidden', position: 'relative' }}>
                        <div style={{ flex: 1, height: '100%', position: 'relative', overflow: 'hidden' }}>
                            <AnimatePresence mode="wait">
                                {state.HEAD && state.HEAD.type !== 'none' ? (
                                    viewMode === 'graph' ? (
                                        <motion.div
                                            key="graph"
                                            initial={{ opacity: 0, x: -10 }}
                                            animate={{ opacity: 1, x: 0 }}
                                            exit={{ opacity: 0, x: 10 }}
                                            transition={{ duration: 0.2 }}
                                            style={{ width: '100%', height: '100%' }}
                                        >
                                            <GitGraphViz
                                                // state={state} // Use context state to show all branches including remotes
                                                onSelect={(commitData) => handleObjectSelect({ type: 'commit', id: commitData.id, data: commitData })}
                                                selectedCommitId={selectedObject?.type === 'commit' ? selectedObject.id : undefined}
                                                searchQuery={searchQuery}
                                            />
                                        </motion.div>
                                    ) : (
                                        <motion.div
                                            key={viewMode}
                                            initial={{ opacity: 0, x: -10 }}
                                            animate={{ opacity: 1, x: 0 }}
                                            exit={{ opacity: 0, x: 10 }}
                                            transition={{ duration: 0.2 }}
                                            style={{ width: '100%', height: '100%' }}
                                        >
                                            <GitReferenceList
                                                type={viewMode === 'branches' ? 'branches' : 'tags'}
                                                onSelect={(commitData) => handleObjectSelect({ type: 'commit', id: commitData.id, data: commitData })}
                                                selectedCommitId={selectedObject?.type === 'commit' ? selectedObject.id : undefined}
                                            />
                                        </motion.div>
                                    )
                                ) : (
                                    <motion.div
                                        key="empty"
                                        initial={{ opacity: 0 }}
                                        animate={{ opacity: 1 }}
                                        exit={{ opacity: 0 }}
                                        style={{ height: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center', color: 'var(--text-tertiary)' }}
                                    >
                                        {t('common.noRepoLoaded')}
                                    </motion.div>
                                )}
                            </AnimatePresence>
                        </div>

                        {/* RIGHT PANE: Commit Details */}
                        {selectedObject?.type === 'commit' && (
                            <>
                                <Resizer orientation="vertical" onMouseDown={startResizeDetails} />
                                <div style={{ width: `${detailsPaneWidth}px`, flexShrink: 0 }}>
                                    <CommitDetails
                                        commitId={selectedObject.id}
                                        onClose={() => setSelectedObject(null)}
                                    />
                                </div>
                            </>
                        )}
                    </div>

                    {/* Resizer Vert (Graph vs Bottom) */}
                    <Resizer orientation="horizontal" onMouseDown={startResizeCenterVert} />

                    {/* Bottom Area: Explorer | Terminal (Custom Resizable) */}
                    <BottomPanel onSelect={(fileObj: SelectedObject) => handleObjectSelect(fileObj)} />

                </div>

                {/* --- Modals (Refactored) --- */}
                <AddDeveloperModal
                    isOpen={isAddDevModalOpen}
                    onClose={() => setIsAddDevModalOpen(false)}
                    onAddDeveloper={addDeveloper}
                />

                <SkillRadar
                    isOpen={isSkillRadarOpen}
                    onClose={() => setIsSkillRadarOpen(false)}
                />

                <MissionPanel />
            </div>
        </div>
    );
};

export default AppLayout;
