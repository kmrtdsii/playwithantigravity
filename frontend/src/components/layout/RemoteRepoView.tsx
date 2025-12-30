import React, { useMemo, useState, useEffect } from 'react';

import { useGit } from '../../context/GitAPIContext';
import GitGraphViz from '../visualization/GitGraphViz';
import type { GitState } from '../../types/gitTypes';
import { RemoteHeader, PullRequestSection, CloneProgress, containerStyle } from './remote';
import EmptyState from './remote/EmptyState';
import { useRemoteClone } from '../../hooks/useRemoteClone';
import { useAutoDiscovery } from '../../hooks/useAutoDiscovery';
import { gitService } from '../../services/gitService';

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
        activeRemoteView,
        setActiveRemoteView,
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

    // Local UI State
    const [setupUrl, setSetupUrl] = useState(DEFAULT_REMOTE_URL);
    // Control for showing the configuration/settings view (EmptyState used as config panel)
    const [isSettingsOpen, setIsSettingsOpen] = useState(false);

    // Auto Discovery
    useAutoDiscovery({ setupUrl, setSetupUrl, cloneStatus, performClone });

    // Multi-Remote List - now stores objects with name and url
    const [remoteList, setRemoteList] = useState<Array<{ name: string, url: string }>>([]);
    const updateRemoteList = async () => {
        try {
            const names = await gitService.listRemotes();
            const details = await Promise.all(names.map(async (name) => {
                try {
                    // Fetch state to get origin URL
                    const state = await gitService.getRemoteState(name);
                    const origin = state.remotes?.find(r => r.name === 'origin');
                    return { name, url: origin?.urls[0] || '' };
                } catch (e) {
                    return { name, url: '' };
                }
            }));
            setRemoteList(details);
        } catch (e) {
            console.error("Failed to list remotes", e);
        }
    };

    // Update list on mount and when serverState changes (e.g. added/removed remote)
    useEffect(() => {
        updateRemoteList();
    }, [serverState]);

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

    // Close settings/edit mode on success
    useEffect(() => {
        if (cloneStatus === 'complete') {
            setIsSettingsOpen(false);
        }
    }, [cloneStatus]);

    const handleRetry = () => {
        if (setupUrl) performClone(setupUrl, 0);
    };

    // Open settings view
    const handleOpenSettings = () => {
        setIsSettingsOpen(true);
    };

    const handleCancelEdit = () => {
        setIsSettingsOpen(false);
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
        cancelClone();
        setIsSettingsOpen(false);
    };

    const remoteGraphState: GitState = useMemo(() => {
        if (!serverState) {
            return createEmptyGitState();
        }

        const mappedBranches = serverState.branches || {};
        let newHEAD = serverState.HEAD;

        if (!newHEAD || newHEAD.type === 'none') {
            if (mappedBranches['main']) {
                newHEAD = { type: 'branch', ref: 'main' };
            } else if (mappedBranches['master']) {
                newHEAD = { type: 'branch', ref: 'master' };
            }
        }

        const syntheticState: GitState = {
            ...serverState,
            branches: mappedBranches,
            remoteBranches: {},
            HEAD: newHEAD,
            staging: [],
            modified: [],
            untracked: [],
        };

        return {
            ...syntheticState,
            commits: serverState.commits
        };
    }, [serverState]);

    const hasSharedRemotes = !!serverState;
    const isSettingUp = cloneStatus === 'fetching_info' || cloneStatus === 'cloning';
    const showSettings = !hasSharedRemotes || isSettingsOpen;

    // Filter PRs
    const filteredPRs = useMemo(() => {
        const target = activeRemoteView || 'origin';
        return pullRequests.filter(pr => (pr.remoteName || 'origin') === target);
    }, [pullRequests, activeRemoteView]);

    const handleCreatePR = async (title: string, desc: string, source: string, target: string) => {
        await createPullRequest(title, desc, source, target);
    };

    return (
        <div style={containerStyle} data-testid="remote-repo-view">
            {/* TOP SPLIT: Info & Graph */}
            <div style={{ height: topHeight, display: 'flex', flexDirection: 'column', flexShrink: 0, minHeight: 0 }}>
                {/* Header is always visible */}
                <RemoteHeader
                    remoteUrl={showSettings ? '' : remoteUrl}
                    projectName={showSettings ? '' : projectName}
                    isEditMode={false} // Deprecated
                    isSettingUp={isSettingUp}
                    setupUrl={setupUrl}
                    onSetupUrlChange={setSetupUrl}
                    onEditRemote={handleOpenSettings} // Trigger settings view
                    onDisconnect={handleDisconnect}
                    onCancelEdit={handleCancelEdit}
                    onSubmit={onCloneSubmit}
                    // Multi-remote - pass names only for compatibility if needed, or update Header?
                    // Header expects string[], so mapping names is correct.
                    remotes={remoteList.map(r => r.name)}
                    activeRemote={activeRemoteView}
                    onSelectRemote={setActiveRemoteView}
                    isSettingsOpen={isSettingsOpen}
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

                {/* Graph Area / Settings Area */}
                <div style={{ flex: 1, minHeight: 0, position: 'relative', background: 'var(--bg-primary)' }}>
                    {!showSettings ? (
                        <GitGraphViz state={remoteGraphState} />
                    ) : (
                        <EmptyState
                            isEditMode={true} // Always usable in settings view
                            cloneStatus={cloneStatus}
                            onConnect={handleOpenSettings}
                            // Pass full objects for the new UI
                            recentRemotes={remoteList}
                            onSelectRemote={(name) => {
                                setActiveRemoteView(name);
                                setIsSettingsOpen(false); // Close settings when selecting a remote from list
                            }}
                            onDeleteRemote={async (name) => {
                                try {
                                    await gitService.deleteRemote(name);
                                    await updateRemoteList();
                                    // If the deleted one was active or being viewed, reset?
                                    if (name === activeRemoteView) {
                                        setActiveRemoteView('origin'); // fallback
                                    }
                                    // Also if it matches current setupUrl/project, disconnect?
                                    if (name === projectName) {
                                        handleDisconnect();
                                    }
                                } catch (e) {
                                    console.error('Failed to delete remote:', e);
                                }
                            }}
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
                {/* Pull Requests - only show when remote exists AND has branches */}
                {hasSharedRemotes && Object.keys(remoteGraphState.branches).length > 0 && (
                    <PullRequestSection
                        pullRequests={filteredPRs}
                        branches={remoteGraphState.branches}
                        onCreatePR={handleCreatePR}
                        onMergePR={mergePullRequest}
                        onDeletePR={deletePullRequest}
                    />
                )}
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
