import { useState } from 'react';
import './AppLayout.css';
import { useGit } from '../../context/GitAPIContext';
import GitTerminal from '../terminal/GitTerminal';
import GitGraphViz from '../visualization/GitGraphViz';
import StageWorkingTree from './StageWorkingTree';
import ObjectInspector from './ObjectInspector';

export interface SelectedObject {
    type: 'commit' | 'file';
    id: string; // Hash or Path
    data?: any;
}

const AppLayout = () => {
    const { showAllCommits, toggleShowAllCommits } = useGit();
    const [selectedObject, setSelectedObject] = useState<SelectedObject | null>(null);

    const handleObjectSelect = (obj: SelectedObject) => {
        setSelectedObject(obj);
    };

    return (
        <div className="layout-container">
            {/* LEFT PANE: Stage & Working Tree (1/4) */}
            <aside className="left-pane">
                <div className="pane-header">Stage & Working Tree</div>
                <div className="pane-content">
                    <StageWorkingTree onSelect={(fileObj: SelectedObject) => handleObjectSelect(fileObj)} />
                </div>
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

                <div className="center-content">
                    {/* Upper: Visualization */}
                    <div className="viz-pane">
                        <GitGraphViz onSelect={(commitData) => handleObjectSelect({ type: 'commit', id: commitData.id, data: commitData })} />
                    </div>

                    {/* Lower: Terminal */}
                    <div className="terminal-pane">
                        {/* Terminal header removed as per plan to reduce clutter, relying on main header */}
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
