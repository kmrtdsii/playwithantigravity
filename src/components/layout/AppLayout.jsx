import React, { useState } from 'react';
import { Terminal, GitGraph, Settings, Play, ShieldAlert } from 'lucide-react';
import './AppLayout.css';
import GitTerminal from '../terminal/GitTerminal';
import GitGraphViz from '../visualization/GitGraphViz';

const AppLayout = () => {
    const [activeTab, setActiveTab] = useState('sandbox');

    return (
        <div class="app-container">
            <nav class="sidebar">
                <div class="sidebar-header">
                    <Terminal size={24} color="var(--accent-primary)" />
                    <span>GitForge</span>
                </div>

                <div
                    class={`nav-item ${activeTab === 'sandbox' ? 'active' : ''}`}
                    onClick={() => setActiveTab('sandbox')}
                >
                    <GitGraph size={20} />
                    <span>Visual Sandbox</span>
                </div>

                <div
                    class={`nav-item ${activeTab === 'strategy' ? 'active' : ''}`}
                    onClick={() => setActiveTab('strategy')}
                >
                    <ShieldAlert size={20} />
                    <span>Strategy Sim</span>
                </div>

                <div
                    class={`nav-item ${activeTab === 'settings' ? 'active' : ''}`}
                    onClick={() => setActiveTab('settings')}
                >
                    <Settings size={20} />
                    <span>Settings</span>
                </div>
            </nav>

            <main class="main-content">
                <header class="top-bar">
                    <div style={{ fontWeight: 600 }}>Playground / Demo Project</div>
                    <button style={{
                        background: 'var(--accent-primary)',
                        color: 'white',
                        padding: '8px 16px',
                        borderRadius: '4px',
                        display: 'flex',
                        alignItems: 'center',
                        gap: '8px',
                        fontSize: '0.9rem'
                    }}>
                        <Play size={16} /> Run Scenario
                    </button>
                </header>

                <div class="workspace">
                    {/* Visualization Area */}
                    <div class="viz-panel">
                        <GitGraphViz />
                    </div>

                    {/* Terminal Area */}
                    <div class="terminal-panel">
                        <div class="panel-header">
                            <span class="panel-title">Terminal</span>
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
