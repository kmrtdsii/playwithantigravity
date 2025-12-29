import React, { useMemo, useState, useEffect } from 'react';
import { useGit } from '../../context/GitAPIContext';
import GitGraphViz from '../visualization/GitGraphViz';
import type { GitState } from '../../types/gitTypes';
import { RemoteHeader, PullRequestSection, CloneProgress, containerStyle } from './remote';
import EmptyState from './remote/EmptyState';
import { useRemoteClone } from '../../hooks/useRemoteClone';
import { useAutoDiscovery } from '../../hooks/useAutoDiscovery';

// Default remote URL for the GitGym application
// This repository is automatically available for cloning
// Default remote URL removed
const DEFAULT_REMOTE_URL = '';

interface RemoteRepoViewProps {
    topHeight: number;
    onResizeStart: () => void;
}

/**
 * RemoteRepoView - Right panel showing the remote repository state.
 * Refactored to use 'useRemoteClone' and 'useAutoDiscovery' hooks.
 */
const RemoteRepoView: React.FC<RemoteRepoViewProps> = ({ topHeight, onResizeStart }) => {
    const {
        serverState,
        pullRequests,
        mergePullRequest,
        refreshPullRequests,
        createPullRequest,
        deletePullRequest,
        resetRemote,
    } = useGit();

    // Custom Hooks
    const {
        cloneStatus,
        setCloneStatus,
        estimatedSeconds,
        elapsedSeconds,
        repoInfo,
        errorMessage,
        performClone,
        cancelClone
    } = useRemoteClone();

    // Local UI State - Initialize with default URL
    const [setupUrl, setSetupUrl] = useState(DEFAULT_REMOTE_URL);
    const [originalUrl, setOriginalUrl] = useState(DEFAULT_REMOTE_URL); // Store URL before editing
    const [isEditMode, setIsEditMode] = useState(false);

    // Auto Discovery
    useAutoDiscovery({ setupUrl, setSetupUrl, cloneStatus, performClone });

    // Initial Load - Auto-clone default remote on first render
    useEffect(() => {
        refreshPullRequests();
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    // Handlers
    const onCloneSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!setupUrl) return;
        await performClone(setupUrl, 0);
    };

    // Close edit mode on success
    useEffect(() => {
        if (cloneStatus === 'complete') {
            setIsEditMode(false);
        }
    }, [cloneStatus]);

    const handleRetry = () => {
        if (setupUrl) performClone(setupUrl, 0);
    };

    const handleEditRemote = () => {
        const currentUrl = setupUrl || (serverState?.remotes?.[0]?.urls?.[0]) || DEFAULT_REMOTE_URL;
        setOriginalUrl(currentUrl); // Save current URL before editing
        setSetupUrl(currentUrl);
        setIsEditMode(true);
    };

    const handleCancelEdit = () => {
        setSetupUrl(originalUrl); // Restore the URL to pre-edit value
        setIsEditMode(false);
        setCloneStatus('idle');
    };

    // Computed Values
    const remoteUrl = setupUrl || (serverState?.remotes?.[0]?.urls?.[0]) || '';
    const projectName = remoteUrl.split('/').pop()?.replace('.git', '') || 'Remote Repository';

    const handleDisconnect = async () => {
        try {
            await resetRemote(projectName);
        } catch (e) {
            console.warn('Reset remote failed, continuing with cleanup:', e);
        }
        // Clear all local state to return to initial view
        setSetupUrl('');
        cancelClone(); // This resets cloneStatus to 'idle' and clears error state
    };

    const remoteGraphState: GitState = useMemo(() => {
        if (!serverState) {
            return createEmptyGitState();
        }

        // Since serverState represents the remote repo itself, its local branches (refs/heads/*) 
        // are what we want to display. backend/handlers_remote.go explicitly clears remoteBranches.
        const mappedBranches = serverState.branches || {};

        // Determine HEAD
        let newHEAD = serverState.HEAD;

        // Fallback for HEAD if missing (common in bare repos if HEAD ref is missing or detached)
        if (!newHEAD || newHEAD.type === 'none') {
            if (mappedBranches['main']) {
                newHEAD = { type: 'branch', ref: 'main' };
            } else if (mappedBranches['master']) {
                newHEAD = { type: 'branch', ref: 'master' };
            }
        }

        // Construct the synthetic state representing the remote
        const syntheticState: GitState = {
            ...serverState,
            branches: mappedBranches,
            remoteBranches: {},
            HEAD: newHEAD,
            // Clear workstation specific state
            staging: [],
            modified: [],
            untracked: [],
        };

        // For remote repository, show ALL commits (no filtering)
        // This includes all branches, tags, and any orphan commits
        return {
            ...syntheticState,
            commits: serverState.commits  // All commits from server
        };
    }, [serverState]);

    const hasSharedRemotes = !!serverState;
    const isSettingUp = cloneStatus === 'fetching_info' || cloneStatus === 'cloning';

    return (
        <div style={containerStyle} data-testid="remote-repo-view">
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
                    onDisconnect={handleDisconnect}
                    onCancelEdit={handleCancelEdit}
                    onSubmit={onCloneSubmit}
                />

                {/* Clone Progress Display */}
                {cloneStatus !== 'idle' && (
                    <div style={{ padding: '0 16px' }}>
                        <CloneProgress
                            status={cloneStatus}
                            estimatedSeconds={estimatedSeconds}
                            elapsedSeconds={elapsedSeconds}
                            repoInfo={repoInfo}
                            errorMessage={errorMessage}
                            onRetry={handleRetry}
                            onCancel={cancelClone}
                        />
                    </div>
                )}

                {/* Graph Area */}
                <div style={{ flex: 1, minHeight: 0, position: 'relative', background: 'var(--bg-primary)' }}>
                    {hasSharedRemotes ? (
                        <GitGraphViz state={remoteGraphState} />
                    ) : (
                        <EmptyState
                            isEditMode={isEditMode}
                            cloneStatus={cloneStatus}
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
                {/* Pull Requests */}
                <PullRequestSection
                    pullRequests={pullRequests}
                    branches={remoteGraphState.branches}
                    onCreatePR={createPullRequest}
                    onMergePR={mergePullRequest}
                    onDeletePR={deletePullRequest}
                />

            </div>
        </div>
    );
};

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
