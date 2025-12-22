import React, { useMemo } from 'react';
import { useGit } from '../../context/GitAPIContext';
import GitGraphViz from '../visualization/GitGraphViz';
import type { GitState } from '../../types/gitTypes';

interface RemoteRepoViewProps {
    topHeight: number;
    onResizeStart: () => void;
}

// Styles
const containerStyle: React.CSSProperties = {
    height: '100%',
    display: 'flex',
    flexDirection: 'column',
    background: 'var(--bg-primary)',
    overflow: 'hidden'
};

const sectionLabelStyle: React.CSSProperties = {
    fontSize: '0.7rem',
    fontWeight: 800,
    color: 'var(--text-tertiary)',
    textTransform: 'uppercase',
    letterSpacing: '0.1em'
};

const cardStyle: React.CSSProperties = {
    background: 'var(--bg-secondary)',
    borderRadius: '12px',
    padding: '12px 16px',
    border: '1px solid var(--border-subtle)',
    boxShadow: '0 4px 12px rgba(0,0,0,0.1)'
};

const actionButtonStyle: React.CSSProperties = {
    background: 'var(--accent-primary)',
    color: 'white',
    border: 'none',
    borderRadius: '4px',
    padding: '4px 10px',
    fontSize: '10px',
    fontWeight: 700,
    cursor: 'pointer'
};

const prCardStyle: React.CSSProperties = {
    display: 'flex',
    flexDirection: 'column',
    gap: '4px',
    padding: '10px 12px',
    background: 'var(--bg-secondary)',
    borderRadius: '10px',
    border: '1px solid var(--border-subtle)',
};

const mergeButtonStyle: React.CSSProperties = {
    width: '100%',
    padding: '5px',
    background: '#8957e5',
    color: 'white',
    border: 'none',
    borderRadius: '6px',
    fontSize: '11px',
    fontWeight: 700,
    cursor: 'pointer',
    marginTop: '4px'
};

const emptyStyle: React.CSSProperties = {
    fontSize: '0.8rem',
    color: 'var(--text-tertiary)',
    fontStyle: 'italic',
    padding: '12px',
    border: '1px dashed var(--border-subtle)',
    borderRadius: '10px',
    textAlign: 'center'
};

