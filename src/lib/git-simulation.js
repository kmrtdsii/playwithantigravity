import { v4 as uuidv4 } from 'uuid';

export const ACTION_TYPES = {
    INIT: 'INIT',
    COMMIT: 'COMMIT',
    CHECKOUT: 'CHECKOUT',
    BRANCH: 'BRANCH',
    LOG: 'LOG'
};

export const initialState = {
    initialized: false,
    commits: [], // Array of { id, message, parentId, timestamp }
    branches: {}, // Map branchName -> commitId
    HEAD: { type: 'branch', ref: null }, // { type: 'branch', ref: 'name' } or { type: 'commit', id: 'hash' }
    output: [] // Command output history
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
                output: [...state.output, 'Initialized empty Git repository']
            };

        case ACTION_TYPES.COMMIT: {
            if (!state.initialized) return { ...state, output: [...state.output, 'fatal: not a git repository'] };

            const { message } = action.payload;
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
                timestamp: new Date().toISOString()
            };

            // Update branch pointer if HEAD is on a branch
            let newBranches = { ...state.branches };
            if (state.HEAD.type === 'branch') {
                newBranches[state.HEAD.ref] = newCommitId;
            } else {
                // Detached HEAD mode, we just move HEAD (handled below usually, but actually detached head commits stay detached unless a branch is made)
                // For simplicity, we just update the HEAD commit ID if we are detached?
                // Actually, if detached, committing moves HEAD to new commit.
            }

            const newState = {
                ...state,
                commits: [...state.commits, newCommit],
                branches: newBranches,
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
            const { ref } = action.payload; // ref could be branch name or commit hash (simple version: branch only)

            if (state.branches[ref] !== undefined) {
                // Switch to branch
                return {
                    ...state,
                    HEAD: { type: 'branch', ref: ref },
                    output: [...state.output, `Switched to branch '${ref}'`]
                };
            } else {
                return { ...state, output: [...state.output, `error: pathspec '${ref}' did not match any file(s) known to git`] };
            }
        }

        default:
            return state;
    }
}
