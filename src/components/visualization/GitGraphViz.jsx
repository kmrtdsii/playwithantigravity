import React, { useMemo } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { useGit } from '../../lib/GitContext';

// Basic constants
const NODE_RADIUS = 16;
const X_SPACING = 80;
const Y_SPACING = 60;
const START_X = 50;
const START_Y = 100;

// Helper to compute layout
const computeLayout = (commits, branches, HEAD) => {
    if (commits.length === 0) return { nodes: [], edges: [] };

    // 1. Build adjacency list (children)
    const childrenMap = {};
    const roots = [];

    commits.forEach(c => {
        if (!c.parentId) {
            roots.push(c.id);
        } else {
            if (!childrenMap[c.parentId]) childrenMap[c.parentId] = [];
            childrenMap[c.parentId].push(c.id);
        }

        // Handle merge parents
        if (c.secondParentId) {
            if (!childrenMap[c.secondParentId]) childrenMap[c.secondParentId] = [];
            childrenMap[c.secondParentId].push(c.id);
        }
    });

    // 2. Assign lanes via DFS
    const positions = {}; // commitId -> { x, y, lane }
    const lanes = {};     // laneIndex -> nextAvailableX (not used strictly, but good for tracking)

    // Create a map to look up commit by ID easily
    const commitMap = commits.reduce((acc, c) => ({ ...acc, [c.id]: c }), {});

    const traverse = (commitId, lane, xIndex) => {
        positions[commitId] = {
            x: START_X + xIndex * X_SPACING,
            y: START_Y + lane * Y_SPACING,
            lane
        };

        const children = childrenMap[commitId] || [];

        // Sort children? Chronological usually.
        // We want the "main" child to stay on the same lane.
        // Heuristic: First child keeps lane, others get new lanes.

        children.forEach((childId, idx) => {
            let childLane = lane;
            if (idx > 0) {
                // Find next available lane? For now just +1 per fork.
                // A better algo would find free lanes.
                // Simple hack: lane + idx
                // But what if lane+1 is occupied?
                // Let's just use a global counter or check collision?
                // Simple Tree Layout: childLane = lane + idx
                // This might overlap if multiple branches merge or split close by.
                // For MVP sandbox, let's assume simple branching.
                childLane = lane + idx;

                // If current position already taken at this x-index?
                // Actually X is strictly increasing? No, branches can exist in parallel time.
                // In my current simulation, commits are chronological list.
                // Lets use index in `commits` array as X for strict time ordering?
                // Yes, that avoids collision in X.
                const childCommit = commitMap[childId];
                const globalIndex = commits.findIndex(c => c.id === childId);

                // Re-calculate X based on global index to ensure time flow
                // positions[childId] assigned below.
            }

            const childGlobalIndex = commits.findIndex(c => c.id === childId);
            // We recurse, but pass the computed lane.
            // We use globalIndex for X to ensure no overlaps and chronological order.
            traverse(childId, childLane, childGlobalIndex);
        });
    };

    roots.forEach(rootId => {
        traverse(rootId, 0, commits.findIndex(c => c.id === rootId));
    });

    // 3. Generate Nodes and Edges
    const nodes = commits.map(c => ({
        ...c,
        ...positions[c.id]
    }));

    const edges = [];
    commits.forEach(c => {
        if (c.parentId) {
            const source = positions[c.parentId];
            const target = positions[c.id];
            if (source && target) {
                edges.push({
                    id: `${c.parentId}-${c.id}`,
                    x1: source.x, y1: source.y,
                    x2: target.x, y2: target.y,
                    isMerge: false
                });
            }
        }
        if (c.secondParentId) {
            const source = positions[c.secondParentId];
            const target = positions[c.id];
            if (source && target) {
                edges.push({
                    id: `${c.secondParentId}-${c.id}`,
                    x1: source.x, y1: source.y,
                    x2: target.x, y2: target.y,
                    isMerge: true
                });
            }
        }
    });

    // 4. Determine Active Status (HEAD)
    // HEAD can be a commit or a branch ref.
    let headCommitId = null;
    if (HEAD.type === 'commit') headCommitId = HEAD.id;
    else if (HEAD.type === 'branch') headCommitId = branches[HEAD.ref];

    return { nodes, edges, headCommitId };
};