const RemoteRepoView: React.FC<RemoteRepoViewProps> = ({ topHeight, onResizeStart }) => {
    const { state, pullRequests, mergePullRequest, refreshPullRequests, createPullRequest, ingestRemote, addDeveloper, resetRemote, runCommand, refreshState } = useGit();
    const { remoteBranches } = state;

    const remoteState: GitState = useMemo(() => {
        const transformedBranches: Record<string, string> = {};
        Object.entries(state.remoteBranches).forEach(([name, hash]) => {
            const parts = name.split('/');
            const shortName = parts.length > 1 ? parts.slice(1).join('/') : name;
            transformedBranches[shortName] = hash;
        });

        let remoteHead: GitState['HEAD'] = { type: 'none', ref: null };
        const headCandidate = transformedBranches['main'] || transformedBranches['master'] || Object.keys(transformedBranches)[0];
        if (headCandidate) {
            remoteHead = {
                type: 'branch',
                ref: transformedBranches['main'] ? 'main' : (transformedBranches['master'] ? 'master' : Object.keys(transformedBranches)[0]),
                id: transformedBranches[transformedBranches['main'] ? 'main' : (transformedBranches['master'] ? 'master' : Object.keys(transformedBranches)[0])]
            };
        }

        return {
            ...state,
            branches: transformedBranches,
            HEAD: remoteHead,
            potentialCommits: [],
            remoteBranches: {},
            staging: [],
            modified: [],
            untracked: [],
            fileStatuses: {}
        };
    }, [state]);

    const [setupUrl, setSetupUrl] = React.useState('');
    const [isSettingUp, setIsSettingUp] = React.useState(false);
    const [setupLog, setSetupLog] = React.useState<string[]>([]);
    const [setupSuccess, setSetupSuccess] = React.useState(false);
    const [isEditMode, setIsEditMode] = React.useState(false);

    React.useEffect(() => {
        refreshPullRequests();
    }, []);

    // Reset setupSuccess when sharedRemotes is populated
    React.useEffect(() => {
        if (state.sharedRemotes && state.sharedRemotes.length > 0 && setupSuccess) {
            setSetupSuccess(false);
        }
    }, [state.sharedRemotes, setupSuccess]);

    const handleInitialSetup = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!setupUrl) return;
        setIsSettingUp(true);
        setSetupLog(['Connecting to server...']);

        try {
            await new Promise(resolve => setTimeout(resolve, 800));
            setSetupLog(prev => [...prev, 'Ingesting remote repository from GitHub...']);
            await ingestRemote('origin', setupUrl);

            await new Promise(resolve => setTimeout(resolve, 1000));
            setSetupLog(prev => [...prev, 'Cloning to local environment...']);

            // Auto-clone the project
            // Extract project name from URL to use as folder name
            const projectName = setupUrl.split('/').pop()?.replace('.git', '') || 'my-project';
            await runCommand(`git clone ${setupUrl} ${projectName}`); // Clone into a folder
            await runCommand(`cd ${projectName}`); // Switch to it (simulated in frontend context contextually via active path usually, but here just CLI)

            await addDeveloper('Alice');
            await addDeveloper('Bob');

            await refreshState(); // Ensure UI updates with new files/graph

            await new Promise(resolve => setTimeout(resolve, 800));
            setSetupLog(prev => [...prev, 'Setup complete!']);
            setSetupSuccess(true);

            await new Promise(resolve => setTimeout(resolve, 1200));
        } catch (e) {
            console.error(e);
            setSetupLog(prev => [...prev, 'Error: Failed to initialize remote.']);
        } finally {
            setIsSettingUp(false);
        }
    };

    const handleEditRemote = () => {
        setSetupUrl(state.remotes?.[0]?.urls?.[0] || '');
        setIsEditMode(true);
    };

    const handleCancelEdit = () => {
        setIsEditMode(false);
        setSetupUrl('');
    };

    const handleResetRemote = async () => {
        if (!confirm('本当にリモートリポジトリをリセットしますか？全てのリモートデータと関連するPRが失われます。')) {
            return;
        }
        try {
            await resetRemote('origin');
            setIsEditMode(false);
            setSetupUrl('');
        } catch (e) {
            console.error('Failed to reset remote:', e);
            alert('リモートのリセットに失敗しました');
        }
    };

    const handleCreatePR = () => {
        const title = prompt('PR Title');
        const source = prompt('Source Branch', 'feature');
        const target = prompt('Target Branch', 'main');
        if (title && source && target) {
            createPullRequest(title, '', source, target);
        }
    };

    const hasSharedRemotes = state.sharedRemotes && state.sharedRemotes.length > 0;

    // Show setup form
    if (!hasSharedRemotes || isEditMode) {
        return (
            <div style={{ padding: '16px', height: '100%', overflowY: 'auto' }}>
                <div style={{ ...cardStyle, background: 'linear-gradient(to bottom, var(--bg-secondary), var(--bg-primary))', textAlign: 'center', padding: '24px 16px' }}>
                    {setupSuccess ? (
                        <div style={{ padding: '20px' }}>
                            <div style={{ fontSize: '48px', marginBottom: '16px' }}>🎉</div>
                            <div style={{ fontSize: '1.1rem', fontWeight: 800, color: 'var(--accent-primary)', marginBottom: '8px' }}>INITIALIZED!</div>
                            <p style={{ fontSize: '0.85rem', color: 'var(--text-secondary)' }}>
                                Remote repository is ready. Alice and Bob are online.
                            </p>
                        </div>
                    ) : isSettingUp ? (
                        <div style={{ padding: '10px 0' }}>
                            <div className="spinner" style={{
                                width: '30px',
                                height: '30px',
                                border: '3px solid var(--border-subtle)',
                                borderTop: '3px solid var(--accent-primary)',
                                borderRadius: '50%',
                                margin: '0 auto 20px',
                                animation: 'spin 1s linear infinite'
                            }} />
                            <div style={{ textAlign: 'left', background: 'rgba(0,0,0,0.2)', padding: '12px', borderRadius: '8px', border: '1px solid var(--border-subtle)' }}>
                                {setupLog.map((log, i) => (
                                    <div key={i} style={{
                                        fontSize: '11px',
                                        fontFamily: 'monospace',
                                        color: i === setupLog.length - 1 ? 'var(--text-primary)' : 'var(--text-tertiary)',
                                        marginBottom: '4px',
                                        display: 'flex',
                                        gap: '8px'
                                    }}>
                                        <span style={{ color: 'var(--accent-primary)' }}>{i === setupLog.length - 1 ? '➜' : '✓'}</span>
                                        {log}
                                    </div>
                                ))}
                            </div>
                        </div>
                    ) : (
                        <>
                            <div style={{ fontSize: '24px', marginBottom: '12px' }}>{isEditMode ? '⚙️' : '🤝'}</div>
                            <div style={{ fontSize: '0.9rem', fontWeight: 800, marginBottom: '8px' }}>
                                {isEditMode ? 'CHANGE REMOTE' : 'TEAM MODE SETUP'}
                            </div>
                            <p style={{ fontSize: '0.75rem', color: 'var(--text-secondary)', marginBottom: '16px', lineHeight: '1.4' }}>
                                {isEditMode
                                    ? 'Update the remote repository URL or reset to start fresh.'
                                    : 'Initialize a shared remote to start collaborative simulation.'}
                            </p>
                            <form onSubmit={handleInitialSetup}>
                                <input
                                    type="text"
                                    placeholder="GitHub Repository URL"
                                    value={setupUrl}
                                    onChange={(e) => setSetupUrl(e.target.value)}
                                    style={{
                                        width: '100%',
                                        padding: '8px 12px',
                                        background: 'var(--bg-primary)',
                                        border: '1px solid var(--border-subtle)',
                                        borderRadius: '6px',
                                        color: 'var(--text-primary)',
                                        fontSize: '12px',
                                        marginBottom: '10px',
                                        outline: 'none'
                                    }}
                                />
                                <div style={{ display: 'flex', gap: '8px' }}>
                                    {isEditMode && (
                                        <button
                                            type="button"
                                            onClick={handleCancelEdit}
                                            style={{
                                                flex: 1,
                                                padding: '8px',
                                                background: 'var(--bg-tertiary)',
                                                color: 'var(--text-secondary)',
                                                border: '1px solid var(--border-subtle)',
                                                borderRadius: '6px',
                                                fontSize: '12px',
                                                fontWeight: 700,
                                                cursor: 'pointer'
                                            }}
                                        >
                                            Cancel
                                        </button>
                                    )}
                                    <button
                                        type="submit"
                                        disabled={isSettingUp || !setupUrl}
                                        style={{
                                            flex: 1,
                                            padding: '8px',
                                            background: 'var(--accent-primary)',
                                            color: 'white',
                                            border: 'none',
                                            borderRadius: '6px',
                                            fontSize: '12px',
                                            fontWeight: 700,
                                            cursor: 'pointer',
                                            opacity: (isSettingUp || !setupUrl) ? 0.5 : 1
                                        }}
                                    >
                                        {isEditMode ? 'UPDATE REMOTE' : 'INITIALIZE REMOTE'}
                                    </button>
                                </div>
                            </form>
                            {isEditMode && (
                                <button
                                    onClick={handleResetRemote}
                                    style={{
                                        marginTop: '16px',
                                        fontSize: '10px',
                                        color: '#f85149',
                                        background: 'transparent',
                                        border: 'none',
                                        cursor: 'pointer',
                                        textDecoration: 'underline'
                                    }}
                                >
                                    🗑️ Reset Remote (Dangerous)
                                </button>
                            )}
                        </>
                    )}
                </div>
            </div>
        );
    }

    const remoteUrl = state.remotes?.[0]?.urls?.[0] || '';
    // Simply extract last part of path as project name
    const projectName = remoteUrl.split('/').pop()?.replace('.git', '') || 'Remote Repository';
    const remoteName = state.remotes?.[0]?.name || 'origin';

    return (
        <div style={containerStyle}>
            {/* TOP SPLIT: Info & Graph (Height synced via props) */}
            <div style={{ height: topHeight, display: 'flex', flexDirection: 'column', flexShrink: 0, minHeight: 0 }}>
                {/* Repo Info Header - ENHANCED */}
                <div style={{
                    padding: '12px',
                    background: 'var(--bg-secondary)',
                    borderBottom: '1px solid var(--border-subtle)',
                    display: 'flex',
                    alignItems: 'flex-start',
                    justifyContent: 'space-between',
                    flexShrink: 0
                }}>
                    <div style={{ display: 'flex', flexDirection: 'column', gap: '2px', overflow: 'hidden' }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                            <span style={{ fontSize: '1.0rem', fontWeight: 800, color: 'var(--text-primary)' }}>{projectName}</span>
                            <span style={{
                                fontSize: '0.65rem',
                                padding: '1px 6px',
                                borderRadius: '10px',
                                background: 'var(--bg-tertiary)',
                                color: 'var(--text-tertiary)',
                                border: '1px solid var(--border-subtle)'
                            }}>
                                {remoteName}
                            </span>
                        </div>
                        <div style={{
                            fontSize: '0.7rem',
                            color: 'var(--text-tertiary)',
                            fontFamily: 'monospace',
                            whiteSpace: 'nowrap',
                            overflow: 'hidden',
                            textOverflow: 'ellipsis',
                            maxWidth: '100%'
                        }} title={remoteUrl}>
                            {remoteUrl || 'No URL configured'}
                        </div>
                    </div>
                    <button
                        onClick={handleEditRemote}
                        style={{
                            background: 'none',
                            border: 'none',
                            color: 'var(--text-tertiary)',
                            cursor: 'pointer',
                            fontSize: '14px',
                            padding: '0 0 0 8px'
                        }}
                        title="Configure Remote"
                    >
                        ⚙️
                    </button>
                </div>

                {/* Compact Remote Graph */}
                <div style={{ flex: 1, minHeight: 0, overflow: 'hidden', position: 'relative' }}>
                    <GitGraphViz
                        state={remoteState}
                        title=""
                    />
                </div>
            </div>

            {/* RESIZER HANDLE */}
            <div
                className="resizer"
                onMouseDown={onResizeStart}
                style={{
                    height: '4px',
                    background: 'var(--border-subtle)',
                    cursor: 'row-resize',
                    zIndex: 10,
                    flexShrink: 0
                }}
            />

            {/* BOTTOM SPLIT: PRs & Branches - Takes Remaining Space */}
            <div style={{ flex: 1, overflowY: 'auto', display: 'flex', flexDirection: 'column', gap: '12px', padding: '12px' }}>

                {/* Pull Requests Dashboard */}
                <div>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '8px' }}>
                        <div style={sectionLabelStyle}>PULL REQUESTS</div>
                        <button
                            onClick={handleCreatePR}
                            style={{ ...actionButtonStyle, fontSize: '9px', padding: '3px 8px' }}
                        >
                            + NEW
                        </button>
                    </div>
                    {pullRequests.length > 0 ? (
                        <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                            {pullRequests.map(pr => (
                                <div key={pr.id} style={prCardStyle}>
                                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '2px' }}>
                                        <span style={{ fontWeight: 700, fontSize: '0.75rem', flex: 1, marginRight: '8px' }}>#{pr.id} {pr.title}</span>
                                        <span style={{
                                            fontSize: '8px',
                                            padding: '1px 5px',
                                            borderRadius: '8px',
                                            background: pr.status === 'OPEN' ? '#238636' : pr.status === 'MERGED' ? '#8957e5' : '#7d8590',
                                            color: 'white',
                                            fontWeight: 600,
                                            flexShrink: 0
                                        }}>
                                            {pr.status}
                                        </span>
                                    </div>
                                    <div style={{ fontSize: '9px', color: 'var(--text-tertiary)', marginBottom: '4px' }}>
                                        <code style={{ fontSize: '9px' }}>{pr.sourceBranch}</code> → <code style={{ fontSize: '9px' }}>{pr.targetBranch}</code>
                                    </div>
                                    {pr.status === 'OPEN' && (
                                        <button
                                            onClick={() => mergePullRequest(pr.id)}
                                            style={mergeButtonStyle}
                                        >
                                            Merge
                                        </button>
                                    )}
                                </div>
                            ))}
                        </div>
                    ) : (
                        <div style={emptyStyle}>No active PRs</div>
                    )}
                </div>

                {/* Remote Branches List */}
                <div>
                    <div style={{ ...sectionLabelStyle, marginBottom: '8px' }}>REMOTE BRANCHES</div>
                    {Object.keys(remoteBranches).length > 0 ? (
                        <div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
                            {Object.entries(remoteBranches).map(([name, hash]) => (
                                <div key={name} style={{
                                    display: 'flex',
                                    alignItems: 'center',
                                    justifyContent: 'space-between',
                                    padding: '6px 8px',
                                    background: 'var(--bg-secondary)',
                                    borderRadius: '6px',
                                    fontSize: '0.8rem'
                                }}>
                                    <span style={{ fontWeight: 500, color: 'var(--text-secondary)' }}>{name}</span>
                                    <span style={{ fontFamily: 'monospace', fontSize: '0.7rem', color: 'var(--text-tertiary)', background: 'rgba(255,255,255,0.05)', padding: '1px 4px', borderRadius: '3px' }}>
                                        {hash.substring(0, 6)}
                                    </span>
                                </div>
                            ))}
                        </div>
                    ) : (
                        <div style={emptyStyle}>No branches found</div>
                    )}
                </div>
            </div>
        </div>
    );
};

export default RemoteRepoView;
