import React, { createContext, useContext, useState, useEffect, useRef } from 'react';
import type { GitState, PullRequest } from '../types/gitTypes';
import { gitService } from '../services/gitService';
import { filterReachableCommits } from '../utils/graphUtils';
import { useTerminalTranscript, type TranscriptLine } from '../hooks/useTerminalTranscript';

interface GitContextType {
    state: GitState;
    sessionId: string;
    runCommand: (cmd: string, options?: { silent?: boolean }) => Promise<string[]>; // Return output for terminal to display
    // Terminal Recording API
    appendToTranscript: (text: string, hasNewline?: boolean) => void;
    getTranscript: () => TranscriptLine[];
    clearTranscript: () => void;
    showAllCommits: boolean;
    toggleShowAllCommits: () => void;
    stageFile: (file: string) => Promise<void>;
    unstageFile: (file: string) => Promise<void>;

    developers: string[];
    activeDeveloper: string;
    switchDeveloper: (name: string) => Promise<void>;
    addDeveloper: (name: string) => Promise<void>;
    pullRequests: PullRequest[];
    refreshPullRequests: () => Promise<void>;
    ingestRemote: (name: string, url: string) => Promise<void>;
    createPullRequest: (title: string, desc: string, source: string, target: string) => Promise<void>;
    mergePullRequest: (id: number) => Promise<void>;
    resetRemote: (name?: string) => Promise<void>;
    refreshState: () => Promise<void>;
    serverState: GitState | null;
    fetchServerState: (name: string) => Promise<void>;
}

const GitContext = createContext<GitContextType | undefined>(undefined);

