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
    staging: string[];
    modified: string[];
    untracked: string[];
    fileStatuses: Record<string, string>;
    files: string[];
    currentPath?: string;
    projects?: string[];
    remotes?: Remote[]; // Defined remotes


    output: string[];
    commandCount: number;
}

export interface BranchingStrategy {
    id: string;
    name: string;
    description: string;
    mainBranch: string;
    flowSteps: string[];
}
