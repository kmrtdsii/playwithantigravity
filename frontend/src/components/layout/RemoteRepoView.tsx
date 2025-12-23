import React, { useMemo, useState, useEffect } from 'react';
import { useGit } from '../../context/GitAPIContext';
import GitGraphViz from '../visualization/GitGraphViz';
import type { GitState } from '../../types/gitTypes';
import { RemoteHeader, RemoteBranchList, PullRequestSection, containerStyle, actionButtonStyle } from './remote';

interface RemoteRepoViewProps {
    topHeight: number;
    onResizeStart: () => void;
}

/**
 * RemoteRepoView - Right panel showing the remote repository state.
 * 
 * Features:
 * - Repository URL configuration
 * - Remote Git graph visualization
 * - Pull request management
 * - Remote branch listing
 */
const RemoteRepoView: React.FC<RemoteRepoViewProps> = ({ topHeight, onResizeStart }) => {
    const {
        serverState,
        fetchServerState,
        pullRequests,
        mergePullRequest,
        refreshPullRequests,
        createPullRequest,
        ingestRemote
    } = useGit();

    // --- Local State ---
    const [setupUrl, setSetupUrl] = useState('');
    const [isSettingUp, setIsSettingUp] = useState(false);
    const [isEditMode, setIsEditMode] = useState(false);

    // Refresh PRs on mount
    useEffect(() => {
        refreshPullRequests();
    }, [refreshPullRequests]);

    // --- Computed Values ---
    const remoteGraphState: GitState = useMemo(() => {
        if (!serverState) {
            return createEmptyGitState();
        }
        return serverState;
    }, [serverState]);

    const remoteBranches = remoteGraphState.remoteBranches || {};
    const remoteUrl = setupUrl || (serverState?.remotes?.[0]?.urls?.[0]) || '';
    const projectName = remoteUrl.split('/').pop()?.replace('.git', '') || 'Remote Repository';
    const hasSharedRemotes = !!serverState;

    // --- Event Handlers ---
    const handleInitialSetup = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!setupUrl) return;
        setIsSettingUp(true);

        try {
            await ingestRemote('origin', setupUrl);
            await fetchServerState('origin');
            setIsEditMode(false);
        } catch (err) {
            console.error('Failed to update remote:', err);
            alert('Failed to update remote.');
        } finally {
            setIsSettingUp(false);
        }
    };

    const handleEditRemote = () => {
        const currentUrl = setupUrl || (serverState?.remotes?.[0]?.urls?.[0]) || '';
        setSetupUrl(currentUrl);
        setIsEditMode(true);
    };

    const handleCancelEdit = () => {
        setIsEditMode(false);
    };

    // --- Render ---
    return (
        <div style={containerStyle}>
            {/* TOP SPLIT: Info & Graph */}
            <div style={{ height: topHeight, display: 'flex', flexDirection: 'column', flexShrink: 0, minHeight: 0 }}>
                <RemoteHeader
                    remoteUrl={remoteUrl}
                    projectName={projectName}
                    isEditMode={isEditMode}
                    isSettingUp={isSettingUp}
                    setupUrl={setupUrl}
                    onSetupUrlChange={setSetupUrl}
                    onEditRemote={handleEditRemote}
                    onCancelEdit={handleCancelEdit}
                    onSubmit={handleInitialSetup}
                />

                {/* Graph Area */}
                <div style={{ flex: 1, minHeight: 0, position: 'relative', background: 'var(--bg-primary)' }}>
                    {hasSharedRemotes ? (
                        <GitGraphViz state={remoteGraphState} />
                    ) : (
                        <EmptyGraphPlaceholder
                            isEditMode={isEditMode}
                            onConnect={handleEditRemote}
                        />
                    )}
                </div>
            </div>

            {/* Resizer */}
            <div
                className="resizer"
                onMouseDown={onResizeStart}
                style={{
                    height: '4px',
                    cursor: 'row-resize',
                    background: 'var(--border-subtle)',
                    width: '100%',
                    zIndex: 10
                }}
            />

            {/* BOTTOM SPLIT: Remote Operations */}
            <div style={{ flex: 1, minHeight: 0, overflowY: 'auto', background: 'var(--bg-primary)' }}>
                <PullRequestSection
                    pullRequests={pullRequests}
                    branches={remoteGraphState.branches}
                    onCreatePR={createPullRequest}
                    onMergePR={mergePullRequest}
                />
                <RemoteBranchList remoteBranches={remoteBranches} />
            </div>
        </div>
    );
};

// --- Helper Components ---

interface EmptyGraphPlaceholderProps {
    isEditMode: boolean;
    onConnect: () => void;
}

const EmptyGraphPlaceholder: React.FC<EmptyGraphPlaceholderProps> = ({ isEditMode, onConnect }) => (
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
                    onClick={onConnect}
                    style={{
                        ...actionButtonStyle,
                        background: 'var(--bg-tertiary)',
                        color: 'var(--text-primary)',
                        border: '1px solid var(--border-subtle)'
                    }}
                >
                    Connect Repository
                </button>
            </>
        )}
    </div>
);

// --- Utility Functions ---

/**
 * Creates an empty GitState object for initial/fallback state.
 */
function createEmptyGitState(): GitState {
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

export default RemoteRepoView;
