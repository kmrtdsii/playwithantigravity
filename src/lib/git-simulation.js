import { v4 as uuidv4 } from 'uuid';
export const ACTION_TYPES = {
    INIT: 'INIT',
    ADD: 'ADD',
    COMMIT: 'COMMIT',
    CHECKOUT: 'CHECKOUT',
    BRANCH: 'BRANCH',
    MERGE: 'MERGE',
    STATUS: 'STATUS',
    LOG: 'LOG'
};

export const initialState = {
    initialized: false,
    commits: [], // Array of { id, message, parentId, timestamp }
    branches: {}, // Map branchName -> commitId
    HEAD: { type: 'branch', ref: null }, // { type: 'branch', ref: 'name' } or { type: 'commit', id: 'hash' }
    staging: [], // Array of file paths
    output: [], // Command output history
    lastUpdated: 0 // Timestamp to force updates on silent commands
};

export function gitReducer(state, action) {
    switch (action.type) {
        case ACTION_TYPES.INIT:
            if (state.initialized) {
                return {
                    ...state,
                    output: [...state.output, 'Reinitialized existing Git repository']
                };
            }
            return {
                ...state,
                initialized: true,
                branches: { main: null },
                HEAD: { type: 'branch', ref: 'main' },
                staging: [],
                output: [...state.output, 'Initialized empty Git repository']
            };

        case ACTION_TYPES.ADD: {
            if (!state.initialized) return { ...state, output: [...state.output, 'fatal: not a git repository'] };
            const { files } = action.payload; // array of filenames
            // Simplified: just add everything to staging if it's not already there
            const newStaging = [...new Set([...state.staging, ...files])];
            return {
                ...state,
                staging: newStaging,
                output: state.output,
                lastUpdated: Date.now()
            };
        }

        case ACTION_TYPES.COMMIT: {
            if (!state.initialized) return { ...state.output, output: [...state.output, 'fatal: not a git repository'] };

            const { message } = action.payload;

            // Check if nothing to commit?
            // For simulation fun, let's allow empty commits if they forced it, 
            // OR if we didn't implement 'add' fully, maybe we auto-stage?
            // User wants "Authentic", so we should check staging?
            // But currently we don't track Modified vs Staged fully.
            // Let's assume if staging is empty, "nothing to commit".
            if (state.staging.length === 0) {
                return { ...state, output: [...state.output, 'nothing to commit (create/copy files and use "git add" to track)'] };
            }

            const newCommitId = uuidv4().substring(0, 7); // Short hash

            // Resolve current HEAD commit ID
            let parentId = null;
            if (state.HEAD.type === 'branch') {
                parentId = state.branches[state.HEAD.ref];
            } else {
                parentId = state.HEAD.id;
            }

            const newCommit = {
                id: newCommitId,
                message,
                parentId,
                secondParentId: null, // For merges
                timestamp: new Date().toISOString()
            };

            // Update branch pointer if HEAD is on a branch
            let newBranches = { ...state.branches };
            if (state.HEAD.type === 'branch') {
                newBranches[state.HEAD.ref] = newCommitId;
            }

            const newState = {
                ...state,
                commits: [...state.commits, newCommit],
                branches: newBranches,
                staging: [], // clear staging after commit
                output: [...state.output, `[${state.HEAD.type === 'branch' ? state.HEAD.ref : 'detached'} ${newCommitId}] ${message}`]
            };

            if (state.HEAD.type === 'commit') {
                newState.HEAD = { type: 'commit', id: newCommitId };
            }

            return newState;
        }

        case ACTION_TYPES.BRANCH: {
            if (!state.initialized) return { ...state, output: [...state.output, 'fatal: not a git repository'] };
            const { name } = action.payload;

            if (state.branches[name]) {
                return { ...state, output: [...state.output, `fatal: A branch named '${name}' already exists.`] };
            }

            // Get current commit ID
            let currentCommitId = null;
            if (state.HEAD.type === 'branch') {
                currentCommitId = state.branches[state.HEAD.ref];
            } else {
                currentCommitId = state.HEAD.id;
            }

            if (!currentCommitId) {
                return { ...state, output: [...state.output, `fatal: Not a valid object name: '${name}'. (No commits yet)`] };
            }

            return {
                ...state,
                branches: { ...state.branches, [name]: currentCommitId },
                output: [...state.output, `Created branch ${name}`]
            };
        }

        case ACTION_TYPES.CHECKOUT: {
            if (!state.initialized) return { ...state, output: [...state.output, 'fatal: not a git repository'] };
            const { ref, isNewBranch } = action.payload;

            if (isNewBranch) {
                // Equivalent to git checkout -b <name>
                if (state.branches[ref]) {
                    return { ...state, output: [...state.output, `fatal: A branch named '${ref}' already exists.`] };
                }

                // Create branch at current HEAD
                let currentCommitId = state.HEAD.type === 'branch'
                    ? state.branches[state.HEAD.ref]
                    : state.HEAD.id;

                // If no commits yet and creating master -> allowed? Yes, but pointer is null.

                return {
                    ...state,
                    branches: { ...state.branches, [ref]: currentCommitId },
                    HEAD: { type: 'branch', ref: ref },
                    output: [...state.output, `Switched to a new branch '${ref}'`]
                };
            }

            // Normal checkout
            if (state.branches[ref] !== undefined) {
                return {
                    ...state,
                    HEAD: { type: 'branch', ref: ref },
                    output: [...state.output, `Switched to branch '${ref}'`]
                };
            }
            // Checkout commit hash (Detached HEAD)
            const commit = state.commits.find(c => c.id === ref || c.id.startsWith(ref));
            if (commit) {
                return {
                    ...state,
                    HEAD: { type: 'commit', id: commit.id },
                    output: [...state.output, `Note: switching to '${ref}'.\n\nYou are in 'detached HEAD' state...`, `HEAD is now at ${commit.id} ${commit.message}`]
                };
            }

            return { ...state, output: [...state.output, `error: pathspec '${ref}' did not match any file(s) known to git`] };
        }

        case ACTION_TYPES.MERGE: {
            if (!state.initialized) return { ...state, output: [...state.output, 'fatal: not a git repository'] };
            const { source } = action.payload; // branch name or commit

            // Resolve source commit
            let sourceCommitId = state.branches[source];
            if (!sourceCommitId) {
                // Try finding by hash
                const c = state.commits.find(c => c.id === source);
                if (c) sourceCommitId = c.id;
                else return { ...state, output: [...state.output, `merge: ${source} - not something we can merge`] };
            }

            // Resolve HEAD
            let headCommitId = state.HEAD.type === 'branch' ? state.branches[state.HEAD.ref] : state.HEAD.id;

            if (sourceCommitId === headCommitId) {
                return { ...state, output: [...state.output, 'Already up to date.'] };
            }

            // Check Ancestry for Fast-Forward
            // (Simplified: if head is ancestor of source, fast-forward)
            // (If source is ancestor of head, do nothing "Already up to date")

            // TODO: Real ancestry check using DFS/BFS?
            // For now, let's just make a Merge Commit always unless trivial?
            // No, Fast-forward is key.

            // Let's implement a simple ancestry check
            // isAncestor(ancestor, decendant)
            const isAncestor = (aId, dId) => {
                if (!dId) return false;
                if (aId === dId) return true;
                const commit = state.commits.find(c => c.id === dId);
                if (!commit) return false;
                if (commit.parentId && isAncestor(aId, commit.parentId)) return true;
                if (commit.secondParentId && isAncestor(aId, commit.secondParentId)) return true;
                return false;
            };

            if (isAncestor(headCommitId, sourceCommitId)) {
                // Fast-forward
                // Move HEAD to source
                let newBranches = { ...state.branches };
                if (state.HEAD.type === 'branch') {
                    newBranches[state.HEAD.ref] = sourceCommitId;
                } else {
                    // detached head moves
                }
                return {
                    ...state,
                    branches: newBranches,
                    HEAD: state.HEAD.type === 'commit' ? { type: 'commit', id: sourceCommitId } : state.HEAD,
                    output: [...state.output, `Updating ${headCommitId || 'null'}..${sourceCommitId}`, 'Fast-forward']
                };
            }

            // Merge Commit
            const newCommitId = uuidv4().substring(0, 7);
            const newCommit = {
                id: newCommitId,
                message: `Merge branch '${source}'`,
                parentId: headCommitId,
                secondParentId: sourceCommitId,
                timestamp: new Date().toISOString()
            };

            let newBranches = { ...state.branches };
            if (state.HEAD.type === 'branch') {
                newBranches[state.HEAD.ref] = newCommitId;
            }

            return {
                ...state,
                commits: [...state.commits, newCommit],
                branches: newBranches,
                output: [...state.output, `Merge made by the 'ort' strategy.`, ` ${sourceCommitId} merged into ${state.HEAD.ref || headCommitId}`]
            };
        }

        case ACTION_TYPES.STATUS: {
            if (!state.initialized) return { ...state, output: [...state.output, 'fatal: not a git repository'] };
            const branchName = state.HEAD.type === 'branch' ? state.HEAD.ref : null;
            const headId = branchName ? state.branches[branchName] : state.HEAD.id;

            let lines = [];
            if (branchName) lines.push(`On branch ${branchName}`);
            else lines.push(`HEAD detached at ${headId}`);

            if (state.staging.length === 0) {
                lines.push('nothing to commit, working tree clean');
            } else {
                lines.push('Changes to be committed:');
                state.staging.forEach(f => lines.push(`  (use "git restore --staged <file>..." to unstage)\n\tnew file:   ${f}`));
            }

            return { ...state, output: [...state.output, ...lines] };
        }

        default:
            return state;
    }
}
