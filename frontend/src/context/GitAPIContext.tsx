import React, { createContext, useContext, useState, useEffect, useRef } from 'react';
import type { GitState, PullRequest, BranchingStrategy } from '../types/gitTypes';
import { gitService } from '../services/gitService';
import { filterReachableCommits } from '../utils/graphUtils';

// [Architectural Decision] Terminal Recording System
// To ensure exact reproduction of the terminal state (including prompts, colors, empty lines),
// we treat the terminal output as a "Transcript" that matches exactly what xterm.js displayed.
export interface TranscriptLine {
    text: string;
    hasNewline: boolean;
}

interface GitContextType {
    state: GitState;
    sessionId: string;
    runCommand: (cmd: string) => Promise<string[]>; // Return output for terminal to display
    // Terminal Recording API
    appendToTranscript: (text: string, hasNewline?: boolean) => void;
    getTranscript: () => TranscriptLine[];
    clearTranscript: () => void;
    showAllCommits: boolean;
    toggleShowAllCommits: () => void;
    stageFile: (file: string) => Promise<void>;
    unstageFile: (file: string) => Promise<void>;
    strategies: BranchingStrategy[];
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
    const [strategies, setStrategies] = useState<BranchingStrategy[]>([]); // Typing fixed

    const [developers, setDevelopers] = useState<string[]>([]);
    const [developerSessions, setDeveloperSessions] = useState<Record<string, string>>({});
    const [activeDeveloper, setActiveDeveloper] = useState<string>('');
    const [pullRequests, setPullRequests] = useState<PullRequest[]>([]);

    // Init session on mount - Create Alice and Bob
    useEffect(() => {
        const init = async () => {
            // Only init if not already done
            if (developers.length > 0) return;

            // 1. Create Alice
            await addDeveloper('Alice');

            // 2. Create Bob
            await addDeveloper('Bob');

            // 3. Load Strategies
            try {
                const stratData = await gitService.fetchStrategies();
                setStrategies(stratData);
            } catch (e) {
                console.error("Failed to load strategies", e);
            }
        };
        init();
    }, []);

    const [sessionOutputs, setSessionOutputs] = useState<Record<string, string[]>>({});
    const [sessionCmdCounts, setSessionCmdCounts] = useState<Record<string, number>>({});

    // Terminal Transcript Store
    const [terminalTranscripts, setTerminalTranscripts] = useState<Record<string, TranscriptLine[]>>({});
    // Ref to access latest transcripts in callbacks without stale closures
    const terminalTranscriptsRef = useRef<Record<string, TranscriptLine[]>>({});

    // FIX: Use refs to avoid stale closure issues in async callbacks
    // These refs always hold the latest value
    const sessionOutputsRef = useRef<Record<string, string[]>>({});
    const sessionCmdCountsRef = useRef<Record<string, number>>({});

    // Keep refs in sync with state
    useEffect(() => {
        sessionOutputsRef.current = sessionOutputs;
    }, [sessionOutputs]);

    useEffect(() => {
        terminalTranscriptsRef.current = terminalTranscripts;
    }, [terminalTranscripts]);

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

    const runCommand = async (cmd: string): Promise<string[]> => {
        if (!sessionId) {
            console.error("No session ID");
            return [];
        }

        console.log(`Executing command: ${cmd} (Session: ${sessionId})`);
        // Store command echo BEFORE backend call (persists on tab switch)
        const commandEcho = `> ${cmd}`;
        setSessionOutputs(prev => {
            const current = prev[sessionId] || [];
            return { ...prev, [sessionId]: [...current, commandEcho] };
        });
        setState(prev => ({
            ...prev,
            output: [...prev.output, commandEcho]
        }));

        try {
            const data = await gitService.executeCommand(sessionId, cmd);
            console.log("GitAPI: Command response:", data);

            let newLines: string[] = [];
            if (data.error) {
                newLines = [`Error: ${data.error}`];
            } else if (data.output) {
                newLines = [data.output]; // Output is usually a single block string, terminal splits it
            }

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

            // Always fetch fresh state after command (using current sessionId)
            // This ensures currentPath and HEAD are updated before prompt is shown
            await fetchState(sessionId);

            // NOW increment command count - this triggers prompt rendering with correct state
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
            // Handle error store update
            setSessionOutputs(prev => ({ ...prev, [sessionId]: [...(prev[sessionId] || []), errorLine] }));
            setSessionCmdCounts(prev => ({ ...prev, [sessionId]: (prev[sessionId] || 0) + 1 }));

            setState(prev => ({ ...prev, output: [...prev.output, errorLine], commandCount: prev.commandCount + 1 }));
            return [errorLine];
        }
    };

    // --- Terminal Recording Implementation ---

    const appendToTranscript = (text: string, hasNewline: boolean = true) => {
        if (!sessionId) return;

        const line: TranscriptLine = { text, hasNewline };

        setTerminalTranscripts(prev => {
            const current = prev[sessionId] || [];
            return {
                ...prev,
                [sessionId]: [...current, line]
            };
        });
    };

    const getTranscript = (): TranscriptLine[] => {
        // Use ref to access latest state immediately
        return terminalTranscriptsRef.current[sessionId] || [];
    };

    const clearTranscript = () => {
        if (!sessionId) return;
        setTerminalTranscripts(prev => ({
            ...prev,
            [sessionId]: []
        }));
    };

    // -----------------------------------------

    const toggleShowAllCommits = () => {
        setShowAllCommits(prev => !prev);
    };

    const stageFile = async (file: string) => {
        await runCommand(`add ${file}`);
    };

    const unstageFile = async (file: string) => {
        await runCommand(`restore --staged ${file}`);
    };

    const addDeveloper = async (name: string) => {
        try {
            if (developers.includes(name)) return; // Prevent duplicates
            const data = await gitService.initSession();
            setDevelopers(prev => [...prev, name]);
            setDeveloperSessions(prev => ({ ...prev, [name]: data.sessionId }));
            if (!activeDeveloper) {
                setActiveDeveloper(name);
                setSessionId(data.sessionId);
                await fetchState(data.sessionId);
            }
        } catch (e) {
            console.error("Failed to add developer", e);
        }
    };

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
    }, [showAllCommits]);

    return (
        <GitContext.Provider value={{
            state,
            sessionId, // Expose current session ID
            runCommand,
            appendToTranscript,
            getTranscript,
            clearTranscript,
            showAllCommits,
            toggleShowAllCommits,
            stageFile,
            unstageFile,
            strategies,
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
        }}>
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
