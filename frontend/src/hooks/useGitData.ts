import { useState, useCallback, useRef, useEffect } from 'react';
import type { GitState, PullRequest } from '../types/gitTypes';
import { gitService, type DirectoryNode } from '../services/gitService';
import { filterReachableCommits } from '../utils/filterReachableCommits';

export interface GitDataHook {
    state: GitState;
    serverState: GitState | null;
    pullRequests: PullRequest[];
    showAllCommits: boolean;
    toggleShowAllCommits: () => void;
    fetchState: (sid: string) => Promise<void>;
    fetchServerState: (name: string) => Promise<void>;
    refreshPullRequests: () => Promise<void>;
    setState: React.Dispatch<React.SetStateAction<GitState>>;
    setServerState: React.Dispatch<React.SetStateAction<GitState | null>>;
    updateCommandOutput: (sid: string, output: string[]) => void;
    incrementCommandCount: (sid: string) => void;
    workspaceTree: DirectoryNode[];
    currentRepo: string;
    fetchWorkspaceTree: (sid: string) => Promise<void>;
}

const INITIAL_STATE: GitState = {
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
};

export const useGitData = (sessionId: string): GitDataHook => {
    const [state, setState] = useState<GitState>(INITIAL_STATE);
    const [serverState, setServerState] = useState<GitState | null>(null);
    const [pullRequests, setPullRequests] = useState<PullRequest[]>([]);
    const [showAllCommits, setShowAllCommits] = useState<boolean>(false);

    // Workspace Tree State
    const [workspaceTree, setWorkspaceTree] = useState<DirectoryNode[]>([]);
    const [currentRepo, setCurrentRepo] = useState<string>('');

    // Session specific storage for output/counts to persist across user switches
    const [sessionOutputs, setSessionOutputs] = useState<Record<string, string[]>>({});
    const [sessionCmdCounts, setSessionCmdCounts] = useState<Record<string, number>>({});

    const sessionOutputsRef = useRef(sessionOutputs);
    const sessionCmdCountsRef = useRef(sessionCmdCounts);

    useEffect(() => {
        sessionOutputsRef.current = sessionOutputs;
    }, [sessionOutputs]);

    useEffect(() => {
        sessionCmdCountsRef.current = sessionCmdCounts;
    }, [sessionCmdCounts]);

    const updateCommandOutput = useCallback((sid: string, output: string[]) => {
        setSessionOutputs(prev => ({
            ...prev,
            [sid]: output
        }));
    }, []);

    const incrementCommandCount = useCallback((sid: string) => {
        setSessionCmdCounts(prev => ({
            ...prev,
            [sid]: (prev[sid] || 0) + 1
        }));
    }, []);

    const fetchState = useCallback(async (sid: string) => {
        if (!sid) return;
        try {
            const newState = await gitService.fetchState(sid, showAllCommits);

            setState(prev => {
                const storedOutput = sessionOutputsRef.current[sid] || [];
                const storedCount = sessionCmdCountsRef.current[sid] || 0;

                const finalCommits = showAllCommits
                    ? newState.commits
                    : filterReachableCommits(newState.commits, newState);

                return {
                    ...prev,
                    ...newState,
                    commits: finalCommits,
                    output: storedOutput,
                    commandCount: storedCount,
                    _sessionId: sid
                };
            });
        } catch (e) {
            console.error("fetchState failed", e);
        }
    }, [showAllCommits]);

    const fetchServerState = useCallback(async (name: string) => {
        try {
            const sState = await gitService.getRemoteState(name);
            setServerState(sState);
        } catch (e) {
            console.error("Failed to fetch server state", e);
            setServerState(null);
        }
    }, []);

    const refreshPullRequests = useCallback(async () => {
        try {
            const prs = await gitService.fetchPullRequests();
            setPullRequests(prs);
        } catch (e) {
            console.error("Failed to fetch PRs", e);
        }
    }, []);

    const toggleShowAllCommits = useCallback(() => {
        setShowAllCommits(prev => !prev);
    }, []);

    const fetchWorkspaceTree = useCallback(async (sid: string) => {
        if (!sid) return;
        try {
            const data = await gitService.getWorkspaceTree(sid);
            setWorkspaceTree(data.tree);
            setCurrentRepo(data.currentRepo);
        } catch (e) {
            console.error("Failed to fetch workspace tree", e);
        }
    }, []);

    // Re-fetch when showAllCommits changes
    useEffect(() => {
        if (sessionId) {
            fetchState(sessionId);
        }
    }, [sessionId, showAllCommits, fetchState]);

    return {
        state,
        serverState,
        pullRequests,
        showAllCommits,
        toggleShowAllCommits,
        fetchState,
        fetchServerState,
        refreshPullRequests,
        setState,
        setServerState,
        updateCommandOutput,
        incrementCommandCount,
        workspaceTree,
        currentRepo,
        fetchWorkspaceTree
    };
};
