import React, { useMemo } from 'react';
import { useGit } from '../../context/GitAPIContext';
import type { Commit, GitState } from '../../context/GitAPIContext';

// --- Constants & Config ---
const ROW_HEIGHT = 36; // Taller rows
const LANE_WIDTH = 18; // Wider lanes
const CIRCLE_RADIUS = 5; // Slightly larger nodes
const PADDING_TOP = 24;
const GRAPH_LEFT_PADDING = 24;

const LANE_COLORS = [
    '#58a6ff', // Blue
    '#d2a8ff', // Purple
    '#3fb950', // Green
    '#ffa657', // Orange
    '#ff7b72', // Red
    '#79c0ff', // Light Blue
    '#f2cc60', // Yellow
    '#56d364', // Light Green
];

interface VizNode extends Commit {
    x: number;
    y: number;
    lane: number;
    color: string;
}

interface VizEdge {
    id: string;
    path: string;
    color: string;
}

interface Badge {
    text: string;
    type: 'branch' | 'head' | 'tag';
    isActive?: boolean;
}

// --- Layout Engine ---

// Helper to compute layout
const computeLayout = (commits: Commit[], branches: Record<string, string>, HEAD: GitState['HEAD']) => {
    if (commits.length === 0) return { nodes: [], edges: [], height: 0, badgesMap: {} };

    // 0. Ensure Sort order (Newest first)
    // We create a copy to avoid mutating props, though typically props are immutable.
    const sortedCommits = [...commits].sort((a, b) => {
        return new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime();
    });

    const nodes: VizNode[] = [];
    const edges: VizEdge[] = [];

    // "activePaths" = list of slots (lanes). 
    // Each slot contains the parent_hash it is looking for.
    // If a slot is null, it is free.
    const activePaths: (string | null)[] = [];

    // --- SEED LANES FOR BRANCHES ---
    // We want each branch tip to start on its own lane to prevent visual collision.
    // Lane 0 is reserved for 'main' if it exists.
    // Other branches get subsequent lanes based on name sort.
    const branchNames = Object.keys(branches).sort((a, b) => {
        if (a === 'main') return -1;
        if (b === 'main') return 1;
        return a.localeCompare(b);
    });

    branchNames.forEach((name, index) => {
        const hash = branches[name];
        if (hash) {
            // If lane is already taken by a previous branch (pointing to same commit),
            // we could skip it to merge trails, OR we can force it to reserve a new lane (index)
            // to show distinct tips initially.
            // Let's force distinct start lanes by index.
            // But we must respect the array index = lane logic.
            activePaths[index] = hash;
        }
    });

    // Helper to find a lane for a target commit hash
    const getLaneForHash = (h: string) => {
        for (let i = 0; i < activePaths.length; i++) {
            if (activePaths[i] === h) return i;
        }
        return -1;
    };

    // Helper to get next free lane
    const getFreeLane = () => {
        for (let i = 0; i < activePaths.length; i++) {
            if (activePaths[i] === null) return i;
        }
        // If we are appending a new lane, make sure we don't accidentally take Lane 0 
        // if it was reserved (activePaths[0] != null) but just not matched yet.
        // Actually, if activePaths[0] is set (waiting for main), getFreeLane won't return 0. 
        // It will return index > length.
        return activePaths.length;
    };

    sortedCommits.forEach((c, i) => {
        let lane = getLaneForHash(c.id);

        // If this commit was not expected by any lane (start of a new branch / ref)
        if (lane === -1) {
            lane = getFreeLane();
        }

        // Use this lane.
        // If we "found" the commit activePaths[lane] was looking for, we consume it.
        // We will set the NEW expectation (parent) below.
        activePaths[lane] = null;

        // Assign color based on lane index
        const color = LANE_COLORS[lane % LANE_COLORS.length];

        const x = GRAPH_LEFT_PADDING + lane * LANE_WIDTH + LANE_WIDTH / 2;
        const y = PADDING_TOP + i * ROW_HEIGHT + ROW_HEIGHT / 2;

        nodes.push({
            ...c,
            x, y, lane, color
        });

        // Process Parents
        const parentIds = [];
        if (c.parentId) parentIds.push(c.parentId);
        if (c.secondParentId) parentIds.push(c.secondParentId);

        parentIds.forEach((pid, pIdx) => {
            // Check if ANY lane is already looking for this parent
            let parentLane = getLaneForHash(pid);

            if (parentLane !== -1) {
                // Parent already has a reserved lane (from a sibling).
                // We will just draw a merge line to it later.
            } else {
                // Parent not yet accounted for.
                if (pIdx === 0) {
                    // Primary parent: extend CURRENT lane
                    activePaths[lane] = pid;
                    parentLane = lane;
                } else {
                    // Secondary parent: must fork to a NEW lane
                    const newLane = getFreeLane();
                    activePaths[newLane] = pid;
                    parentLane = newLane;
                }
            }
        });
    });

    // Pass 2: Generate Edges based on calculated positions
    const nodeMap = new Map(nodes.map(n => [n.id, n]));

    nodes.forEach(node => {
        const parents = [];
        if (node.parentId) parents.push(node.parentId);
        if (node.secondParentId) parents.push(node.secondParentId);

        parents.forEach(pid => {
            const parentNode = nodeMap.get(pid);
            if (!parentNode) {
                // Determine if parent is off-screen (not in list). 
                // For now, ignore.
                return;
            }

            // CURVE LOGIC
            // Standard: Bezier S-curve
            // If lane is same: Straight line

            let path = '';
            if (node.lane === parentNode.lane) {
                path = `M ${node.x} ${node.y} L ${parentNode.x} ${parentNode.y}`;
            } else {
                path = createBezierPath(node.x, node.y, parentNode.x, parentNode.y);
            }

            edges.push({
                id: `${node.id}-${pid}`,
                color: node.color, // Use child color? Or parent? Usually child.
                path
            });
        });
    });

    // 2. Badges & Labels
    const badgesMap: Record<string, Badge[]> = {};

    // Active Branch logic
    let activeBranchName = null;
    if (HEAD.type === 'branch' && HEAD.ref) {
        activeBranchName = HEAD.ref;
    }

    Object.entries(branches).forEach(([name, commitId]) => {
        if (!commitId) return;
        if (!badgesMap[commitId]) badgesMap[commitId] = [];
        const isActive = name === activeBranchName;

        badgesMap[commitId].push({
            text: name,
            type: 'branch',
            isActive
        });
    });

    // HEAD (only if detached)
    if (HEAD.type === 'commit' && HEAD.id) {
        if (!badgesMap[HEAD.id]) badgesMap[HEAD.id] = [];
        badgesMap[HEAD.id].push({ text: 'HEAD', type: 'head' });
    }

    return {
        nodes,
        edges,
        height: PADDING_TOP + commits.length * ROW_HEIGHT + PADDING_TOP,
        badgesMap
    };
};

