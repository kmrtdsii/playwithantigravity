export interface Commit {
    id: string;
    message: string;
    parentId: string | null;
    secondParentId: string | null;
    branch: string;
    timestamp: string;
}


export interface Remote {
    name: string;
    urls: string[];
}

export interface GitState {
    initialized: boolean;
    commits: Commit[];
    branches: Record<string, string>; // branchName -> commitId
    remoteBranches: Record<string, string>; // remote/branchName -> commitId
    tags: Record<string, string>; // tagName -> commitId
    references: Record<string, string>; // references like ORIG_HEAD -> commitId
    HEAD: { type: 'branch' | 'commit' | 'none', ref: string | null, id?: string };
    potentialCommits: Commit[];
    staging: string[];
    modified: string[];
    untracked: string[];
    fileStatuses: Record<string, string>;
    files: string[];
    currentPath?: string;
    projects?: string[];
    remotes?: Remote[]; // Defined remotes
    sharedRemotes?: string[];


    output: string[];
    commandCount: number;
    _sessionId?: string;
}

export interface BranchingStrategy {
    id: string;
    name: string;
    description: string;
    mainBranch: string;
    flowSteps: string[];
}

export type PullRequestStatus = 'OPEN' | 'MERGED' | 'CLOSED';

export interface PullRequest {
    id: number;
    title: string;
    description: string;
    sourceBranch: string;
    targetBranch: string;
    status: PullRequestStatus;
    creator: string;
    createdAt: string;
}
