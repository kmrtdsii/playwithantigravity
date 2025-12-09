import React, { useMemo } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { useGit } from '../../context/GitAPIContext';
import type { Commit, GitState } from '../../context/GitAPIContext';

// Basic constants
const NODE_RADIUS = 16;
const X_SPACING = 80;
const Y_SPACING = 60;
const START_X = 50;
const START_Y = 100;

interface Node extends Commit {
    x: number;
    y: number;
    lane: number;
}

interface Edge {
    id: string;
    x1: number;
    y1: number;
    x2: number;
    y2: number;
    isMerge: boolean;
}

interface Label {
    text: string;
    type: 'branch' | 'head';
}

// Helper to compute layout
const computeLayout = (commits: Commit[], branches: Record<string, string>, HEAD: GitState['HEAD']) => {
    if (commits.length === 0) return { nodes: [], edges: [], headCommitId: null, labelsMap: {} };

    // 1. Assign Lanes
    const laneMap: Record<string, number> = { 'main': 0 };
    let nextLane = 1;

    commits.forEach(c => {
        const branchName = c.branch || 'detached';
        if (laneMap[branchName] === undefined) {
            laneMap[branchName] = nextLane++;
        }
    });

    const positions: Record<string, { x: number, y: number, lane: number }> = {}; // commitId -> pos

    // 2. Compute Positions
    commits.forEach((c, index) => {
        const lane = laneMap[c.branch || 'detached'] || 0;
        positions[c.id] = {
            x: START_X + index * X_SPACING,
            y: START_Y + lane * Y_SPACING,
            lane
        };
    });

    // 3. Generate Nodes
    const nodes: Node[] = commits.map(c => ({
        ...c,
        ...positions[c.id]
    }));

    // 4. Generate Edges
    const edges: Edge[] = [];
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

    // 5. Determine Active Status (HEAD)
    let headCommitId: string | null = null;
    if (HEAD.type === 'commit') headCommitId = HEAD.id || null;
    else if (HEAD.type === 'branch' && HEAD.ref) headCommitId = branches[HEAD.ref];

    // 6. Compute Labels
    const labelsMap: Record<string, Label[]> = {};

    // Add branches
    Object.entries(branches).forEach(([name, commitId]) => {
        if (!commitId) return;
        if (!labelsMap[commitId]) labelsMap[commitId] = [];
        labelsMap[commitId].push({ text: name, type: 'branch' });
    });

    // Add HEAD
    if (headCommitId) {
        if (!labelsMap[headCommitId]) labelsMap[headCommitId] = [];
        labelsMap[headCommitId].push({ text: 'HEAD', type: 'head' });
    }

    return { nodes, edges, headCommitId, labelsMap };
};

const GitGraphViz: React.FC = () => {
    const { state, runCommand } = useGit();
    const { commits, branches, HEAD } = state;

    const { nodes, edges, headCommitId, labelsMap } = useMemo(() =>
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

    const getFileStatus = (file: string) => {
        const isStaged = state.staging.includes(file);
        const isModified = state.modified.includes(file);
        if (isStaged) return 'staged';
        if (isModified) return 'modified';
        return 'clean';
    };

    return (
        <div style={{ width: '100%', height: '100%', display: 'flex', flexDirection: 'column' }}>
            <div style={{
                display: 'flex',
                gap: '1rem',
                padding: '1rem',
                borderBottom: '1px solid var(--border-primary)',
                backgroundColor: 'var(--bg-secondary)',
                flexShrink: 0
            }}>
                <div style={{ flex: 1, border: '1px solid var(--border-primary)', borderRadius: '4px', padding: '0.5rem' }}>
                    <h3 style={{ margin: '0 0 0.5rem', fontSize: '0.8rem', color: 'var(--text-primary)', textTransform: 'uppercase' }}>Working Directory (Files)</h3>
                    {state.files && state.files.length > 0 ? (
                        <ul style={{ margin: 0, paddingLeft: '0', fontSize: '0.85rem', listStyle: 'none' }}>
                            {state.files.map(f => {
                                const status = getFileStatus(f);
                                let color = 'var(--text-tertiary)';
                                if (status === 'modified') color = '#e5534b';
                                else if (status === 'staged') color = '#238636';

                                return (
                                    <li key={f} style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '4px', color }}>
                                        <span>{f} <span style={{ fontSize: '0.7em', opacity: 0.7 }}>({status})</span></span>
                                        {status === 'clean' || status === 'staged' ? (
                                            <button
                                                onClick={() => runCommand(`touch ${f}`)}
                                                style={{
                                                    background: 'none',
                                                    border: '1px solid var(--border-primary)',
                                                    borderRadius: '3px',
                                                    color: 'var(--text-secondary)',
                                                    cursor: 'pointer',
                                                    fontSize: '0.7rem',
                                                    padding: '2px 6px',
                                                    position: 'relative',
                                                    zIndex: 10
                                                }}
                                                title="Edit file (touch)"
                                            >
                                                Edit
                                            </button>
                                        ) : null}
                                    </li>
                                );
                            })}
                        </ul>
                    ) : (
                        <div style={{ fontSize: '0.8rem', color: 'var(--text-tertiary)', fontStyle: 'italic' }}>No files</div>
                    )}
                </div>

                <div style={{ flex: 1, border: '1px solid var(--border-primary)', borderRadius: '4px', padding: '0.5rem' }}>
                    <h3 style={{ margin: '0 0 0.5rem', fontSize: '0.8rem', color: '#238636', textTransform: 'uppercase' }}>Staging Area</h3>
                    {state.staging && state.staging.length > 0 ? (
                        <ul style={{ margin: 0, paddingLeft: '1.2rem', fontSize: '0.85rem' }}>
                            {state.staging.map(f => <li key={f} style={{ color: '#238636' }}>{f}</li>)}
                        </ul>
                    ) : (
                        <div style={{ fontSize: '0.8rem', color: 'var(--text-tertiary)', fontStyle: 'italic' }}>Empty</div>
                    )}
                </div>
            </div>

            <div style={{ flex: 1, overflow: 'auto', position: 'relative' }}>
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

                                    {(labelsMap[node.id] || []).map((label, i) => {
                                        const yPos = node.y - NODE_RADIUS - 10 - (i * 14);
                                        const isHeadLabel = label.type === 'head';

                                        return (
                                            <motion.text
                                                key={`${node.id}-${label.text}`}
                                                initial={{ y: -10, opacity: 0 }}
                                                animate={{ y: yPos - node.y + NODE_RADIUS + 10 + (i * 14), opacity: 1 }}
                                                x={node.x}
                                                y={yPos}
                                                textAnchor="middle"
                                                fill={isHeadLabel ? "var(--accent-primary)" : "var(--text-secondary)"}
                                                fontSize="11"
                                                fontWeight={isHeadLabel ? "bold" : "normal"}
                                            >
                                                {label.text}
                                            </motion.text>
                                        );
                                    })}
                                </motion.g>
                            );
                        })}
                    </AnimatePresence>
                </svg>
            </div>
        </div>
    );
};

export default GitGraphViz;
