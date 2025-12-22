import React, { createContext, useContext, useState, useEffect } from 'react';
import type { GitState } from '../types/gitTypes';
import { gitService } from '../services/gitService';

interface GitContextType {
    state: GitState;
    runCommand: (cmd: string) => Promise<void>;
    showAllCommits: boolean;
    toggleShowAllCommits: () => void;
    stageFile: (file: string) => Promise<void>;
    unstageFile: (file: string) => Promise<void>;
    isSandbox: boolean;
    enterSandbox: () => Promise<void>;
    exitSandbox: () => Promise<void>;
    strategies: any[];
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
        output: [],
        commandCount: 0
    });

    const [sessionId, setSessionId] = useState<string>('');
    const [realSessionId, setRealSessionId] = useState<string>('');
    const [isSandbox, setIsSandbox] = useState<boolean>(false);
    const [showAllCommits, setShowAllCommits] = useState<boolean>(false);
    const [strategies, setStrategies] = useState<any[]>([]);

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
        if (isSandbox) return;

        try {
            const sandboxId = `sandbox-${Date.now()}`;
            console.log(`Creating sandbox: ${sandboxId} from ${realSessionId}`);

            await gitService.forkSession(realSessionId, sandboxId);
            setSessionId(sandboxId);
            setIsSandbox(true);

            // Clear output for fresh feel or keep? Maybe output message.
            setState(prev => ({
                ...prev,
                output: [...prev.output, "--- SANDBOX MODE ENABLED (Experimental changes only) ---"]
            }));
            await fetchState(sandboxId);
        } catch (e) {
            console.error("Failed to enter sandbox", e);
            setState(prev => ({ ...prev, output: [...prev.output, "Failed to enter Sandbox mode"] }));
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
            enterSandbox,
            exitSandbox,
            strategies
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
