import React, { createContext, useContext, useState, useEffect } from 'react';
import type { GitState, PullRequest, BranchingStrategy } from '../types/gitTypes';
import { gitService } from '../services/gitService';
import { filterReachableCommits } from '../utils/graphUtils';

interface GitContextType {
    state: GitState;
    runCommand: (cmd: string) => Promise<void>;
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



    const fetchState = async (sid: string) => {
        try {
            const newState = await gitService.fetchState(sid, showAllCommits);
            // When fetching state, we must ensure we are updating the "active" view state
            // But if we switched users, we want to load THAT user's output/count.
            // Since this function is async, by the time it returns, sessionId state might match sid.

            setState(prev => {
                const storedOutput = sessionOutputs[sid] || [];
                const storedCount = sessionCmdCounts[sid] || 0;

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
                    commandCount: storedCount
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

    const runCommand = async (cmd: string) => {
        if (!sessionId) {
            console.error("No session ID");
            return;
        }

        console.log(`Executing command: ${cmd} (Session: ${sessionId})`);
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

            // Update Command Count Store
            setSessionCmdCounts(prev => {
                const current = prev[sessionId] || 0;
                return { ...prev, [sessionId]: current + 1 };
            });

            // Update Transient State (for immediate UI reflection)
            setState(prev => ({
                ...prev,
                // Simplified: Update local state immediately with new lines appended
                // CAUTION: prev.output might be stale if store updated? 
                // Let's rely on fetchState or manual sync.
                // Safest: Append to prev.output.
                output: [...prev.output, ...newLines],
                commandCount: prev.commandCount + 1
            }));

            // Always fetch fresh state after command (using current sessionId)
            // This will re-sync state.output from sessionOutputs via fetchState logic below?
            // Wait, fetchState uses sessionOutputs[sid]. We just scheduled a setSessionOutputs.
            // React batching might mean fetchState sees OLD sessionOutputs.
            // But visual terminal limits append-only.
            await fetchState(sessionId);

            // AUTO-REFRESH SERVER STATE
            if (serverState && serverState.remotes?.length === 0) {
                await fetchServerState('origin');
            } else if (serverState) {
                await fetchServerState('origin'); // Default fallback
            }

        } catch (e) {
            console.error(e);
            // Handle error store update
            setSessionOutputs(prev => ({ ...prev, [sessionId]: [...(prev[sessionId] || []), "Network error"] }));
            setSessionCmdCounts(prev => ({ ...prev, [sessionId]: (prev[sessionId] || 0) + 1 }));

            setState(prev => ({ ...prev, output: [...prev.output, "Network error"], commandCount: prev.commandCount + 1 }));
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
            runCommand,
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