// SVG Path Helper
const createBezierPath = (x1: number, y1: number, x2: number, y2: number) => {
    // Vertical distance
    const dy = y2 - y1;
    // Control points for smooth S-curve
    const cy1 = y1 + dy * 0.5;
    const cy2 = y2 - dy * 0.5;

    return `M ${x1} ${y1} C ${x1} ${cy1}, ${x2} ${cy2}, ${x2} ${y2}`;
};


// --- Component ---

const GitGraphViz: React.FC = () => {
    const { state } = useGit();
    const { commits, branches, HEAD } = state;

    const { nodes, edges, height, badgesMap } = useMemo(() =>
        computeLayout(commits, branches, HEAD),
        [commits, branches, HEAD]
    );

    if (!state.initialized) {
        return (
            <div className="flex h-full items-center justify-center text-gray-500 font-mono text-sm">
                Type <code className="mx-1 text-gray-400">git init</code> to start.
            </div>
        );
    }

    return (
        <div style={{
            height: '100%',
            overflow: 'auto',
            background: 'var(--bg-primary)',
            color: 'var(--text-primary)',
            fontFamily: 'Menlo, Monaco, Consolas, monospace',
            fontSize: '12px'
        }}>
            <div style={{ position: 'relative', height: Math.max(height, 500) }}>
                <svg width="100%" height={height} style={{ position: 'absolute', left: 0, top: 0, pointerEvents: 'none' }}>
                    {/* Render Edges first (behind nodes) */}
                    {edges.map(edge => (
                        <path
                            key={edge.id}
                            d={edge.path}
                            stroke={edge.color}
                            strokeWidth="2"
                            fill="none"
                            strokeLinecap="round"
                        />
                    ))}

                    {/* Render Nodes */}
                    {nodes.map(node => (
                        <circle
                            key={node.id}
                            cx={node.x}
                            cy={node.y}
                            r={CIRCLE_RADIUS}
                            fill={node.color}
                            stroke="var(--bg-primary)"
                            strokeWidth="1"
                        />
                    ))}
                </svg>

                {/* Render Text Content (Positioned absolutely over the graph) */}
                {nodes.map(node => {
                    // Text should start after the graph area.
                    // Calculate max lane x to avoid overlapping text? 
                    // Or just strict columns. 
                    // Let's put text at a fixed offset + max lane width?
                    // Simpler: x = (Graph Area) + 10px.
                    // Graph Area Width ~ 150px or dynamic.

                    const textX = 140; // Fixed gutter for rails
                    const hasBadges = badgesMap[node.id] && badgesMap[node.id].length > 0;

                    return (
                        <div
                            key={node.id}
                            style={{
                                position: 'absolute',
                                left: textX,
                                top: node.y - ROW_HEIGHT / 2,
                                height: ROW_HEIGHT,
                                display: 'flex',
                                alignItems: 'center',
                                whiteSpace: 'nowrap',
                                gap: '8px'
                            }}
                        >
                            {/* Badges */}
                            {hasBadges && (
                                <div style={{ display: 'flex', gap: '4px' }}>
                                    {badgesMap[node.id].map((badge, i) => (
                                        <span
                                            key={i}
                                            style={{
                                                fontSize: '10px',
                                                padding: '1px 6px',
                                                borderRadius: '10px',
                                                fontWeight: badge.isActive ? 'bold' : 'normal',
                                                backgroundColor: 'transparent',
                                                border: `1px solid ${node.color}`,
                                                color: node.color,
                                                opacity: 0.9
                                            }}
                                        >
                                            {badge.isActive && '● '}
                                            {badge.text}
                                        </span>
                                    ))}
                                </div>
                            )}

                            {/* Message */}
                            <span style={{ color: 'var(--text-secondary)' }}>
                                {node.message}
                            </span>

                            {/* ID (faded) */}
                            <span style={{ color: 'var(--text-tertiary)', fontSize: '10px', marginLeft: '4px' }}>
                                {node.id.substring(0, 7)}
                            </span>
                        </div>
                    );
                })}
            </div>
        </div>
    );
};

export default GitGraphViz;
