import { useState } from 'react';
import { Terminal, GitGraph, Settings, Play, ShieldAlert } from 'lucide-react';
import './AppLayout.css';
import GitTerminal from '../terminal/GitTerminal';
import GitGraphViz from '../visualization/GitGraphViz';

const AppLayout = () => {
    const [activeTab, setActiveTab] = useState('sandbox');

    return (
        <div className="app-container">
            <nav className="sidebar">
                <div className="sidebar-header">
                    <Terminal size={24} color="var(--accent-primary)" />
                    <span>GitForge</span>
                </div>

                <div
                    className={`nav-item ${activeTab === 'sandbox' ? 'active' : ''}`}
                    onClick={() => setActiveTab('sandbox')}
                >
                    <GitGraph size={20} />
                    <span>Visual Sandbox</span>
                </div>

                <div
                    className={`nav-item ${activeTab === 'strategy' ? 'active' : ''}`}
                    onClick={() => setActiveTab('strategy')}
                >
                    <ShieldAlert size={20} />
                    <span>Strategy Sim</span>
                </div>

                <div
                    className={`nav-item ${activeTab === 'settings' ? 'active' : ''}`}
                    onClick={() => setActiveTab('settings')}
                >
                    <Settings size={20} />
                    <span>Settings</span>
                </div>
            </nav>

            <main className="main-content">
                <header className="top-bar">
                    <div style={{ fontWeight: 600 }}>Playground / Demo Project</div>
                    <button style={{
                        background: 'var(--accent-primary)',
                        color: 'white',
                        padding: '8px 16px',
                        borderRadius: '4px',
                        display: 'flex',
                        alignItems: 'center',
                        gap: '8px',
                        fontSize: '0.9rem',
                        border: 'none',
                        cursor: 'pointer'
                    }}>
                        <Play size={16} /> Run Scenario
                    </button>
                </header>

                <div className="workspace">
                    {/* Visualization Area */}
                    <div className="viz-panel">
                        <GitGraphViz />
                    </div>

                    {/* Terminal Area */}
                    <div className="terminal-panel">
                        <div className="panel-header">
                            <span className="panel-title">Terminal</span>
                            <div style={{ display: 'flex', gap: '6px' }}>
                                <div style={{ width: 10, height: 10, borderRadius: '50%', background: '#ff5f56' }} />
                                <div style={{ width: 10, height: 10, borderRadius: '50%', background: '#ffbd2e' }} />
                                <div style={{ width: 10, height: 10, borderRadius: '50%', background: '#27c93f' }} />
                            </div>
                        </div>
                        <GitTerminal />
                    </div>
                </div>
            </main>
        </div>
    );
};

export default AppLayout;
