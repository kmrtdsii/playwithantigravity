export interface Commit {
    id: string;
    message: string;
    parentId: string | null;
    secondParentId: string | null;
    branch: string;
    timestamp: string;
}

export interface TreeEntry {
    name: string;
    hash: string;
    type: 'tree' | 'blob';
}

export interface ObjectNode {
    id: string;
    type: 'tree' | 'blob' | 'commit';
    entries?: TreeEntry[]; // For Tree
    size?: number; // For Blob
    content?: string; // For Blob (preview)
    message?: string; // For Commit
    treeId?: string; // For Commit
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
    objects?: Record<string, ObjectNode>; // id -> ObjectNode
    remotes?: Remote[]; // Defined remotes


    output: string[];
    commandCount: number;
}
