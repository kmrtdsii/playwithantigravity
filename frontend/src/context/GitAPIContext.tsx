import React, { createContext, useContext, useState, useEffect } from 'react';

// Types (shared with Backend ideally, but defined here for now)
export interface Commit {
    id: string;
    message: string;
    parentId: string | null;
    secondParentId: string | null;
    branch: string;
    timestamp: string;
}

export interface GitState {
    initialized: boolean;
    commits: Commit[];
    branches: Record<string, string>; // branchName -> commitId
    HEAD: { type: 'branch' | 'commit', ref: string | null, id?: string };
    staging: string[];
    modified: string[];
    untracked: string[];
    fileStatuses: Record<string, string>;
    files: string[];

    output: string[];
    commandCount: number;
}

interface GitContextType {
    state: GitState;
    runCommand: (cmd: string) => Promise<void>;
}

const GitContext = createContext<GitContextType | undefined>(undefined);

export const GitProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
    console.log("GitProvider: Mounted");
    const [state, setState] = useState<GitState>({
        initialized: false,
        commits: [],
        branches: {},
        HEAD: { type: 'branch', ref: null },
        staging: [],
        modified: [],
        untracked: [],
        fileStatuses: {},
        files: [],

        output: [],
        commandCount: 0
    });

    const [sessionId, setSessionId] = useState<string>('');

    // Init session on mount
    useEffect(() => {
        const init = async () => {
            try {
                const res = await fetch('/api/session/init', { method: 'POST' });
                const data = await res.json();
                console.log("GitAPI: Session init response:", data); // Log the session init response
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
            const res = await fetch(`/api/state?sessionId=${sid}&t=${Date.now()}`);
            if (!res.ok) throw new Error('Failed to fetch state');
            const data = await res.json();

            // Transform backend state to frontend structure if needed
            // Currently they match closely
            setState(prev => ({
                ...prev,
                commits: data.commits || [],
                branches: data.branches || {},
                HEAD: data.HEAD || { type: 'branch', ref: 'main' },
                files: data.files || [],
                staging: data.staging || [],
                modified: data.modified || [],
                untracked: data.untracked || [],
                fileStatuses: data.fileStatuses || {},
                initialized: true
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
            const res = await fetch('/api/command', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ sessionId, command: cmd })
            });
            const data = await res.json();

            console.log("GitAPI: Command response:", data);

            if (data.error) {
                setState(prev => ({ ...prev, output: [...prev.output, `Error: ${data.error}`] }));
            } else if (data.output) {
                setState(prev => ({ ...prev, output: [...prev.output, data.output] }));
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

    return (
        <GitContext.Provider value={{ state, runCommand }}>
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