const GitGraphViz = () => {
    const { state } = useGit();
    const { commits, branches, HEAD } = state;

    const { nodes, edges, headCommitId } = useMemo(() =>
        computeLayout(commits, branches, HEAD),
        [commits, branches, HEAD]
    );

    if (!state.initialized) {
        return (
            <div style={{
                height: '100%',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                color: 'var(--text-secondary)',
                fontSize: '0.9rem'
            }}>
                Type <code>git init</code> to start visualizing.
            </div>
        );
    }

    return (
        <div style={{ width: '100%', height: '100%', overflow: 'auto' }}>
            <svg width="2000" height="1000">
                <defs>
                    <marker
                        id="arrowhead"
                        markerWidth="10"
                        markerHeight="7"
                        refX="24"
                        refY="3.5"
                        orient="auto"
                    >
                        <polygon points="0 0, 10 3.5, 0 7" fill="var(--border-active)" />
                    </marker>
                </defs>

                {/* Edges */}
                <AnimatePresence>
                    {edges.map(edge => (
                        <motion.line
                            key={edge.id}
                            initial={{ pathLength: 0, opacity: 0 }}
                            animate={{ pathLength: 1, opacity: 1 }}
                            exit={{ opacity: 0 }}
                            x1={edge.x1}
                            y1={edge.y1}
                            x2={edge.x2}
                            y2={edge.y2}
                            stroke="var(--border-active)"
                            strokeWidth="2"
                            markerEnd="url(#arrowhead)"
                        />
                    ))}
                </AnimatePresence>

                {/* Nodes */}
                <AnimatePresence>
                    {nodes.map(node => {
                        const isHead = node.id === headCommitId;
                        return (
                            <motion.g
                                key={node.id}
                                initial={{ scale: 0, opacity: 0 }}
                                animate={{ scale: 1, opacity: 1 }}
                                exit={{ scale: 0, opacity: 0 }}
                                transition={{ type: "spring", stiffness: 300, damping: 20 }}
                            >
                                <circle
                                    cx={node.x}
                                    cy={node.y}
                                    r={NODE_RADIUS}
                                    fill={isHead ? "var(--bg-secondary)" : "var(--bg-primary)"}
                                    stroke={isHead ? "var(--accent-primary)" : "var(--text-tertiary)"}
                                    strokeWidth={isHead ? 3 : 2}
                                />
                                <text
                                    x={node.x}
                                    y={node.y + 4}
                                    textAnchor="middle"
                                    fill={isHead ? "var(--accent-primary)" : "var(--text-secondary)"}
                                    fontSize="10"
                                    style={{ fontFamily: 'monospace', pointerEvents: 'none', userSelect: 'none' }}
                                >
                                    {node.id}
                                </text>

                                {/* Message Label */}
                                <text
                                    x={node.x}
                                    y={node.y + NODE_RADIUS + 16}
                                    textAnchor="middle"
                                    fill="var(--text-secondary)"
                                    fontSize="10"
                                    style={{ whiteSpace: 'pre' }}
                                >
                                    {node.message}
                                </text>

                                {/* HEAD Indicator Label */}
                                {isHead && (
                                    <motion.text
                                        initial={{ y: -10, opacity: 0 }}
                                        animate={{ y: 0, opacity: 1 }}
                                        x={node.x}
                                        y={node.y - NODE_RADIUS - 8}
                                        textAnchor="middle"
                                        fill="var(--accent-primary)"
                                        fontSize="11"
                                        fontWeight="bold"
                                    >
                                        HEAD
                                    </motion.text>
                                )}
                            </motion.g>
                        );
                    })}
                </AnimatePresence>
            </svg>
        </div>
    );
};

export default GitGraphViz;
