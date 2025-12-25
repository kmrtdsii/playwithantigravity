import React, { createContext, useContext, useState, useEffect, useRef, useCallback } from 'react';
import type { GitState, PullRequest } from '../types/gitTypes';
import { gitService } from '../services/gitService';
import { filterReachableCommits } from '../utils/filterReachableCommits';
import { useTerminalTranscript, type TranscriptLine } from '../hooks/useTerminalTranscript';

interface GitContextType {
    state: GitState;
    sessionId: string;
    runCommand: (cmd: string, options?: { silent?: boolean }) => Promise<string[]>; // Return output for terminal to display
    // Terminal Recording API
    appendToTranscript: (text: string, hasNewline?: boolean) => void;
    terminalTranscripts: Record<string, TranscriptLine[]>;
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

    const [sessionOutputs, setSessionOutputs] = useState<Record<string, string[]>>({});
    const [sessionCmdCounts, setSessionCmdCounts] = useState<Record<string, number>>({});

    // Use extracted hook for transcript management
    const { terminalTranscripts, appendToTranscript, clearTranscript } = useTerminalTranscript(sessionId);

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

    // --- Core Functions (Memoized) ---

    // 1. fetchState: Independent logic, depends on showAllCommits
    const fetchState = useCallback(async (sid: string) => {
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
    }, [showAllCommits]);

    // 2. fetchServerState: Independent
    const fetchServerState = useCallback(async (name: string) => {
        try {
            const sState = await gitService.getRemoteState(name);
            setServerState(sState);
        } catch (e) {
            console.error("Failed to fetch server state", e);
            setServerState(null);
        }
    }, []);

    // 3. runCommand: Depends on fetchState, fetchServerState
    const runCommand = useCallback(async (cmd: string, options?: { silent?: boolean }): Promise<string[]> => {
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
    }, [sessionId, fetchState, fetchServerState, serverState]);

    // 4. Other Actions
    const toggleShowAllCommits = useCallback(() => {
        setShowAllCommits(prev => !prev);
    }, []);

    const stageFile = useCallback(async (file: string) => {
        await runCommand(`add ${file}`);
    }, [runCommand]);

    const unstageFile = useCallback(async (file: string) => {
        await runCommand(`restore --staged ${file}`);
    }, [runCommand]);

    const switchDeveloper = useCallback(async (name: string) => {
        const sid = developerSessions[name];
        if (sid) {
            setActiveDeveloper(name);
            setSessionId(sid);
            await fetchState(sid);
        }
    }, [developerSessions, fetchState]);

    const addDeveloper = useCallback(async (name: string) => {
        try {
            if (developers.includes(name)) return; // Prevent duplicates
            const data = await gitService.initSession();
            setDevelopers(prev => [...prev, name]);
            setDeveloperSessions(prev => ({ ...prev, [name]: data.sessionId }));
            if (!activeDeveloper) {
                setActiveDeveloper(name);
                setSessionId(data.sessionId);
            }
        } catch (e) {
            console.error("Failed to add developer", e);
        }
    }, [developers, activeDeveloper]);

    const refreshPullRequests = useCallback(async () => {
        try {
            const prs = await gitService.fetchPullRequests();
            setPullRequests(prs);
        } catch (e) {
            console.error("Failed to fetch PRs", e);
        }
    }, []);

    const ingestRemote = useCallback(async (name: string, url: string) => {
        await gitService.ingestRemote(name, url);
        await fetchState(sessionId);
    }, [sessionId, fetchState]);

    const createPullRequest = useCallback(async (title: string, desc: string, source: string, target: string) => {
        await gitService.createPullRequest({
            title,
            description: desc,
            sourceBranch: source,
            targetBranch: target,
            creator: activeDeveloper
        });
        await refreshPullRequests();
    }, [activeDeveloper, refreshPullRequests]);

    const mergePullRequest = useCallback(async (id: number) => {
        await gitService.mergePullRequest(id);
        await refreshPullRequests();
        // Refresh server state to show update in Left Pane
        await fetchServerState('origin');
        // Refresh local state too, just in case (though merge is remote-side)
        if (sessionId) await fetchState(sessionId);
    }, [sessionId, refreshPullRequests, fetchServerState, fetchState]);

    const resetRemote = useCallback(async (name: string = 'origin') => {
        await gitService.resetRemote(name);
        await fetchState(sessionId); // Refresh state after reset
    }, [sessionId, fetchState]);

    const refreshStateWrapper = useCallback(async () => {
        if (sessionId) await fetchState(sessionId);
    }, [sessionId, fetchState]);

    // Init session on mount - Create Alice and Bob
    useEffect(() => {
        const init = async () => {
            // Only init if not already done
            // NOTE: Check against state ref or simple check?
            // Since this runs once on mount, state is fresh.
            // But we can't access `developers` state in this closure correctly if we don't have it in deps?
            // Actually `addDeveloper` updates state.
            // The check `if (developers.length > 0)` might be stale if strict mode runs twice.
            // But `developers` is [] initially.
            // We use a local variable or ref to ensure init only runs once effectively?
            // Or just check if 'Alice' exists in server?
            // We will just call it. addDeveloper has duplication check `developers.includes` but that relies on state.
            // In React 18 strict mode, this might run twice.
            // Let's rely on `addDeveloper`'s duplicate check, but we need `developers` in deps for addDeveloper usually.
            // But here we want ONCE.

            // 1. Create Alice
            await addDeveloper('Alice');
            // 2. Create Bob
            await addDeveloper('Bob');
        };
        // We only want to run this ONCE.
        if (developers.length === 0) {
            init();
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    // Re-fetch when toggle changes
    useEffect(() => {
        if (sessionId) {
            fetchState(sessionId);
        }
    }, [sessionId, showAllCommits, fetchState]);

    const contextValue = React.useMemo(() => ({
        state,
        sessionId,
        runCommand,
        appendToTranscript,
        terminalTranscripts,
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
        refreshState: refreshStateWrapper,
        serverState,
        fetchServerState
    }), [
        state,
        sessionId,
        runCommand,
        appendToTranscript,
        terminalTranscripts,
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
        refreshStateWrapper,
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
