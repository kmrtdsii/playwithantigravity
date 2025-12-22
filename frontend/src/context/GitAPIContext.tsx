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
    const [showAllCommits, setShowAllCommits] = useState<boolean>(false);

    // Init session on mount
    useEffect(() => {
        const init = async () => {
            try {
                const data = await gitService.initSession();
                console.log("GitAPI: Session init response:", data);
                if (data.sessionId) {
                    setSessionId(data.sessionId);
                    await fetchState(data.sessionId);
                }
            } catch (e) {
                console.error("Failed to init session", e);
                setState(prev => ({ ...prev, output: [...prev.output, "Error connecting to server"] }));
            }
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

    const runCommand = async (cmd: string) => {
        if (!sessionId) {
            console.error("No session ID");
            return;
        }

        console.log(`Executing command: ${cmd}`);
        try {
            const data = await gitService.executeCommand(sessionId, cmd);
            console.log("GitAPI: Command response:", data);

            if (data.error) {
                setState(prev => ({ ...prev, output: [...prev.output, `Error: ${data.error}`] }));
            } else if (data.output) {
                setState(prev => ({ ...prev, output: [...prev.output, data.output || ""] }));
            }

            // Always fetch fresh state after command
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
        <GitContext.Provider value={{ state, runCommand, showAllCommits, toggleShowAllCommits, stageFile, unstageFile }}>
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
