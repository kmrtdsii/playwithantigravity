import type { Commit, GitState } from '../types/gitTypes';

/**
 * Filters commits to only include those reachable from known references (HEAD, branches, tags).
 * Uses BFS traversal starting from reference points.
 *
 * @param commits - All commits from the state
 * @param state - The GitState containing HEAD, branches, and tags
 * @returns Filtered array of commits that are reachable
 */
export function filterReachableCommits(
    commits: Commit[],
    state: Pick<GitState, 'HEAD' | 'branches' | 'tags'>
): Commit[] {
    if (commits.length === 0) return [];

    const reachable = new Set<string>();
    const queue: string[] = [];

    // 1. Seeds: HEAD, Branches, Tags
    if (state.HEAD?.id) {
        queue.push(state.HEAD.id);
    }

    if (state.branches) {
        Object.values(state.branches).forEach((commitId) => {
            queue.push(commitId);
        });
    }

    if (state.tags) {
        Object.values(state.tags).forEach((commitId) => {
            queue.push(commitId);
        });
    }

    // Build a lookup map for O(1) access
    const commitMap = new Map<string, Commit>();
    commits.forEach((c) => commitMap.set(c.id, c));

    // 2. BFS Traverse
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
