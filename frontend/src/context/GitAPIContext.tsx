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
    isSandbox: boolean;
    isForking: boolean;
    enterSandbox: () => Promise<void>;
    exitSandbox: () => Promise<void>;
    resetSandbox: () => Promise<void>;
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

    const [sessionId, setSessionId] = useState<string>('');
    const [realSessionId, setRealSessionId] = useState<string>('');
    const [isSandbox, setIsSandbox] = useState<boolean>(false);
    const [isForking, setIsForking] = useState<boolean>(false);
    const [showAllCommits, setShowAllCommits] = useState<boolean>(false);
    const [strategies, setStrategies] = useState<any[]>([]);

    const [developers, setDevelopers] = useState<string[]>([]);
    const [developerSessions, setDeveloperSessions] = useState<Record<string, string>>({});
    const [activeDeveloper, setActiveDeveloper] = useState<string>('');
    const [pullRequests, setPullRequests] = useState<PullRequest[]>([]);

    // Init session on mount
    useEffect(() => {
        const init = async () => {
            try {
                // Init Session
                const data = await gitService.initSession();
                console.log("GitAPI: Session init response:", data);
                if (data.sessionId) {
                    setSessionId(data.sessionId);
                    setRealSessionId(data.sessionId); // Store original
                    await fetchState(data.sessionId);
                }

                // Load Strategies
                const stratData = await gitService.fetchStrategies();
                setStrategies(stratData);
            } catch (e) {
                console.error("Failed to init session or load strategies", e);
                setState(prev => ({ ...prev, output: [...prev.output, "Error connecting to server"] }));
            }
        };
        init();
    }, []);

    const enterSandbox = async () => {
        if (isSandbox || isForking) return;

        setIsForking(true);
        try {
            const sandboxId = `sandbox-${Date.now()}`;
            console.log(`Creating sandbox: ${sandboxId} from ${realSessionId}`);

            await gitService.forkSession(realSessionId, sandboxId);
            setSessionId(sandboxId);
            setIsSandbox(true);

            setState(prev => ({
                ...prev,
                output: [...prev.output, "--- SANDBOX MODE ENABLED (Experimental changes only) ---"]
            }));
            await fetchState(sandboxId);
        } catch (e) {
            console.error("Failed to enter sandbox", e);
            setState(prev => ({ ...prev, output: [...prev.output, "Failed to enter Sandbox mode"] }));
        } finally {
            setIsForking(false);
        }
    };

    const resetSandbox = async () => {
        if (!isSandbox || isForking) return;

        setIsForking(true);
        try {
            const sandboxId = `sandbox-${Date.now()}`;
            console.log(`Resetting sandbox: ${sandboxId} from ${realSessionId}`);

            await gitService.forkSession(realSessionId, sandboxId);
            setSessionId(sandboxId);

            setState(prev => ({
                ...prev,
                output: [...prev.output, "--- SANDBOX RESET (State refreshed from main session) ---"]
            }));
            await fetchState(sandboxId);
        } catch (e) {
            console.error("Failed to reset sandbox", e);
            setState(prev => ({ ...prev, output: [...prev.output, "Failed to reset Sandbox"] }));
        } finally {
            setIsForking(false);
        }
    };

    const exitSandbox = async () => {
        if (!isSandbox) return;

        // Discard sandbox session (just switch back)
        console.log("Exiting sandbox, returning to:", realSessionId);
        setSessionId(realSessionId);
        setIsSandbox(false);
        setState(prev => ({
            ...prev,
            output: [...prev.output, "--- SANDBOX MODE DISABLED (Changes discarded) ---"]
        }));
        await fetchState(realSessionId);
    };

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
            const data = await gitService.initSession();
            // In a real multi-user app, we'd name the session more uniquely
            // For now, we use the ID from server or decorate it
            // Actually server init session returns a fixed ID currently, 
            // we should ideally pass preferred ID or get a unique one.
            // Let's just use the returned one and assume we create new ones.
            // Update: Current backend handleInitSession returns "user-session-1" (fixed).
            // I should probably update backend to be more unique.

            setDevelopers(prev => [...prev, name]);
            setDeveloperSessions(prev => ({ ...prev, [name]: data.sessionId }));
            if (!activeDeveloper) {
                setActiveDeveloper(name);
                setSessionId(data.sessionId);
                setRealSessionId(data.sessionId);
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
            setRealSessionId(sid);
            setIsSandbox(false); // Reset sandbox when switching?
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
            isSandbox,
            isForking,
            enterSandbox,
            exitSandbox,
            resetSandbox,
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
            resetRemote
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
