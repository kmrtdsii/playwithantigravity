import React, { useMemo, useState, useEffect, useRef, useCallback } from 'react';
import { useGit } from '../../context/GitAPIContext';
import GitGraphViz from '../visualization/GitGraphViz';
import type { GitState } from '../../types/gitTypes';
import { RemoteHeader, RemoteBranchList, PullRequestSection, CloneProgress, containerStyle, actionButtonStyle } from './remote';
import type { CloneStatus } from './remote';
import { gitService } from '../../services/gitService';

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
        state, // Access local state for auto-discovery
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
    const [isEditMode, setIsEditMode] = useState(false);

    // Clone Progress State
    const [cloneStatus, setCloneStatus] = useState<CloneStatus>('idle');
    const [estimatedSeconds, setEstimatedSeconds] = useState<number>(0);
    const [elapsedSeconds, setElapsedSeconds] = useState(0);
    const [repoInfo, setRepoInfo] = useState<{
        name: string;
        sizeDisplay: string;
        message: string;
    } | undefined>(undefined);
    const [errorMessage, setErrorMessage] = useState<string | undefined>(undefined);

    // Timer ref for elapsed time tracking
    const timerRef = useRef<number | null>(null);

    // Cleanup timer on unmount
    useEffect(() => {
        return () => {
            if (timerRef.current) {
                clearInterval(timerRef.current);
            }
        };
    }, []);

    // Refresh PRs on mount only
    useEffect(() => {
        refreshPullRequests();
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    // Auto-Discovery: Detect 'origin' remote from local state (e.g. after git clone)
    useEffect(() => {
        const localOrigin = state.remotes?.find(r => r.name === 'origin');
        if (localOrigin && localOrigin.urls.length > 0) {
            const detectedUrl = localOrigin.urls[0];

            // Should properly handle the case where serverState is already set but might be different?
            // For now, only auto-configure if we are in a "disconnected" state (no serverState)
            // or if we have a setupUrl but haven't committed it (e.g. user manually typing vs auto)
            // We prioritize the auto-detected one if current UI is empty.
            if (!serverState && !setupUrl && cloneStatus === 'idle') {
                console.log('Auto-detected remote origin:', detectedUrl);
                setSetupUrl(detectedUrl);

                // Auto-connect functionality
                performClone(detectedUrl);
            }
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [state.remotes, serverState, setupUrl, cloneStatus]);

    // --- Clone Process ---
    const performClone = useCallback(async (url: string) => {
        // Reset state
        setErrorMessage(undefined);
        setElapsedSeconds(0);
        setEstimatedSeconds(0);
        setRepoInfo(undefined);

        try {
            // Step 1: Fetch repo info
            setCloneStatus('fetching_info');

            const info = await gitService.getRemoteInfo(url);
            setRepoInfo({
                name: info.repoInfo.name,
                sizeDisplay: info.sizeDisplay,
                message: info.message,
            });
            setEstimatedSeconds(info.estimatedSeconds);

            // Step 2: Start cloning
            setCloneStatus('cloning');

            // Start elapsed timer
            const startTime = Date.now();
            timerRef.current = window.setInterval(() => {
                const elapsed = (Date.now() - startTime) / 1000;
                setElapsedSeconds(elapsed);
            }, 500);

            // Perform the actual clone
            await ingestRemote('origin', url);
            await fetchServerState('origin');

            // Step 3: Complete
            if (timerRef.current) {
                clearInterval(timerRef.current);
                timerRef.current = null;
            }
            setCloneStatus('complete');
            setIsEditMode(false);

            // Reset to idle after a short delay
            setTimeout(() => {
                setCloneStatus('idle');
            }, 2000);

        } catch (err) {
            // Handle error
            if (timerRef.current) {
                clearInterval(timerRef.current);
                timerRef.current = null;
            }
            setCloneStatus('error');
            setErrorMessage(err instanceof Error ? err.message : 'Unknown error occurred');
            console.error('Clone failed:', err);
        }
    }, [ingestRemote, fetchServerState]);

    // --- Event Handlers ---
    const handleInitialSetup = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!setupUrl) return;
        performClone(setupUrl);
    };

    const handleRetry = () => {
        if (setupUrl) {
            performClone(setupUrl);
        }
    };

    const handleCancelClone = () => {
        if (timerRef.current) {
            clearInterval(timerRef.current);
            timerRef.current = null;
        }
        setCloneStatus('idle');
        setErrorMessage(undefined);
    };

    const handleEditRemote = () => {
        const currentUrl = setupUrl || (serverState?.remotes?.[0]?.urls?.[0]) || '';
        setSetupUrl(currentUrl);
        setIsEditMode(true);
    };

    const handleCancelEdit = () => {
        setIsEditMode(false);
        setCloneStatus('idle');
        setErrorMessage(undefined);
    };

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

    // Map cloneStatus to isSettingUp for RemoteHeader compatibility
    const isSettingUp = cloneStatus === 'fetching_info' || cloneStatus === 'cloning';

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
                            onCancel={handleCancelClone}
                        />
                    </div>
                )}

                {/* Graph Area */}
                <div style={{ flex: 1, minHeight: 0, position: 'relative', background: 'var(--bg-primary)' }}>
                    {hasSharedRemotes ? (
                        <GitGraphViz state={remoteGraphState} />
                    ) : (
                        <EmptyGraphPlaceholder
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
    cloneStatus?: CloneStatus;
    onConnect: () => void;
}

const EmptyGraphPlaceholder: React.FC<EmptyGraphPlaceholderProps> = ({ isEditMode, cloneStatus, onConnect }) => (
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
        {!isEditMode && cloneStatus === 'idle' && (
            <>
                <div style={{ fontSize: '24px', opacity: 0.3 }}>🌐</div>
                <div style={{ fontSize: '0.85rem' }}>No Remote Configured</div>
                <button
                    onClick={onConnect}
                    style={{
                        ...actionButtonStyle,
                        background: 'var(--bg-tertiary)',
                        color: 'var(--text-primary)',
                        border: '1px solid var(--border-subtle)',
                        fontSize: '14px', // Increased size
                        padding: '10px 20px', // Increased padding
                        marginTop: '10px'
                    }}
                >
                    Connect Repository
                </button>
            </>
        )}
        {(cloneStatus === 'fetching_info' || cloneStatus === 'cloning') && (
            <div style={{ fontSize: '0.85rem', color: 'var(--text-secondary)' }}>
                Connecting to repository...
            </div>
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
