import type { GitState } from '../types/gitTypes';

interface InitResponse {
    status: string;
    sessionId: string;
}

interface CommandResponse {
    output?: string;
    error?: string;
}

export const gitService = {
    async initSession(): Promise<InitResponse> {
        const res = await fetch('/api/session/init', { method: 'POST' });
        if (!res.ok) throw new Error('Failed to init session');
        return res.json();
    },

    async fetchState(sessionId: string, showAll: boolean = false): Promise<GitState> {
        const res = await fetch(`/api/state?sessionId=${sessionId}&t=${Date.now()}&showAll=${showAll}`);
        if (!res.ok) throw new Error('Failed to fetch state');
        const data = await res.json();

        // Ensure default structure matches GitState interface
        return {
            commits: data.commits || [],
            branches: data.branches || {},
            tags: data.tags || {},
            references: data.references || {},
            remotes: data.remotes || [],
            remoteBranches: data.remoteBranches || {},
            HEAD: data.HEAD || { type: 'none' },
            files: data.files || [],
            potentialCommits: data.potentialCommits || [],
            staging: data.staging || [],
            modified: data.modified || [],
            untracked: data.untracked || [],
            fileStatuses: data.fileStatuses || {},
            currentPath: data.currentPath || '',
            projects: data.projects || [],
            sharedRemotes: data.sharedRemotes || [],
            initialized: true,
            output: [], // State API doesn't return output history
            commandCount: 0 // Managed by context
        };
    },

    async executeCommand(sessionId: string, cmd: string): Promise<CommandResponse> {
        const res = await fetch('/api/command', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ sessionId, command: cmd })
        });
        if (!res.ok) throw new Error('Failed to execute command');
        return res.json();
    },

    async fetchStrategies(): Promise<any[]> {
        const res = await fetch('/api/strategies');
        if (!res.ok) throw new Error('Failed to fetch strategies');
        return res.json();
    },

    async ingestRemote(name: string, url: string): Promise<void> {
        const res = await fetch('/api/remote/ingest', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name, url })
        });
        if (!res.ok) throw new Error('Failed to ingest remote');
    },

    async fetchPullRequests(): Promise<any[]> {
        const res = await fetch('/api/remote/pull-requests');
        if (!res.ok) throw new Error('Failed to fetch pull requests');
        return res.json();
    },

    async createPullRequest(pr: { title: string; description: string; sourceBranch: string; targetBranch: string; creator: string }): Promise<any> {
        const res = await fetch('/api/remote/pull-requests/create', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(pr)
        });
        if (!res.ok) throw new Error('Failed to create pull request');
        return res.json();
    },

    async mergePullRequest(id: number, remoteName: string = 'origin'): Promise<void> {
        const res = await fetch('/api/remote/pull-requests/merge', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ id, remoteName })
        });
        if (!res.ok) throw new Error('Failed to merge pull request');
    },

    async resetRemote(name: string = 'origin'): Promise<void> {
        const res = await fetch('/api/remote/reset', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name })
        });
        if (!res.ok) throw new Error('Failed to reset remote');
    }
};
