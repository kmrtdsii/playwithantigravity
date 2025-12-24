import type { Commit, GitState } from '../types/gitTypes';

/**
 * Filters commits to only include those reachable from HEAD.
 * This is used when SHOW ALL is OFF - only HEAD-reachable commits are displayed.
 * Uses BFS traversal starting from HEAD only.
 *
 * @param commits - All commits from the state
 * @param state - The GitState containing HEAD
 * @returns Filtered array of commits that are reachable from HEAD
 */
export function filterReachableCommits(
    commits: Commit[],
    state: Pick<GitState, 'HEAD' | 'branches' | 'tags' | 'remoteBranches'>
): Commit[] {
    if (commits.length === 0) return [];

    const reachable = new Set<string>();
    const queue: string[] = [];

    // SHOW ALL = OFF: Only use HEAD as the seed
    // This means only commits reachable by following parent links from HEAD are shown
    if (state.HEAD?.id) {
        queue.push(state.HEAD.id);
    } else if (state.HEAD?.ref && state.branches) {
        // HEAD points to a branch, resolve it
        const branchCommit = state.branches[state.HEAD.ref];
        if (branchCommit) {
            queue.push(branchCommit);
        }
    }

    // If no HEAD is set, return empty (no commits to show)
    if (queue.length === 0) return [];

    // Build a lookup map for O(1) access
    const commitMap = new Map<string, Commit>();
    commits.forEach((c) => commitMap.set(c.id, c));

    // BFS Traverse from HEAD only
    while (queue.length > 0) {
        const currentId = queue.shift()!;
        if (reachable.has(currentId)) continue;
        reachable.add(currentId);

        const commit = commitMap.get(currentId);
        if (commit) {
            if (commit.parentId && !reachable.has(commit.parentId)) {
                queue.push(commit.parentId);
            }
            if (commit.secondParentId && !reachable.has(commit.secondParentId)) {
                queue.push(commit.secondParentId);
            }
        }
    }

    return commits.filter((c) => reachable.has(c.id));
}

