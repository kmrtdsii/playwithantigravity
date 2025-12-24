import type { Commit, GitState } from '../../types/gitTypes';
import type { VizNode, VizEdge, Badge, LayoutResult } from './graphTypes';
import {
    ROW_HEIGHT,
    LANE_WIDTH,
    PADDING_TOP,
    GRAPH_LEFT_PADDING,
    LANE_COLORS,
} from './graphConstants';

/**
 * Creates a bezier curve path string between two points.
 * Used for edges that cross between lanes.
 */
export const createBezierPath = (x1: number, y1: number, x2: number, y2: number): string => {
    const dy = y2 - y1;
    const cy1 = y1 + dy * 0.5;
    const cy2 = y2 - dy * 0.5;
    return `M ${x1} ${y1} C ${x1} ${cy1}, ${x2} ${cy2}, ${x2} ${y2}`;
};

/**
 * Computes the visual layout for the Git graph.
 * 
 * This function takes raw commit data and produces positioned nodes and edges
 * suitable for SVG rendering. It handles:
 * - Sorting commits by timestamp
 * - Assigning lanes (columns) to commits
 * - Computing reachability from branch tips
 * - Creating connecting edges
 * - Generating badges for branches, tags, and HEAD
 * 
 * @param commits - Array of real commits
 * @param potentialCommits - Array of simulated/ghost commits
 * @param branches - Map of branch name to commit ID
 * @param references - Map of reference name to commit ID
 * @param remoteBranches - Map of remote branch name to commit ID
 * @param tags - Map of tag name to commit ID
 * @param HEAD - Current HEAD state
 * @returns Layout result with nodes, edges, height, and badges
 */
