import React, { useMemo, useState } from 'react';
import { useGit } from '../../context/GitAPIContext';
import type { Commit, GitState } from '../../types/gitTypes';
import type { VizNode } from './graphTypes';
import { computeLayout } from './computeLayout';
import { ROW_HEIGHT, CIRCLE_RADIUS } from './graphConstants';

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

    // Hover state for row highlighting
    const [hoveredId, setHoveredId] = useState<string | null>(null);

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
        <div style={{
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

                    {nodes.map(node => (
                        <circle
                            key={node.id}
                            cx={node.x}
                            cy={node.y}
                            r={CIRCLE_RADIUS}
                            fill={node.isGhost ? "transparent" : node.color}
                            stroke={node.color}
                            strokeWidth={node.isGhost ? "2" : "1"}
                            strokeDasharray={node.isGhost ? "3,3" : "0"}
                            opacity={node.opacity}
                        />
                    ))}

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
                        isHovered={node.id === hoveredId}
                        isSelected={node.id === selectedCommitId}
                        onHover={setHoveredId}
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

interface CommitRowProps {
    node: VizNode;
    badges: Array<{ text: string; type: string; isActive?: boolean }>;
    isHovered: boolean;
    isSelected: boolean;
    onHover: (id: string | null) => void;
    onClick: () => void;
}

const TEXT_OFFSET_X = 140;

const CommitRow: React.FC<CommitRowProps> = ({
    node,
    badges,
    isHovered,
    isSelected,
    onHover,
    onClick
}) => (
    <div
        onMouseEnter={() => onHover(node.id)}
        onMouseLeave={() => onHover(null)}
        onClick={onClick}
        style={{
            position: 'absolute',
            left: 0,
            top: node.y - ROW_HEIGHT / 2,
            width: '100%',
            paddingLeft: TEXT_OFFSET_X,
            boxSizing: 'border-box',
            height: ROW_HEIGHT,
            display: 'flex',
            alignItems: 'center',
            whiteSpace: 'nowrap',
            gap: '8px',
            cursor: 'pointer',
            paddingRight: '16px',
            userSelect: 'none',
            opacity: node.opacity,
            backgroundColor: isHovered || isSelected ? 'rgba(59, 130, 246, 0.1)' : 'transparent',
            borderLeft: isHovered || isSelected ? '4px solid var(--accent-primary)' : '4px solid transparent',
        }}
        className="commit-row"
    >
        {/* Badges */}
        {badges.length > 0 && (
            <div style={{ display: 'flex', gap: '4px' }}>
                {badges.map((badge, i) => (
                    <CommitBadge key={i} badge={badge} color={node.color} />
                ))}
            </div>
        )}

        {/* Message */}
        <span
            title={node.message}
            style={{
                color: node.isGhost ? 'var(--text-tertiary)' : 'var(--text-secondary)',
                fontStyle: node.isGhost ? 'italic' : 'normal',
                flex: 1,
                overflow: 'hidden',
                textOverflow: 'ellipsis',
                minWidth: 0,
            }}
        >
            {node.isGhost && '[SIMULATION] '}
            {node.message}
        </span>

        {/* Timestamp */}
        <span style={{
            color: 'var(--text-tertiary)',
            fontSize: '10px',
            width: '140px',
            textAlign: 'right',
            flexShrink: 0,
            marginRight: '8px'
        }}>
            {new Date(node.timestamp).toLocaleString('ja-JP', {
                year: 'numeric', month: '2-digit', day: '2-digit',
                hour: '2-digit', minute: '2-digit', second: '2-digit'
            })}
        </span>

        {/* Commit ID */}
        <span style={{
            color: isSelected ? 'var(--accent-primary, #3b82f6)' : 'var(--text-tertiary)',
            fontSize: '10px',
            width: '60px',
            textAlign: 'right',
            flexShrink: 0,
            fontWeight: isSelected ? 'bold' : 'normal',
        }}>
            {node.id.substring(0, 7)}
        </span>
    </div>
);

interface CommitBadgeProps {
    badge: { text: string; type: string; isActive?: boolean };
    color: string;
}

const CommitBadge: React.FC<CommitBadgeProps> = ({ badge, color }) => (
    <span style={{
        fontSize: '10px',
        padding: '1px 6px',
        borderRadius: '10px',
        fontWeight: badge.isActive ? 'bold' : 'normal',
        backgroundColor: 'transparent',
        border: `1px solid ${color}`,
        color: color,
        opacity: 0.9,
        display: 'flex',
        alignItems: 'center'
    }}>
        {badge.type === 'tag' && <TagIcon />}
        {badge.type === 'remote-branch' && <CloudIcon />}
        {badge.text}
    </span>
);

const TagIcon: React.FC = () => (
    <svg
        viewBox="0 0 24 24"
        width="11"
        height="11"
        stroke="currentColor"
        strokeWidth="2.5"
        fill="none"
        strokeLinecap="round"
        strokeLinejoin="round"
        style={{ marginRight: '4px', verticalAlign: 'middle', display: 'inline-block', flexShrink: 0, opacity: 0.8 }}
    >
        <path d="M12 2H2v10l9.29 9.29c.94.94 2.48.94 3.42 0l7.29-7.29c.94-.94.94-2.48 0-3.42L12 2z" />
        <path d="M7 7h.01" />
    </svg>
);

const CloudIcon: React.FC = () => (
    <svg
        viewBox="0 0 24 24"
        width="11"
        height="11"
        stroke="currentColor"
        strokeWidth="2.5"
        fill="none"
        strokeLinecap="round"
        strokeLinejoin="round"
        style={{ marginRight: '4px', verticalAlign: 'middle', display: 'inline-block', flexShrink: 0, opacity: 0.8 }}
    >
        <path d="M17.5 19c0-3.037-2.463-5.5-5.5-5.5S6.5 15.963 6.5 19" />
        <path d="M20.9 14.1a6 6 0 1 0-8.9-8.1 4 4 0 0 0-5.8 5.7" />
    </svg>
);

export default GitGraphViz;
