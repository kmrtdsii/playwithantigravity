import React, { useMemo } from 'react';
import { useGit } from '../../context/GitAPIContext';
import type { Commit, GitState } from '../../types/gitTypes';

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
const computeLayout = (commits: Commit[], branches: Record<string, string>, references: Record<string, string>, remoteBranches: Record<string, string>, HEAD: GitState['HEAD']) => {
    if (commits.length === 0) return { nodes: [], edges: [], height: 0, badgesMap: {} };

    // 0. Ensure Sort order (Newest first)
    // Stable sort respecting Backend's topological order (index) if timestamps are equal.
    const sortedCommits = commits
        .map((c, i) => ({ c, i }))
        .sort((a, b) => {
            const timeA = new Date(a.c.timestamp).getTime();
            const timeB = new Date(b.c.timestamp).getTime();
            if (timeA === timeB) {
                // If times are equal, respect original backend order (Topological Child First)
                // Lower index = Newer
                return a.i - b.i;
            }
            return timeB - timeA;
        })
        .map(wrapper => wrapper.c);

    // --- REACHABILITY ANALYSIS ---
    const commitMap = new Map(commits.map(c => [c.id, c]));
    const reachable = new Set<string>();
    const queue: string[] = [];

    // Seed from Branches
    Object.values(branches).forEach(hash => {
        if (hash && commitMap.has(hash)) queue.push(hash);
    });

    // Seed from HEAD (if detached or pointing to a commit)
    if (HEAD.id && commitMap.has(HEAD.id)) {
        queue.push(HEAD.id);
    }

    // Traverse
    const visited = new Set<string>();
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
    // -----------------------------

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
            activePaths[index] = hash;
        }
    });

    // Helper ...
    const getLaneForHash = (h: string) => {
        for (let i = 0; i < activePaths.length; i++) {
            if (activePaths[i] === h) return i;
        }
        return -1;
    };

    const getFreeLane = () => {
        for (let i = 0; i < activePaths.length; i++) {
            if (activePaths[i] === null) return i;
        }
        return activePaths.length;
    };

    sortedCommits.forEach((c, i) => {
        let lane = getLaneForHash(c.id);

        if (lane === -1) {
            lane = getFreeLane();
        }

        activePaths[lane] = null;

        const color = LANE_COLORS[lane % LANE_COLORS.length];
        const x = GRAPH_LEFT_PADDING + lane * LANE_WIDTH + LANE_WIDTH / 2;
        const y = PADDING_TOP + i * ROW_HEIGHT + ROW_HEIGHT / 2;

        // Determine opacity
        // If no branches exist (empty repo init state?), everything is arguably dangling or everything is head?
        // But if we have commits, we usually have a branch or HEAD.
        // If reachable is empty (maybe HEAD is none?), default to visible? 
        // No, if we have commits but no refs, they ARE dangling.

        // However, if we just did 'git init', we have 0 commits, logic handles it at top.
        // If we have 1 commit and no branches/HEAD (weird), it is dangling.

        // VISUAL TWEAK: logic implies "if reachable has ANY entries, use it, else assume all visible (fallback)"?
        // No, strict reachability is better for "Show All" feature.
        const isReachable = reachable.size === 0 ? true : reachable.has(c.id);
        const opacity = isReachable ? 1 : 0.3;

        nodes.push({
            ...c,
            x, y, lane, color,
            // @ts-ignore - Adding dynamic property for rendering
            opacity
        });

        // Process Parents
        const parentIds = [];
        if (c.parentId) parentIds.push(c.parentId);
        if (c.secondParentId) parentIds.push(c.secondParentId);

        parentIds.forEach((pid, pIdx) => {
            let parentLane = getLaneForHash(pid);
            if (parentLane !== -1) {
                // already reserved
            } else {
                if (pIdx === 0) {
                    activePaths[lane] = pid;
                } else {
                    const newLane = getFreeLane();
                    activePaths[newLane] = pid;
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
            if (!parentNode) return;

            let path = '';
            if (node.lane === parentNode.lane) {
                path = `M ${node.x} ${node.y} L ${parentNode.x} ${parentNode.y}`;
            } else {
                path = createBezierPath(node.x, node.y, parentNode.x, parentNode.y);
            }

            edges.push({
                id: `${node.id}-${pid}`,
                color: node.color,
                path,
                // @ts-ignore
                opacity: (node as any).opacity // Inherit opacity
            });
        });
    });

    // 2. Badges & Labels
    const badgesMap: Record<string, Badge[]> = {};
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

    // References (ORIG_HEAD)
    if (references) {
        Object.entries(references).forEach(([name, commitId]) => {
            if (!commitId) return;
            if (!badgesMap[commitId]) badgesMap[commitId] = [];
            badgesMap[commitId].push({
                text: name,
                type: 'tag', // Re-use tag style or add new type
                isActive: false // Usually not active branch
            });
        });
    }

    // Remote Branches
    if (remoteBranches) {
        Object.entries(remoteBranches).forEach(([name, commitId]) => {
            if (!commitId) return;
            if (!badgesMap[commitId]) badgesMap[commitId] = [];
            badgesMap[commitId].push({
                text: name,
                type: 'tag',
                isActive: false
            });
        });
    }

    // Explicit HEAD Badge (Always show if we have a HEAD ID)
    let headId = HEAD.id;
    if (HEAD.type === 'branch' && HEAD.ref && branches[HEAD.ref]) {
        headId = branches[HEAD.ref];
    }

    if (headId) {
        if (!badgesMap[headId]) badgesMap[headId] = [];
        // Check if HEAD badge already exists? No, we just add it.
        // But we want it to be distinct.
        badgesMap[headId].push({ text: 'HEAD', type: 'head' });
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

interface GitGraphVizProps {
    onSelect?: (commit: Commit) => void;
    selectedCommitId?: string;
}

const GitGraphViz: React.FC<GitGraphVizProps> = ({ onSelect, selectedCommitId }) => {
    const { state } = useGit();
    const { commits, branches, references, remoteBranches, HEAD } = state;

    const { nodes, edges, height, badgesMap } = useMemo(() =>
        computeLayout(commits, branches, references || {}, remoteBranches || {}, HEAD),
        [commits, branches, references, remoteBranches, HEAD]
    );

    // Resolve HEAD commit ID for Halo
    let headCommitId: string | undefined;
    if (HEAD.type === 'commit') {
        headCommitId = HEAD.id;
    } else if (HEAD.type === 'branch' && HEAD.ref) {
        headCommitId = branches[HEAD.ref];
    }

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
                            // @ts-ignore
                            opacity={edge.opacity}
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
                            // @ts-ignore
                            opacity={node.opacity}
                        />
                    ))}

                    {/* Render HEAD Halo (Current Tip) */}
                    {nodes.map(node => {
                        if (node.id === headCommitId) {
                            return (
                                <circle
                                    key={`halo-${node.id}`}
                                    cx={node.x}
                                    cy={node.y}
                                    r={CIRCLE_RADIUS + 4}
                                    fill="none"
                                    stroke="var(--accent-primary, #3b82f6)"
                                    strokeWidth="2"
                                    opacity={0.8}
                                />
                            );
                        }
                        return null;
                    })}
                </svg>

                {/* Render Text Content (Clickable Rows) */}
                {nodes.map(node => {
                    const textX = 140; // Fixed gutter for rails
                    const hasBadges = badgesMap[node.id] && badgesMap[node.id].length > 0;
                    const isSelected = node.id === selectedCommitId;

                    return (
                        <div
                            key={node.id}
                            onClick={() => onSelect && onSelect(node)}
                            style={{
                                position: 'absolute',
                                left: 0,
                                top: node.y - ROW_HEIGHT / 2,
                                width: '100%',
                                paddingLeft: textX,
                                boxSizing: 'border-box',
                                height: ROW_HEIGHT,
                                display: 'flex',
                                alignItems: 'center',
                                whiteSpace: 'nowrap',
                                gap: '8px',
                                cursor: 'pointer',
                                paddingRight: '16px',
                                userSelect: 'none',
                                // @ts-ignore
                                opacity: node.opacity,
                                backgroundColor: isSelected ? 'rgba(59, 130, 246, 0.1)' : 'transparent', // Row Highlight
                                borderLeft: isSelected ? '4px solid var(--accent-primary)' : '4px solid transparent', // Accent bar
                            }}
                            className="commit-row"
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
                                            {/* Removed bullet: {badge.isActive && '● '} */}
                                            {badge.text}
                                        </span>
                                    ))}
                                </div>
                            )}

                            {/* Message */}
                            <span
                                title={node.message} // Tooltip for full message
                                style={{
                                    color: 'var(--text-secondary)',
                                    flex: 1,
                                    overflow: 'hidden',
                                    textOverflow: 'ellipsis',
                                    minWidth: 0,
                                }}
                            >
                                {node.message}
                            </span>

                            {/* Timestamp */}
                            <span style={{
                                color: 'var(--text-tertiary)',
                                fontSize: '10px',
                                width: '140px', // Fixed width for timestamp
                                textAlign: 'right',
                                flexShrink: 0,
                                marginRight: '8px'
                            }}>
                                {new Date(node.timestamp).toLocaleString('ja-JP', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', second: '2-digit' })}
                            </span>

                            {/* ID (Highlighted if selected) */}
                            <span style={{
                                color: isSelected ? 'var(--accent-primary, #3b82f6)' : 'var(--text-tertiary)',
                                fontSize: '10px',
                                width: '60px', // Fixed width for ID
                                textAlign: 'right',
                                flexShrink: 0,
                                fontWeight: isSelected ? 'bold' : 'normal',
                                padding: '0',
                            }}>
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