export const computeLayout = (
    commits: Commit[],
    potentialCommits: Commit[],
    branches: Record<string, string>,
    references: Record<string, string>,
    remoteBranches: Record<string, string>,
    tags: Record<string, string>,
    HEAD: GitState['HEAD']
): LayoutResult => {
    const combinedCommits = [
        ...commits.map(c => ({ ...c, isGhost: false })),
        ...potentialCommits.map(c => ({ ...c, isGhost: true }))
    ];

    if (combinedCommits.length === 0) {
        return { nodes: [], edges: [], height: 0, badgesMap: {} };
    }

    // Sort by timestamp (newest first), with stable secondary sort
    const sortedCommits = combinedCommits
        .map((c, i) => ({ c, i }))
        .sort((a, b) => {
            const timeA = new Date(a.c.timestamp).getTime();
            const timeB = new Date(b.c.timestamp).getTime();
            if (timeA === timeB) {
                return a.i - b.i;
            }
            return timeB - timeA;
        })
        .map(wrapper => wrapper.c);

    // --- TRUNK IDENTIFICATION ---
    const commitMap = new Map(combinedCommits.map(c => [c.id, c]));

    // Identify trunk commits (main/master)
    const trunkCommits = new Set<string>();
    let trunkHeadId: string | undefined;

    // 1. Try Local Branches
    const localTrunkName = ['main', 'master', 'trunk'].find(name => branches[name]);
    if (localTrunkName) {
        trunkHeadId = branches[localTrunkName];
    }
    // 2. Try Remote Branches (if no local trunk found)
    else {
        const remoteTrunkName = ['origin/main', 'origin/master', 'origin/trunk', 'upstream/main', 'upstream/master'].find(name => remoteBranches[name]);
        if (remoteTrunkName) {
            trunkHeadId = remoteBranches[remoteTrunkName];
        }
    }

    if (trunkHeadId) {
        let currentId: string | undefined = trunkHeadId;
        while (currentId) {
            trunkCommits.add(currentId);
            const commit = commitMap.get(currentId);
            if (!commit) break;

            // Prefer first parent for trunk lineage
            currentId = commit.parentId || undefined;
        }
    }

    // --- REACHABILITY ANALYSIS ---
    const reachable = computeReachability(commitMap, branches, HEAD, potentialCommits, remoteBranches, tags);

    // --- LANE ASSIGNMENT ---
    const nodes: VizNode[] = [];
    const edges: VizEdge[] = [];
    // activePaths tracks the commit ID currently occupying the tip of each lane
    // Lane 0 is reserved for Trunk if it exists
    const activePaths: (string | null)[] = [];

    // Initialize lanes from branches
    // We want to assign activePaths based on branch tips, but "Trunk" logic overrides normal flow

    // Helper to find a free lane. 
    // If 'allowZero' is false, it skips lane 0 (reserved for trunk).
    const getFreeLane = (allowZero: boolean): number => {
        const start = allowZero ? 0 : 1;
        for (let i = start; i < activePaths.length; i++) {
            if (activePaths[i] === null) return i;
        }
        return activePaths.length || start; // If empty, start at 'start'
    };

    // Helper: Is a lane occupied by a specific hash?
    const getLaneForHash = (h: string): number => {
        for (let i = 0; i < activePaths.length; i++) {
            if (activePaths[i] === h) return i;
        }
        return -1;
    };

    const isTrunkExists = trunkCommits.size > 0;

    // Position each commit
    sortedCommits.forEach((c, i) => {
        const isTrunk = trunkCommits.has(c.id);

        // 1. Determine Lane
        let lane = getLaneForHash(c.id);

        if (lane === -1) {
            // New processing tip (e.g. branch head or detached head)
            if (isTrunk) {
                lane = 0; // Force trunk to 0
            } else {
                // Determine free lane, skipping 0 if we have a trunk
                lane = getFreeLane(!isTrunkExists);
            }
        } else {
            // Already expected by a child.
            // If this is trunk commit, ensure it IS lane 0. 
            if (isTrunk && lane !== 0) {
                // Should not happen if logic is correct, but safe fallback
                // We could force it, but that might overlap with existing lane 0 content?
                // For now, accept it. Use straight trunk logic.
            }
        }

        // Occupy this lane for this commit
        activePaths[lane] = null; // Clear current

        const color = LANE_COLORS[lane % LANE_COLORS.length];
        const x = GRAPH_LEFT_PADDING + lane * LANE_WIDTH + LANE_WIDTH / 2;
        const y = PADDING_TOP + i * ROW_HEIGHT + ROW_HEIGHT / 2;
        const isReachable = reachable.size === 0 ? true : reachable.has(c.id);
        const opacity = c.isGhost ? 0.6 : (isReachable ? 1 : 0.3);

        // Store Node
        nodes.push({
            ...c,
            x, y, lane, color,
            isGhost: c.isGhost,
            opacity
        });

        // 2. Setup Parents
        const parents = [];
        if (c.parentId) parents.push(c.parentId);
        if (c.secondParentId) parents.push(c.secondParentId);

        parents.forEach((pid, pIdx) => {
            const isParentTrunk = trunkCommits.has(pid);

            // Check if parent is already waiting in a lane? (Merge case)
            const existingLane = getLaneForHash(pid);

            if (existingLane !== -1) {
                // Parent already has a lane assigned
            } else {
                // Parent needs a lane.
                let targetLane: number;

                if (isParentTrunk) {
                    targetLane = 0; // Force trunk parent to 0
                } else {
                    // Parent is not trunk.
                    // If pIdx === 0 (First Parent), try to keep same lane to make straight lines for branches too.
                    if (pIdx === 0 && lane !== 0) {
                        // Try to reuse current lane if free
                        if (activePaths[lane] === null) {
                            targetLane = lane;
                        } else {
                            targetLane = getFreeLane(!isTrunkExists);
                        }
                    } else {
                        targetLane = getFreeLane(!isTrunkExists);
                    }
                }

                // If targetLane is occupied by someone else (not pid), collision.
                // But simplified logic: we just overwrite activePaths[targetLane] = pid
                // Which effectively reserves it.
                // If it was already occupied, that previous expectation is "stolen" or "overwritten"?
                // Wait. 'activePaths' maps Lane -> Expected Commit ID.
                // If activePaths[targetLane] is NOT null and NOT pid, then TWO children want to put DIFFERENT parents in the same lane.
                // This is a collision. We must pick a new lane for the current parent in that case.

                if (activePaths[targetLane] !== null && activePaths[targetLane] !== pid) {
                    // Conflict! Valid lane occupied by another commit.
                    // Find another free one.
                    // If it was supposed to be Trunk (Lane 0), we can't move it easily if we want straight line.
                    // But if Lane 0 is occupied by NON-trunk, we should have kicked non-trunk out?
                    // With this simple logic, we just find next free.
                    targetLane = getFreeLane(!isTrunkExists);
                }

                activePaths[targetLane] = pid;
            }
        });
    });

    // Create edges (No changes needed, uses coordinates)
    const nodeMap = new Map(nodes.map(n => [n.id, n]));
    nodes.forEach(node => {
        const parents = [];
        if (node.parentId) parents.push(node.parentId);
        if (node.secondParentId) parents.push(node.secondParentId);

        parents.forEach(pid => {
            const parentNode = nodeMap.get(pid);
            if (!parentNode) return;

            // Straight line if same lane, specialized bezier if column 0 involved?
            // Existing bezier logic should handle it fine.
            const path = node.lane === parentNode.lane
                ? `M ${node.x} ${node.y} L ${parentNode.x} ${parentNode.y}`
                : createBezierPath(node.x, node.y, parentNode.x, parentNode.y);

            edges.push({
                id: `${node.id}-${pid}`,
                color: node.color,
                path,
                isGhost: node.isGhost || parentNode.isGhost,
                opacity: node.opacity
            });
        });
    });

    // Build badges map
    const badgesMap = buildBadgesMap(branches, tags, references, remoteBranches, HEAD);

    return {
        nodes,
        edges,
        height: PADDING_TOP + combinedCommits.length * ROW_HEIGHT + PADDING_TOP,
        badgesMap
    };
};

