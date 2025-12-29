import React, { createContext, useContext, useCallback } from 'react';
import type { GitState, PullRequest } from '../types/gitTypes';
import { useTerminalTranscript, type TranscriptLine } from '../hooks/useTerminalTranscript';
import { useGitSession } from '../hooks/useGitSession';
import { useGitData } from '../hooks/useGitData';
import { useGitCommand } from '../hooks/useGitCommand';

interface GitContextType {
    state: GitState;
    sessionId: string;
    setSessionId: (id: string) => void;
    runCommand: (cmd: string, options?: { silent?: boolean; skipRefresh?: boolean }) => Promise<string[]>;
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
    removeDeveloper: (name: string) => Promise<void>;
    pullRequests: PullRequest[];
    refreshPullRequests: () => Promise<void>;
    ingestRemote: (name: string, url: string, depth?: number) => Promise<void>;
    createPullRequest: (title: string, desc: string, source: string, target: string) => Promise<void>;
    mergePullRequest: (id: number) => Promise<void>;
    deletePullRequest: (id: number) => Promise<void>;
    resetRemote: (name?: string) => Promise<void>;
    refreshState: () => Promise<void>;
    serverState: GitState | null;
    fetchServerState: (name: string) => Promise<void>;
}

const GitContext = createContext<GitContextType | undefined>(undefined);

export const GitProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
    // 1. Session Management
    const {
        sessionId,
        setSessionId, // Expose for Mission Context
        developers,
        activeDeveloper,
        switchDeveloper: rawSwitchDeveloper,
        addDeveloper: rawAddDeveloper,
        removeDeveloper,
    } = useGitSession();

    // 2. Data Management (State, PRs, Server)
    const gitData = useGitData(sessionId);
    const {
        state,
        serverState,
        pullRequests,
        showAllCommits,
        toggleShowAllCommits,
        fetchState,
        refreshPullRequests,
        fetchServerState,
    } = gitData;

    // 3. Command Execution
    const { runCommand } = useGitCommand({ sessionId, gitData });

    // 4. Terminal Transcript
    const { terminalTranscripts, appendToTranscript, clearTranscript } = useTerminalTranscript(sessionId);

    // --- Wrappers and Composite Actions ---

    // Wrap switchDeveloper to fetch state after switch
    const switchDeveloper = useCallback(async (name: string) => {
        await rawSwitchDeveloper(name);
        // Note: rawSwitchDeveloper updates state async. fetchState depends on sessionId.
        // We might need an effect to fetchState when sessionId changes, which we HAVE in useGitData.
        // But invalidating cache/forcing refresh might be good?
        // Actually useGitData has an effect interacting with sessionId
    }, [rawSwitchDeveloper]);

    // Wrap addDeveloper
    const addDeveloper = useCallback(async (name: string) => {
        await rawAddDeveloper(name);
    }, [rawAddDeveloper]);


    const stageFile = useCallback(async (file: string) => {
        await runCommand(`add ${file}`);
    }, [runCommand]);

    const unstageFile = useCallback(async (file: string) => {
        await runCommand(`restore --staged ${file}`);
    }, [runCommand]);

    const ingestRemote = useCallback(async (name: string, url: string, depth?: number) => {
        // We use gitService inside hooks, but ingestRemote is complex?
        // Actually runCommand implementation encapsulates service logic mostly.
        // But ingestRemote is distinct service call.
        // We can just import service here or put in useGitCommand?
        // Let's keep it clean and use service + fetchState.
        const { gitService } = await import('../services/gitService');
        await gitService.ingestRemote(name, url, depth);
        await fetchState(sessionId);
    }, [sessionId, fetchState]);

    const createPullRequest = useCallback(async (title: string, desc: string, source: string, target: string) => {
        const { gitService } = await import('../services/gitService');
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
        const { gitService } = await import('../services/gitService');
        await gitService.mergePullRequest(id);
        await refreshPullRequests();
        await fetchServerState('origin');
        if (sessionId) await fetchState(sessionId);
    }, [sessionId, refreshPullRequests, fetchServerState, fetchState]);

    const deletePullRequest = useCallback(async (id: number) => {
        const { gitService } = await import('../services/gitService');
        await gitService.deletePullRequest(id);
        await refreshPullRequests();
        await fetchServerState('origin');
    }, [refreshPullRequests, fetchServerState]);

    const resetRemote = useCallback(async (name: string = 'origin') => {
        const { gitService } = await import('../services/gitService');
        await gitService.resetRemote(name);
        await fetchState(sessionId);
    }, [sessionId, fetchState]);

    const refreshStateWrapper = useCallback(async () => {
        if (sessionId) await fetchState(sessionId);
    }, [sessionId, fetchState]);

    const contextValue = React.useMemo(() => ({
        state,
        sessionId,
        setSessionId,
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
        removeDeveloper,
        pullRequests,
        refreshPullRequests,
        ingestRemote,
        createPullRequest,
        mergePullRequest,
        deletePullRequest,
        resetRemote,
        refreshState: refreshStateWrapper,
        serverState,
        fetchServerState
    }), [
        state,
        sessionId,
        setSessionId,
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
        removeDeveloper,
        pullRequests,
        refreshPullRequests,
        ingestRemote,
        createPullRequest,
        mergePullRequest,
        deletePullRequest,
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
