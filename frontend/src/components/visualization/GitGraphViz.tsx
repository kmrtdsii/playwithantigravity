import React, { useMemo } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { useGit } from '../../context/GitAPIContext';
import type { Commit, GitState } from '../../types/gitTypes';
import { computeLayout } from './computeLayout';
import { CIRCLE_RADIUS } from './graphConstants';
import { CommitRow } from './CommitRow';

// --- Component Props ---
interface GitGraphVizProps {
    onSelect?: (commit: Commit) => void;
    selectedCommitId?: string;
    state?: GitState;
    title?: string;
}

/**
 * GitGraphViz - Visual representation of Git commit history
 * 
 * Renders an interactive SVG-based graph showing:
 * - Commit nodes with lane-based coloring
 * - Edges connecting parent/child commits
 * - Badges for branches, tags, and HEAD
 * - Hover/selection highlighting
 */
const GitGraphViz: React.FC<GitGraphVizProps> = ({
    onSelect,
    selectedCommitId,
    state: propState,
    title
}) => {
    const { state: contextState } = useGit();
    const state = propState || contextState;
    const { commits, potentialCommits, branches, references, remoteBranches, tags, HEAD } = state;

    // Hover state removed for performance (handled by CSS)

    // Compute layout with memoization
    const { nodes, edges, height, badgesMap } = useMemo(() =>
        computeLayout(
            commits,
            potentialCommits || [],
            branches,
            references || {},
            remoteBranches || {},
            tags || {},
            HEAD
        ),
        [commits, potentialCommits, branches, references, remoteBranches, tags, HEAD]
    );

    // Resolve HEAD commit ID for halo effect
    const headCommitId = useMemo(() => {
        if (HEAD.type === 'commit') return HEAD.id;
        if (HEAD.type === 'branch' && HEAD.ref) return branches[HEAD.ref];
        return undefined;
    }, [HEAD, branches]);

    // Empty state
    if (!state.initialized) {
        return (
            <div className="flex h-full items-center justify-center text-gray-500 font-mono text-sm">
                Type <code className="mx-1 text-gray-400">git init</code> to start.
            </div>
        );
    }

    return (
        <div data-testid="git-graph-container" style={{
            height: '100%',
            overflow: 'auto',
            background: 'var(--bg-primary)',
            color: 'var(--text-primary)',
            fontFamily: 'Menlo, Monaco, Consolas, monospace',
            fontSize: '12px'
        }}>
            {title && <GraphTitle title={title} />}

            <div style={{ position: 'relative', height: Math.max(height, 500) }}>
                {/* SVG Layer: Edges and Nodes */}
                <svg
                    width="100%"
                    height={height}
                    style={{ position: 'absolute', left: 0, top: 0, pointerEvents: 'none' }}
                >
                    {edges.map(edge => (
                        <path
                            key={edge.id}
                            d={edge.path}
                            stroke={edge.color}
                            strokeWidth="2"
                            fill="none"
                            strokeLinecap="round"
                            strokeDasharray={edge.isGhost ? "4,4" : "0"}
                            opacity={edge.opacity}
                        />
                    ))}

                    <AnimatePresence>
                        {nodes.map(node => (
                            <motion.circle
                                key={node.id}
                                cx={node.x}
                                cy={node.y}
                                r={CIRCLE_RADIUS}
                                fill={node.isGhost ? "transparent" : node.color}
                                stroke={node.color}
                                strokeWidth={node.isGhost ? "2" : "1"}
                                strokeDasharray={node.isGhost ? "3,3" : "0"}
                                opacity={node.opacity}
                                initial={{ scale: 0, opacity: 0 }}
                                animate={{ scale: 1, opacity: node.opacity }}
                                exit={{ scale: 0, opacity: 0 }}
                                transition={{ type: 'spring', stiffness: 300, damping: 20 }}
                            />
                        ))}
                    </AnimatePresence>

                    {/* HEAD Halo */}
                    {nodes.map(node =>
                        node.id === headCommitId && (
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
                        )
                    )}
                </svg>

                {/* Interactive Rows */}
                {nodes.map(node => (
                    <CommitRow
                        key={node.id}
                        node={node}
                        badges={badgesMap[node.id] || []}
                        isSelected={node.id === selectedCommitId}
                        onClick={() => onSelect?.(node)}
                    />
                ))}
            </div>

            {/* Legend for Ghost Mode */}
            {state.potentialCommits && state.potentialCommits.length > 0 && (
                <div style={{
                    position: 'absolute',
                    bottom: '16px',
                    right: '16px',
                    background: 'var(--bg-secondary)',
                    border: '1px solid var(--border-subtle)',
                    borderRadius: '6px',
                    padding: '8px 12px',
                    fontSize: '11px',
                    boxShadow: '0 4px 12px rgba(0,0,0,0.2)',
                    zIndex: 20
                }}>
                    <div style={{ fontWeight: 600, marginBottom: '4px', color: 'var(--text-secondary)' }}>Simulation Mode</div>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '6px', color: 'var(--text-tertiary)' }}>
                        <span style={{
                            width: '10px',
                            height: '10px',
                            borderRadius: '50%',
                            border: '2px dashed var(--text-tertiary)',
                            display: 'inline-block'
                        }}></span>
                        <span>Potential Commit</span>
                    </div>
                </div>
            )}
        </div>
    );
};

// --- Sub-Components ---

interface GraphTitleProps {
    title: string;
}

const GraphTitle: React.FC<GraphTitleProps> = ({ title }) => (
    <div style={{
        padding: '8px 16px',
        fontSize: '11px',
        fontWeight: 700,
        color: 'var(--text-tertiary)',
        textTransform: 'uppercase',
        letterSpacing: '0.05em',
        borderBottom: '1px solid var(--border-subtle)',
        background: 'var(--bg-secondary)',
        position: 'sticky',
        top: 0,
        zIndex: 10
    }}>
        {title}
    </div>
);

export default GitGraphViz;
