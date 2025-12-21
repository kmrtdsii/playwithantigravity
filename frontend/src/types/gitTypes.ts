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
    references: Record<string, string>; // references like ORIG_HEAD -> commitId
    HEAD: { type: 'branch' | 'commit', ref: string | null, id?: string };
    staging: string[];
    modified: string[];
    untracked: string[];
    fileStatuses: Record<string, string>;
    files: string[];

    output: string[];
    commandCount: number;
}
