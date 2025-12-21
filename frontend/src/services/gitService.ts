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
            references: data.references || {},
            HEAD: data.HEAD || { type: 'branch', ref: 'main' },
            files: data.files || [],
            staging: data.staging || [],
            modified: data.modified || [],
            untracked: data.untracked || [],
            fileStatuses: data.fileStatuses || {},
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
    }
};
