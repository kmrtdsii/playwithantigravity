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

const sectionLabelStyle: React.CSSProperties = {
    fontSize: '0.75rem',
    fontWeight: 800,
    color: 'var(--text-secondary)',
    letterSpacing: '0.05em'
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
    // Removed unused 'addDeveloper' and 'state'
    const { serverState, fetchServerState, pullRequests, mergePullRequest, refreshPullRequests, createPullRequest, ingestRemote } = useGit();

    // Use serverState for the remote graph
    // If not set, fallback to empty or loading
    const remoteGraphState: GitState = useMemo(() => {
        if (!serverState) {
            return {
                commits: [],
                branches: {},
                tags: {},
                references: {},
                remotes: [],
                remoteBranches: {},
                HEAD: { type: 'none', ref: null },
                staging: [],
                modified: [],
                untracked: [],
                fileStatuses: {},
                files: [],
                potentialCommits: [],
                sharedRemotes: [],
                output: [],
                commandCount: 0,
                initialized: false
            };
        }
        return serverState;
    }, [serverState]);

    const remoteBranches = remoteGraphState.remoteBranches || {};


    const [setupUrl, setSetupUrl] = React.useState('');
    const [isSettingUp, setIsSettingUp] = React.useState(false);
    // Removed setupLog/setupSuccess as they were unused in the new minimalist UI.
    const [isEditMode, setIsEditMode] = React.useState(false);

    React.useEffect(() => {
        refreshPullRequests();
        // If we have a URL already configured, fetch it?
        // We need a stable way to know "Which remote are we looking at?"
        // Ideally, we persist "activeRemoteName" in context or local storage.
        // For now, if setupUrl is present, fetch it?
        // Or just rely on user interaction.
    }, []);

    const handleInitialSetup = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!setupUrl) return;
        setIsSettingUp(true);

        try {
            // 1. Ingest into Backend (Simulated Remote Server)
            // This creates the "Remote Repo" on the server side
            await ingestRemote('origin', setupUrl);

            // 2. Fetch the state of this new Remote Repo to visualize it
            await fetchServerState('origin'); // Assumes ingestRemote stores it as 'origin' or mapped by name
            // Actually ingestRemote stores by name AND url.

            // 3. DO NOT Touch Local Session
            // The user must manually run `git clone` in the terminal to hydrate the Center Pane.

            setIsEditMode(false);
        } catch (e) {
            console.error(e);
            alert('Failed to update remote.');
        } finally {
            setIsSettingUp(false);
        }
    };

    const handleEditRemote = () => {
        // Pre-fill with the currently displayed URL (from setupUrl or serverState)
        const currentUrl = setupUrl || (serverState?.remotes?.[0]?.urls?.[0]) || '';
        setSetupUrl(currentUrl);
        setIsEditMode(true);
    };

    const handleCancelEdit = () => {
        // Simply exit edit mode without modifying setupUrl
        // This preserves whatever was there before (the original remote URL)
        setIsEditMode(false);
    };

    const [isCompareMode, setIsCompareMode] = React.useState(false);
    const [compareBase, setCompareBase] = React.useState('main');
    const [compareCompare, setCompareCompare] = React.useState('');

    // Set default compare branch when branches load
    React.useEffect(() => {
        const branches = Object.keys(remoteGraphState.branches);
        if (branches.length > 0) {
            if (!branches.includes(compareBase)) setCompareBase(branches[0]);
            if (!compareCompare && branches.length > 1) {
                setCompareCompare(branches.find(b => b !== 'main') || branches[1]);
            } else if (!compareCompare) {
                setCompareCompare(branches[0]);
            }
        }
    }, [remoteGraphState.branches, compareBase, compareCompare]);


    const handleCreatePRSubmit = () => {
        const title = prompt('PR Title', `Merge ${compareCompare} into ${compareBase}`);
        if (title) {
            createPullRequest(title, '', compareCompare, compareBase);
            setIsCompareMode(false);
        }
    };

    const remoteUrl = setupUrl || (serverState?.remotes?.[0]?.urls?.[0]) || ''; // Prefer setupUrl or serverState
    const projectName = remoteUrl.split('/').pop()?.replace('.git', '') || 'Remote Repository';
    const displayTitle = remoteUrl ? projectName : 'NO REMOTE CONFIGURED';

    const hasSharedRemotes = !!serverState;

    return (
        <div style={containerStyle}>
            {/* TOP SPLIT: Info & Graph */}
            <div style={{ height: topHeight, display: 'flex', flexDirection: 'column', flexShrink: 0, minHeight: 0 }}>
                {/* Repo Info Header - REFACTORED */}
                <div style={{
                    padding: '8px 12px',
                    background: 'var(--bg-secondary)',
                    borderBottom: '1px solid var(--border-subtle)',
                    display: 'flex',
                    flexDirection: 'column',
                    gap: '4px',
                    flexShrink: 0,
                    minHeight: '72px', // MATCH CENTRE HEADER (32px Tabs + 40px Controls)
                    justifyContent: 'center'
                }}>
                    {isEditMode ? (
                        /* INLINE EDIT FORM */
                        <form onSubmit={handleInitialSetup} style={{ display: 'flex', gap: '8px', alignItems: 'center', width: '100%' }}>
                            <input
                                type="text"
                                placeholder="https://github.com/..."
                                value={setupUrl}
                                onChange={(e) => setSetupUrl(e.target.value)}
                                style={{
                                    flex: 1,
                                    padding: '4px 8px',
                                    borderRadius: '4px',
                                    border: '1px solid var(--accent-primary)',
                                    background: 'var(--bg-primary)',
                                    color: 'var(--text-primary)',
                                    fontSize: '11px',
                                    outline: 'none'
                                }}
                                autoFocus
                            />
                            <button
                                type="button"
                                onClick={handleCancelEdit}
                                style={{
                                    padding: '4px 8px',
                                    fontSize: '10px',
                                    background: 'transparent',
                                    color: 'var(--text-secondary)',
                                    border: '1px solid var(--border-subtle)',
                                    borderRadius: '4px',
                                    cursor: 'pointer'
                                }}
                            >
                                Cancel
                            </button>
                            <button
                                type="submit"
                                disabled={isSettingUp || !setupUrl}
                                style={{
                                    padding: '4px 12px',
                                    fontSize: '10px',
                                    fontWeight: 700,
                                    background: 'var(--accent-primary)',
                                    color: 'white',
                                    border: 'none',
                                    borderRadius: '4px',
                                    cursor: 'pointer',
                                    opacity: isSettingUp ? 0.7 : 1
                                }}
                            >
                                {isSettingUp ? 'UPDATING...' : 'UPDATE'}
                            </button>
                        </form>
                    ) : (
                        /* VIEW MODE */
                        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                            <div style={{ display: 'flex', flexDirection: 'column', gap: '2px', overflow: 'hidden' }}>
                                <div style={{ fontWeight: 700, fontSize: '0.9rem', display: 'flex', alignItems: 'center', gap: '6px', color: 'var(--text-primary)' }}>
                                    <span>{displayTitle}</span>
                                    {remoteUrl && (
                                        <span style={{
                                            fontSize: '0.65rem',
                                            background: '#238636',
                                            color: 'white',
                                            padding: '1px 6px',
                                            borderRadius: '10px',
                                            fontWeight: 600
                                        }}>
                                            origin
                                        </span>
                                    )}
                                </div>
                                <div style={{ fontSize: '0.7rem', color: 'var(--text-secondary)', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                                    {remoteUrl || 'Connect a GitHub repository to visualize remote history.'}
                                </div>
                            </div>

                            <button
                                onClick={handleEditRemote}
                                style={{
                                    padding: '2px 8px',
                                    fontSize: '10px',
                                    background: 'transparent',
                                    border: '1px solid var(--border-subtle)',
                                    borderRadius: '4px',
                                    color: 'var(--text-secondary)',
                                    cursor: 'pointer',
                                    fontWeight: 600,
                                    whiteSpace: 'nowrap'
                                }}
                            >
                                {remoteUrl ? 'Configure' : 'Connect Repository'}
                            </button>
                        </div>
                    )}
                </div>

                {/* Graph Area */}
                <div style={{ flex: 1, minHeight: 0, position: 'relative', background: 'var(--bg-primary)' }}>
                    {hasSharedRemotes || (serverState) ? (
                        <GitGraphViz
                            state={remoteGraphState}
                        // Remote graph selection logic if needed
                        />
                    ) : (
                        <div style={{
                            height: '100%',
                            display: 'flex',
                            flexDirection: 'column',
                            alignItems: 'center',
                            justifyContent: 'center',
                            color: 'var(--text-tertiary)',
                            gap: '12px',
                            padding: '20px',
                            textAlign: 'center'
                        }}>
                            {!isEditMode && (
                                <>
                                    <div style={{ fontSize: '24px', opacity: 0.3 }}>🌐</div>
                                    <div style={{ fontSize: '0.85rem' }}>No Remote Configured</div>
                                    <button
                                        onClick={handleEditRemote}
                                        style={{ ...actionButtonStyle, background: 'var(--bg-tertiary)', color: 'var(--text-primary)', border: '1px solid var(--border-subtle)' }}
                                    >
                                        Connect Repository
                                    </button>
                                </>
                            )}
                        </div>
                    )}
                </div>
            </div>

            {/* Content Resizer */}
            <div
                className="resizer"
                onMouseDown={onResizeStart}
                style={{ height: '4px', cursor: 'row-resize', background: 'var(--border-subtle)', width: '100%', zIndex: 10 }}
            />

            {/* BOTTOM SPLIT: Remote Operations */}
            <div style={{ flex: 1, minHeight: 0, overflowY: 'auto', background: 'var(--bg-primary)' }}>
                {/* Pull Requests Section */}
                <div style={{ padding: '16px' }}>
                    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '12px' }}>
                        <div style={sectionLabelStyle}>PULL REQUESTS</div>
                        {!isCompareMode && (
                            <button onClick={() => setIsCompareMode(true)} style={{ ...actionButtonStyle, background: '#238636' }}>
                                New Pull Request
                            </button>
                        )}
                    </div>

                    {isCompareMode ? (
                        <div style={{ background: 'var(--bg-secondary)', borderRadius: '6px', border: '1px solid var(--border-subtle)', marginBottom: '16px', overflow: 'hidden' }}>
                            <div style={{ padding: '12px', borderBottom: '1px solid var(--border-subtle)', background: 'var(--bg-primary)' }}>
                                <div style={{ fontSize: '1.2rem', fontWeight: 600, marginBottom: '4px' }}>Comparing changes</div>
                                <div style={{ fontSize: '0.8rem', color: 'var(--text-secondary)' }}>
                                    Choose two branches to see what’s changed or to start a new pull request.
                                </div>
                            </div>

                            <div style={{ padding: '12px', display: 'flex', alignItems: 'center', gap: '8px', background: 'var(--bg-secondary)', borderBottom: '1px solid var(--border-subtle)' }}>
                                <div style={{ display: 'flex', alignItems: 'center', gap: '4px', fontSize: '0.85rem' }}>
                                    <span style={{ color: 'var(--text-tertiary)' }}>base:</span>
                                    <select
                                        value={compareBase}
                                        onChange={e => setCompareBase(e.target.value)}
                                        style={{ background: 'var(--bg-primary)', color: 'var(--text-primary)', border: '1px solid var(--border-subtle)', borderRadius: '6px', padding: '4px 8px' }}
                                    >
                                        {Object.keys(remoteGraphState.branches).map(b => <option key={b} value={b}>{b}</option>)}
                                    </select>
                                </div>
                                <span style={{ color: 'var(--text-tertiary)' }}>←</span>
                                <div style={{ display: 'flex', alignItems: 'center', gap: '4px', fontSize: '0.85rem' }}>
                                    <span style={{ color: 'var(--text-tertiary)' }}>compare:</span>
                                    <select
                                        value={compareCompare}
                                        onChange={e => setCompareCompare(e.target.value)}
                                        style={{ background: 'var(--bg-primary)', color: 'var(--text-primary)', border: '1px solid var(--border-subtle)', borderRadius: '6px', padding: '4px 8px' }}
                                    >
                                        {Object.keys(remoteGraphState.branches).map(b => <option key={b} value={b}>{b}</option>)}
                                    </select>
                                </div>
                            </div>

                            <div style={{ padding: '12px', background: '#e6ffec', color: '#1a7f37', fontSize: '0.85rem', display: 'flex', alignItems: 'center', gap: '6px', borderBottom: '1px solid var(--border-subtle)' }}>
                                <span>✓</span>
                                <strong>Able to merge.</strong>
                                <span>These branches can be automatically merged.</span>
                            </div>

                            <div style={{ padding: '12px', display: 'flex', justifyContent: 'flex-end', gap: '8px' }}>
                                <button
                                    onClick={() => setIsCompareMode(false)}
                                    style={{ padding: '6px 12px', background: 'transparent', border: 'none', color: 'var(--text-secondary)', cursor: 'pointer' }}
                                >
                                    Cancel
                                </button>
                                <button
                                    onClick={handleCreatePRSubmit}
                                    style={{ ...actionButtonStyle, background: '#238636', fontSize: '0.9rem', padding: '6px 16px' }}
                                >
                                    Create pull request
                                </button>
                            </div>
                        </div>
                    ) : (
                        <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                            {pullRequests.length === 0 ? (
                                <div style={emptyStyle}>No active PRs</div>
                            ) : (
                                pullRequests.map(pr => (
                                    <div key={pr.id} style={prCardStyle}>
                                        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                                            <div style={{ fontWeight: 700, fontSize: '0.85rem' }}>#{pr.id} {pr.title}</div>
                                            <span style={{ fontSize: '0.7rem', padding: '2px 6px', background: pr.status === 'OPEN' ? '#238636' : '#8957e5', color: 'white', borderRadius: '10px' }}>
                                                {pr.status}
                                            </span>
                                        </div>
                                        <div style={{ fontSize: '0.75rem', color: 'var(--text-secondary)' }}>
                                            {pr.sourceBranch} ➜ {pr.targetBranch}
                                        </div>
                                        <div style={{ fontSize: '0.7rem', color: 'var(--text-tertiary)' }}>
                                            opened by {pr.creator}
                                        </div>
                                        {pr.status === 'OPEN' && (
                                            <button
                                                onClick={() => mergePullRequest(pr.id)}
                                                style={mergeButtonStyle}
                                            >
                                                Merge Pull Request
                                            </button>
                                        )}
                                    </div>
                                ))
                            )}
                        </div>
                    )}
                </div>

                {/* Remote Branches Section */}
                <div style={{ padding: '0 16px 16px 16px' }}>
                    <div style={{ ...sectionLabelStyle, marginBottom: '12px' }}>REMOTE BRANCHES</div>
                    <div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
                        {Object.keys(remoteBranches).length === 0 ? (
                            <div style={emptyStyle}>No branches found</div>
                        ) : (
                            Object.entries(remoteBranches).map(([name, hash]) => (
                                <div key={name} style={{
                                    display: 'flex',
                                    justifyContent: 'space-between',
                                    padding: '6px 10px',
                                    background: 'var(--bg-secondary)',
                                    borderRadius: '6px',
                                    fontSize: '0.8rem',
                                    border: '1px solid var(--border-subtle)'
                                }}>
                                    <span style={{ fontFamily: 'monospace' }}>{name}</span>
                                    <span style={{ color: 'var(--text-tertiary)', fontSize: '0.7rem', fontFamily: 'monospace' }}>
                                        {(hash as string).substring(0, 7)}
                                    </span>
                                </div>
                            ))
                        )}
                    </div>
                </div>
            </div>
        </div>
    );
};

export default RemoteRepoView;