export const GitProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
    console.log("GitProvider: Mounted");
    const [state, setState] = useState<GitState>({
        initialized: false,
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
        commandCount: 0
    });

    const [serverState, setServerState] = useState<GitState | null>(null);

    const [sessionId, setSessionId] = useState<string>('');
    const [showAllCommits, setShowAllCommits] = useState<boolean>(false);

    const [developers, setDevelopers] = useState<string[]>([]);
    const [developerSessions, setDeveloperSessions] = useState<Record<string, string>>({});
    const [activeDeveloper, setActiveDeveloper] = useState<string>('');
    const [pullRequests, setPullRequests] = useState<PullRequest[]>([]);

    const addDeveloper = async (name: string) => {
        try {
            if (developers.includes(name)) return; // Prevent duplicates
            const data = await gitService.initSession();
            setDevelopers(prev => [...prev, name]);
            setDeveloperSessions(prev => ({ ...prev, [name]: data.sessionId }));
            if (!activeDeveloper) {
                setActiveDeveloper(name);
                setSessionId(data.sessionId);
                // We define fetchState below, so we can't call it here directly if we strictly follow order?
                // Actually function hoisting works for `const` functions ONLY IF defined before usage.
                // Circular dependency: addDeveloper -> fetchState -> setState.
                // fetchState is defined BELOW.
                // We need to move fetchState UP as well or define these using function keyword (hoisted).
                // Or use `useEffect` to trigger fetch when session changes.
                // But let's just use the `sessionId` setter and let the existing `useEffect` (line 300) handle fetch?
                // Line 300: `useEffect(() => { if (sessionId) fetchState(sessionId) }, [showAllCommits])`.
                // It depends on `showAllCommits`. It does NOT trigger on `sessionId` change currently.
                // We should update line 300 to depend on `sessionId` too?
                // But let's assume moving `addDeveloper` is tricky if it calls `fetchState`.
            }
        } catch (e) {
            console.error("Failed to add developer", e);
        }
    };

    // Init session on mount - Create Alice and Bob
    useEffect(() => {
        const init = async () => {
            // Only init if not already done
            if (developers.length > 0) return;

            // 1. Create Alice
            await addDeveloper('Alice');

            // 2. Create Bob
            await addDeveloper('Bob');
        };
        init();
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    const [sessionOutputs, setSessionOutputs] = useState<Record<string, string[]>>({});
    const [sessionCmdCounts, setSessionCmdCounts] = useState<Record<string, number>>({});

    // Use extracted hook for transcript management
    const { appendToTranscript, getTranscript, clearTranscript } = useTerminalTranscript(sessionId);

    // FIX: Use refs to avoid stale closure issues in async callbacks
    // These refs always hold the latest value
    const sessionOutputsRef = useRef<Record<string, string[]>>({});
    const sessionCmdCountsRef = useRef<Record<string, number>>({});

    // Keep refs in sync with state
    useEffect(() => {
        sessionOutputsRef.current = sessionOutputs;
    }, [sessionOutputs]);

    useEffect(() => {
        sessionCmdCountsRef.current = sessionCmdCounts;
    }, [sessionCmdCounts]);

    const fetchState = async (sid: string) => {
        try {
            const newState = await gitService.fetchState(sid, showAllCommits);
            // When fetching state, we must ensure we are updating the "active" view state
            // But if we switched users, we want to load THAT user's output/count.
            // Since this function is async, by the time it returns, sessionId state might match sid.

            setState(prev => {
                // FIX: Use refs instead of state to get latest values (avoids stale closure)
                const storedOutput = sessionOutputsRef.current[sid] || [];
                const storedCount = sessionCmdCountsRef.current[sid] || 0;

                // Client-side filtering:
                // If showAllCommits is FALSE, filter out unreachable commits
                const finalCommits = showAllCommits
                    ? newState.commits
                    : filterReachableCommits(newState.commits, newState);

                return {
                    ...prev,
                    ...newState,
                    commits: finalCommits,
                    output: storedOutput,
                    commandCount: storedCount,
                    _sessionId: sid // Inject session ID for validation
                };
            });
        } catch (e) {
            console.error(e);
        }
    };

    const fetchServerState = async (name: string) => {
        try {
            const sState = await gitService.getRemoteState(name);
            setServerState(sState);
        } catch (e) {
            console.error("Failed to fetch server state", e);
            setServerState(null);
        }
    };

    const runCommand = async (cmd: string, options?: { silent?: boolean }): Promise<string[]> => {
        if (!sessionId) {
            console.error("No session ID");
            return [];
        }

        console.log(`Executing command: ${cmd} (Session: ${sessionId})`);

        // 1. Echo Command (Skip if silent)
        const commandEcho = `> ${cmd}`;
        if (!options?.silent) {
            setSessionOutputs(prev => {
                const current = prev[sessionId] || [];
                return { ...prev, [sessionId]: [...current, commandEcho] };
            });
            setState(prev => ({
                ...prev,
                output: [...prev.output, commandEcho]
            }));
        }

        try {
            const data = await gitService.executeCommand(sessionId, cmd);
            console.log("GitAPI: Command response:", data);

            let newLines: string[] = [];
            let isError = false;

            if (data.error) {
                newLines = [`Error: ${data.error}`];
                isError = true;
            } else if (data.output) {
                newLines = [data.output];
            }

            // 2. Append Output (Skip if silent AND no error)
            if (!options?.silent || isError) {
                // Update Persistent Store
                setSessionOutputs(prev => {
                    const current = prev[sessionId] || [];
                    return { ...prev, [sessionId]: [...current, ...newLines] };
                });

                // Update Transient State
                setState(prev => ({
                    ...prev,
                    output: [...prev.output, ...newLines]
                }));
            }

            // Always fetch fresh state after command (using current sessionId)
            // This ensures currentPath and HEAD are updated before prompt is shown
            await fetchState(sessionId);

            // NOW increment command count - this triggers prompt rendering with correct state
            // Even silent commands effectively trigger a prompt refresh via 'pathChanged', 
            // but incrementing this ensures reliability.
            setSessionCmdCounts(prev => {
                const current = prev[sessionId] || 0;
                return { ...prev, [sessionId]: current + 1 };
            });
            setState(prev => ({
                ...prev,
                commandCount: prev.commandCount + 1
            }));

            // AUTO-REFRESH SERVER STATE
            if (serverState && serverState.remotes?.length === 0) {
                await fetchServerState('origin');
            } else if (serverState) {
                await fetchServerState('origin'); // Default fallback
            }

            return newLines;

        } catch (e) {
            console.error(e);

            const errorLine = "Network error";
            // Handle error store update (Always show network errors)
            setSessionOutputs(prev => ({ ...prev, [sessionId]: [...(prev[sessionId] || []), errorLine] }));
            setSessionCmdCounts(prev => ({ ...prev, [sessionId]: (prev[sessionId] || 0) + 1 }));

            setState(prev => ({ ...prev, output: [...prev.output, errorLine], commandCount: prev.commandCount + 1 }));
            return [errorLine];
        }
    };

    const toggleShowAllCommits = () => {
        setShowAllCommits(prev => !prev);
    };

    const stageFile = async (file: string) => {
        await runCommand(`add ${file}`);
    };

    const unstageFile = async (file: string) => {
        await runCommand(`restore --staged ${file}`);
    };

    // addDeveloper moved up

    const switchDeveloper = async (name: string) => {
        const sid = developerSessions[name];
        if (sid) {
            setActiveDeveloper(name);
            setSessionId(sid);
            await fetchState(sid);
        }
    };

    const refreshPullRequests = async () => {
        try {
            const prs = await gitService.fetchPullRequests();
            setPullRequests(prs);
        } catch (e) {
            console.error("Failed to fetch PRs", e);
        }
    };

    const ingestRemote = async (name: string, url: string) => {
        await gitService.ingestRemote(name, url);
        await fetchState(sessionId);
    };

    const createPullRequest = async (title: string, desc: string, source: string, target: string) => {
        await gitService.createPullRequest({
            title,
            description: desc,
            sourceBranch: source,
            targetBranch: target,
            creator: activeDeveloper
        });
        await refreshPullRequests();
    };

    const mergePullRequest = async (id: number) => {
        await gitService.mergePullRequest(id);
        await refreshPullRequests();
        // Refresh server state to show update in Left Pane
        await fetchServerState('origin');
        // Refresh local state too, just in case (though merge is remote-side)
        if (sessionId) await fetchState(sessionId);
    };

    const resetRemote = async (name: string = 'origin') => {
        await gitService.resetRemote(name);
        await fetchState(sessionId); // Refresh state after reset
    };

    // Re-fetch when toggle changes
    useEffect(() => {
        if (sessionId) {
            fetchState(sessionId);
        }
    }, [sessionId, showAllCommits]);

    const contextValue = React.useMemo(() => ({
        state,
        sessionId,
        runCommand,
        appendToTranscript,
        getTranscript,
        clearTranscript,
        showAllCommits,
        toggleShowAllCommits,
        stageFile,
        unstageFile,
        developers,
        activeDeveloper,
        switchDeveloper,
        addDeveloper,
        pullRequests,
        refreshPullRequests,
        ingestRemote,
        createPullRequest,
        mergePullRequest,
        resetRemote,
        refreshState: async () => { if (sessionId) await fetchState(sessionId); },
        serverState,
        fetchServerState
    }), [
        state,
        sessionId,
        runCommand, // Unstable but necessary
        appendToTranscript,
        getTranscript,
        clearTranscript,
        showAllCommits,
        developers,
        activeDeveloper,
        switchDeveloper,
        addDeveloper,
        pullRequests,
        refreshPullRequests,
        ingestRemote,
        createPullRequest,
        mergePullRequest,
        resetRemote,
        serverState,
        fetchServerState
    ]);

    return (
        <GitContext.Provider value={contextValue}>
            {children}
        </GitContext.Provider>
    );
};

export const useGit = () => {
    const context = useContext(GitContext);
    if (!context) {
        throw new Error('useGit must be used within a GitProvider');
    }
    return context;
};
export type { GitState, Commit } from '../types/gitTypes';