/**
 * Computes which commits are reachable from branch tips and HEAD.
 */
function computeReachability(
    commitMap: Map<string, Commit & { isGhost: boolean }>,
    branches: Record<string, string>,
    HEAD: GitState['HEAD'],
    potentialCommits: Commit[],
    remoteBranches: Record<string, string>,
    tags: Record<string, string>
): Set<string> {
    const queue: string[] = [];

    // Seed from local branches
    Object.values(branches).forEach(hash => {
        if (hash && commitMap.has(hash)) queue.push(hash);
    });

    // Seed from remote branches
    Object.values(remoteBranches).forEach(hash => {
        if (hash && commitMap.has(hash)) queue.push(hash);
    });

    // Seed from tags
    Object.values(tags).forEach(hash => {
        if (hash && commitMap.has(hash)) queue.push(hash);
    });

    // Seed from HEAD
    if (HEAD.id && commitMap.has(HEAD.id)) {
        queue.push(HEAD.id);
    }

    // Potential commits are always reachable in simulation
    potentialCommits.forEach(c => queue.push(c.id));

    // BFS traversal
    const visited = new Set<string>();
    const reachable = new Set<string>();

    while (queue.length > 0) {
        const currentId = queue.shift()!;
        if (visited.has(currentId)) continue;
        visited.add(currentId);
        reachable.add(currentId);

        const commit = commitMap.get(currentId);
        if (commit) {
            if (commit.parentId) queue.push(commit.parentId);
            if (commit.secondParentId) queue.push(commit.secondParentId);
        }
    }

    return reachable;
}

/**
 * Builds the badges map for commits (branches, tags, HEAD).
 */
function buildBadgesMap(
    branches: Record<string, string>,
    tags: Record<string, string> | undefined,
    references: Record<string, string> | undefined,
    remoteBranches: Record<string, string> | undefined,
    HEAD: GitState['HEAD']
): Record<string, Badge[]> {
    const badgesMap: Record<string, Badge[]> = {};
    const activeBranchName = HEAD.type === 'branch' ? HEAD.ref : null;

    // Add branch badges
    Object.entries(branches).forEach(([name, commitId]) => {
        if (!commitId) return;
        if (!badgesMap[commitId]) badgesMap[commitId] = [];
        badgesMap[commitId].push({
            text: name,
            type: 'branch',
            isActive: name === activeBranchName
        });
    });

    // Add tag badges
    if (tags) {
        Object.entries(tags).forEach(([name, commitId]) => {
            if (!commitId) return;
            if (!badgesMap[commitId]) badgesMap[commitId] = [];
            badgesMap[commitId].push({ text: name, type: 'tag', isActive: false });
        });
    }

    // Add reference badges (except ORIG_HEAD)
    if (references) {
        Object.entries(references).forEach(([name, commitId]) => {
            if (!commitId || name === 'ORIG_HEAD') return;
            if (tags && tags[name]) return; // Skip duplicates
            if (!badgesMap[commitId]) badgesMap[commitId] = [];
            badgesMap[commitId].push({ text: name, type: 'tag', isActive: false });
        });
    }

    // Add remote branch badges
    if (remoteBranches) {
        Object.entries(remoteBranches).forEach(([name, commitId]) => {
            if (!commitId) return;
            if (!badgesMap[commitId]) badgesMap[commitId] = [];
            badgesMap[commitId].push({ text: name, type: 'remote-branch', isActive: false });
        });
    }

    // Add HEAD badge
    const headId = HEAD.type === 'branch' && HEAD.ref && branches[HEAD.ref]
        ? branches[HEAD.ref]
        : HEAD.id;

    if (headId) {
        if (!badgesMap[headId]) badgesMap[headId] = [];
        badgesMap[headId].push({ text: 'HEAD', type: 'head' });
    }

    return badgesMap;
}
