import React, { createContext, useContext, useState, useEffect } from 'react';
import type { GitState, PullRequest } from '../types/gitTypes';
import { gitService } from '../services/gitService';

interface GitContextType {
    state: GitState;
    runCommand: (cmd: string) => Promise<void>;
    showAllCommits: boolean;
    toggleShowAllCommits: () => void;
    stageFile: (file: string) => Promise<void>;
    unstageFile: (file: string) => Promise<void>;
    strategies: any[];
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
    const [strategies, setStrategies] = useState<any[]>([]);

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
        };
        init();
    }, []);

    const fetchState = async (sid: string) => {
        try {
            const newState = await gitService.fetchState(sid, showAllCommits);
            setState(prev => ({
                ...prev,
                ...newState,
                // Preserve UI state that isn't in backend response
                output: prev.output,
                commandCount: prev.commandCount
            }));
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

            if (data.error) {
                setState(prev => ({ ...prev, output: [...prev.output, `Error: ${data.error}`] }));
            } else if (data.output) {
                setState(prev => ({ ...prev, output: [...prev.output, data.output || ""] }));
            }

            // Always fetch fresh state after command (using current sessionId)
            await fetchState(sessionId);

            // Increment command count to signal terminal
            setState(prev => ({ ...prev, commandCount: prev.commandCount + 1 }));

            // AUTO-REFRESH SERVER STATE
            // If we have a server state loaded, refresh it too, as push might have updated it.
            // We blindly refresh 'origin' for now or whatever is active.
            if (serverState && serverState.remotes?.length === 0) {
                // Wait, serverState.remotes was CLEARED by backend for visualization.
                // We need to know which remote we are visualizing.
                // For now, let's just refresh 'origin' if it exists.
                await fetchServerState('origin');
            } else if (serverState) {
                await fetchServerState('origin'); // Default fallback
            }

        } catch (e) {
            console.error(e);
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
        await fetchState(sessionId); // Refresh graph as remote might have changed
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
