import { useState } from 'react';
import './AppLayout.css';
import { useGit } from '../../context/GitAPIContext';
import GitTerminal from '../terminal/GitTerminal';
import GitGraphViz from '../visualization/GitGraphViz';
import FileExplorer from './FileExplorer';
import ObjectInspector from './ObjectInspector';
import CloneModal from '../modals/CloneModal';

export interface SelectedObject {
    type: 'commit' | 'file';
    id: string; // Hash or Path
    data?: any;
}

const AppLayout = () => {
    const { showAllCommits, toggleShowAllCommits, runCommand } = useGit();
    const [selectedObject, setSelectedObject] = useState<SelectedObject | null>(null);
    const [isLeftPaneOpen, setIsLeftPaneOpen] = useState(true);
    const [isCloneModalOpen, setIsCloneModalOpen] = useState(false);

    const handleObjectSelect = (obj: SelectedObject) => {
        setSelectedObject(obj);
    };

    const handleClone = async (url: string) => {
        console.log("Cloning from:", url);
        // Execute git clone command
        // Note: The CloneCommand implementation expects "clone <url>"
        await runCommand(`clone ${url}`);
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
                    <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                        <span>Repository Visualization & Terminal</span>
                        <button
                            onClick={() => setIsCloneModalOpen(true)}
                            style={{
                                background: 'var(--accent-primary)',
                                color: 'white',
                                border: 'none',
                                borderRadius: '4px',
                                padding: '4px 8px',
                                fontSize: '0.75rem',
                                cursor: 'pointer',
                                fontWeight: 500
                            }}
                        >
                            Clone Repo
                        </button>
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
                    </div>
                </div>

                <div className="center-content">
                    {/* Upper: Visualization */}
                    <div className="viz-pane">
                        <GitGraphViz
                            onSelect={(commitData) => handleObjectSelect({ type: 'commit', id: commitData.id, data: commitData })}
                            selectedCommitId={selectedObject?.type === 'commit' ? selectedObject.id : undefined}
                        />
                    </div>

                    {/* Lower: Terminal */}
                    <div className="terminal-pane">
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

            {/* Modals */}
            <CloneModal
                isOpen={isCloneModalOpen}
                onClose={() => setIsCloneModalOpen(false)}
                onClone={handleClone}
            />
        </div>
    );
};

export default AppLayout;
