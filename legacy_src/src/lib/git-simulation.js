import { v4 as uuidv4 } from 'uuid';
export const ACTION_TYPES = {
    INIT: 'INIT',
    ADD: 'ADD',
    COMMIT: 'COMMIT',
    CHECKOUT: 'CHECKOUT',
    BRANCH: 'BRANCH',
    MERGE: 'MERGE',
    STATUS: 'STATUS',
    LOG: 'LOG',
    SWITCH: 'SWITCH',
    RESTORE: 'RESTORE',
    TOUCH: 'TOUCH' // Simulate editing a file
};

export const initialState = {
    initialized: false,
    commits: [], // Array of { id, message, parentId, timestamp }
    branches: {}, // Map branchName -> commitId
    HEAD: { type: 'branch', ref: null }, // { type: 'branch', ref: 'name' } or { type: 'commit', id: 'hash' }
    staging: [], // Array of file paths
    modified: [], // Array of file paths (Working Directory)
    files: [], // Array of all known files in the project
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
                files: ['file.txt', 'style.css', 'app.js', 'README.md'], // Basic project files
                modified: ['file.txt', 'style.css', 'app.js', 'README.md'], // Initially all modified/untracked
                output: [...state.output, 'Initialized empty Git repository']
            };

        case ACTION_TYPES.TOUCH: {
            if (!state.initialized) return { ...state, output: [...state.output, 'fatal: not a git repository'] };
            const { file } = action.payload;

            // If known file, mark as modified
            if (state.files.includes(file)) {
                if (state.staging.includes(file)) {
                    // If staged, it becomes modified AGAIN? (Staged + Modified)
                    // For simplicity, let's just say it's modified.
                    // A file can be both, but our array logic is simple.
                    // Let's ensure it's in modified.
                }
                return {
                    ...state,
                    modified: [...new Set([...state.modified, file])],
                    output: [...state.output, ` touched ${file}`]
                };
            }
            return { ...state, output: [...state.output, `touch: ${file}: No such file`] };
        }

        case ACTION_TYPES.ADD: {
            if (!state.initialized) return { ...state, output: [...state.output, 'fatal: not a git repository'] };
            const { files } = action.payload; // array of filenames

            // Move from modified to staging
            // If specific files, filter. If logic is 'add .', assume simulated behavior

            // For sim: if files includes a file in 'modified', move it.
            // If not in modified, maybe user created it?

            const filesToAdd = files.includes('.') ? state.modified : files;

            const newStaging = [...new Set([...state.staging, ...filesToAdd])];
            const newModified = state.modified.filter(f => !filesToAdd.includes(f));

            return {
                ...state,
                staging: newStaging,
                modified: newModified,
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

            // Determine Tracked Files for Snapshot
            // The commit includes files that were staged (now committed) + files that were already clean.
            // In our Sim:
            // state.files = All files
            // state.modified = Untracked/Modified files
            // state.staging = Files about to be committed (moved out of modified)

            // So Tracked Files in this commit = (state.files - state.modified)
            // Note: 'state.staging' files are already REMOVED from 'state.modified' by ADD action.
            // So 'state.files - state.modified' includes Staged files + Clean files.
            // This is exactly what we want.
            const trackedFiles = state.files.filter(f => !state.modified.includes(f));

            const newCommit = {
                id: newCommitId,
                message,
                parentId,
                secondParentId: null, // For merges
                branch: state.HEAD.type === 'branch' ? state.HEAD.ref : 'detached',
                timestamp: new Date().toISOString(),
                fileSnapshot: trackedFiles // Snapshot ONLY tracked files
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
                // modified remains as is (Untracked files persist)
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
                branch: state.HEAD.type === 'branch' ? state.HEAD.ref : 'detached',
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

        case ACTION_TYPES.SWITCH: {
            if (!state.initialized) return { ...state, output: [...state.output, 'fatal: not a git repository'] };
            const { ref, create } = action.payload;

            if (create) {
                // git switch -c <name> (same as checkout -b)
                if (state.branches[ref]) {
                    return { ...state, output: [...state.output, `fatal: A branch named '${ref}' already exists.`] };
                }
                const currentCommitId = state.HEAD.type === 'branch'
                    ? state.branches[state.HEAD.ref]
                    : state.HEAD.id;

                // When creating a branch, we stay on current files (snapshot doesn't change yet)
                return {
                    ...state,
                    branches: { ...state.branches, [ref]: currentCommitId },
                    HEAD: { type: 'branch', ref: ref },
                    output: [...state.output, `Switched to a new branch '${ref}'`]
                };
            } else {
                // git switch <name>
                if (state.branches[ref] !== undefined) {
                    const targetCommitId = state.branches[ref];

                    // Restore Files from Snapshot
                    let restoredTrackedFiles = [];

                    // Find commit
                    const commit = state.commits.find(c => c.id === targetCommitId);
                    if (commit && commit.fileSnapshot) {
                        restoredTrackedFiles = commit.fileSnapshot;
                    } else if (!commit && !targetCommitId) {
                        // Attempting to switch to a branch with no commits? (e.g. initial 'main')
                        // If we are simulating "initial state", maybe we restore initial files?
                        // But usually 'switch' implies switching to a commit.
                        // For safety, let's default to [] if purely empty.
                        // If we are keeping some "initial" files, we might lose them here if we default to [].
                        // But for now, let's assume empty tracked.
                    }

                    // Current Untracked files = state.modified
                    // (Files in WD that are NOT tracked in current HEAD are in modified)
                    const currentUntracked = state.modified;

                    // New Files List = Restored Tracked + Current Untracked
                    // Use Set to dedup (though they should be disjoint ideally)
                    // If a file is in both, it means it's tracked in Target AND was untracked in Source?
                    // -> Git would overwrite (or complain).
                    // -> We will allow it to become "Tracked Clean" (from snapshot).
                    // So we prioritize Snapshot?
                    // Wait, if it's in Snapshot, it will be Clean.
                    // If it was in Untracked, we should probably remove it from Untracked (state.modified) because now it's tracked!

                    // New Modified = Current Untracked MINUS Restored Tracked
                    // (i.e. if file exists in Target, it becomes clean)
                    const newModified = currentUntracked.filter(f => !restoredTrackedFiles.includes(f));

                    const newFiles = [...new Set([...restoredTrackedFiles, ...newModified])];

                    return {
                        ...state,
                        HEAD: { type: 'branch', ref: ref },
                        files: newFiles,
                        staging: [],
                        modified: newModified,
                        output: [...state.output, `Switched to branch '${ref}'`]
                    };
                }
                return { ...state, output: [...state.output, `fatal: invalid reference: ${ref}`] };
            }
        }

        case ACTION_TYPES.RESTORE: {
            if (!state.initialized) return { ...state, output: [...state.output, 'fatal: not a git repository'] };
            const { files, staged } = action.payload;

            if (staged) {
                // git restore --staged <file>...
                // Move from staging back to modified
                const filesToRestore = files.includes('.') ? state.staging : files;
                const newStaging = state.staging.filter(f => !filesToRestore.includes(f));

                // Add back to modified if it was there (or if we assume it becomes modified)
                const newModified = [...new Set([...state.modified, ...filesToRestore])];

                return {
                    ...state,
                    staging: newStaging,
                    modified: newModified,
                    output: state.output
                };
            }

            // git restore <file>... (working tree)
            // In our sim, this "discards changes". If we had a list of modified files distinct from staged, we'd revert them.
            // Since we assume simple sim where "changes" come from "add .", let's just say we can't fully simulate file content restore yet
            // UNLESS we interpret "restore" as removing from our implicit "modified list" which we don't assume exists until 'add'.
            // Actually, if we have files in 'staging', and we 'restore' them (without --staged), it does nothing unless they are modified relative to index.
            // Let's just output a message that we restored them.
            return {
                ...state,
                output: [...state.output] // Silent success for simulation
            };
        }

        case ACTION_TYPES.STATUS: {
            if (!state.initialized) return { ...state, output: [...state.output, 'fatal: not a git repository'] };
            const branchName = state.HEAD.type === 'branch' ? state.HEAD.ref : null;
            const headId = branchName ? state.branches[branchName] : state.HEAD.id;

            let lines = [];
            if (branchName) lines.push(`On branch ${branchName}`);
            else lines.push(`HEAD detached at ${headId}`);

            if (state.staging.length === 0 && state.modified.length === 0) {
                lines.push('nothing to commit, working tree clean');
            } else {
                if (state.staging.length > 0) {
                    lines.push('Changes to be committed:');
                    state.staging.forEach(f => lines.push(`  (use "git restore --staged <file>..." to unstage)\n\tnew file:   ${f}`));
                }
                if (state.modified.length > 0) {
                    lines.push('Changes not staged for commit:');
                    state.modified.forEach(f => lines.push(`  (use "git add <file>..." to update what will be committed)\n\tmodified:   ${f}`));
                }
            }

            return { ...state, output: [...state.output, ...lines] };
        }

        default:
            return state;
    }
}
