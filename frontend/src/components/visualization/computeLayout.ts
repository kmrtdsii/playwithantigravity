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

    // --- REACHABILITY ANALYSIS ---
    const commitMap = new Map(combinedCommits.map(c => [c.id, c]));
    const reachable = computeReachability(commitMap, branches, HEAD, potentialCommits);

    // --- LANE ASSIGNMENT ---
    const nodes: VizNode[] = [];
    const edges: VizEdge[] = [];
    const activePaths: (string | null)[] = [];

    // Initialize lanes from branches (main first)
    const branchNames = Object.keys(branches).sort((a, b) => {
        if (a === 'main') return -1;
        if (b === 'main') return 1;
        return a.localeCompare(b);
    });

    branchNames.forEach((name, index) => {
        const hash = branches[name];
        if (hash) {
            activePaths[index] = hash;
        }
    });

    const getLaneForHash = (h: string): number => {
        for (let i = 0; i < activePaths.length; i++) {
            if (activePaths[i] === h) return i;
        }
        return -1;
    };

    const getFreeLane = (): number => {
        for (let i = 0; i < activePaths.length; i++) {
            if (activePaths[i] === null) return i;
        }
        return activePaths.length;
    };

    // Position each commit
    sortedCommits.forEach((c, i) => {
        let lane = getLaneForHash(c.id);
        if (lane === -1) {
            lane = getFreeLane();
        }
        activePaths[lane] = null;

        const color = LANE_COLORS[lane % LANE_COLORS.length];
        const x = GRAPH_LEFT_PADDING + lane * LANE_WIDTH + LANE_WIDTH / 2;
        const y = PADDING_TOP + i * ROW_HEIGHT + ROW_HEIGHT / 2;
        const isReachable = reachable.size === 0 ? true : reachable.has(c.id);
        const opacity = c.isGhost ? 0.6 : (isReachable ? 1 : 0.3);

        nodes.push({
            ...c,
            x, y, lane, color,
            isGhost: c.isGhost,
            opacity
        });

        // Track parent lanes
        const parentIds = [];
        if (c.parentId) parentIds.push(c.parentId);
        if (c.secondParentId) parentIds.push(c.secondParentId);

        parentIds.forEach((pid, pIdx) => {
            const parentLane = getLaneForHash(pid);
            if (parentLane === -1) {
                if (pIdx === 0) {
                    activePaths[lane] = pid;
                } else {
                    const newLane = getFreeLane();
                    activePaths[newLane] = pid;
                }
            }
        });
    });

    // Create edges
    const nodeMap = new Map(nodes.map(n => [n.id, n]));
    nodes.forEach(node => {
        const parents = [];
        if (node.parentId) parents.push(node.parentId);
        if (node.secondParentId) parents.push(node.secondParentId);

        parents.forEach(pid => {
            const parentNode = nodeMap.get(pid);
            if (!parentNode) return;

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
    potentialCommits: Commit[]
): Set<string> {
    const queue: string[] = [];

    // Seed from branches
    Object.values(branches).forEach(hash => {
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
            badgesMap[commitId].push({ text: name, type: 'tag', isActive: false });
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
