import React, { useMemo, useRef, useState, useEffect, useLayoutEffect } from 'react';
import { useTranslation } from 'react-i18next';
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
    searchQuery?: string;
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
    title,
    searchQuery
}) => {
    const { state: contextState } = useGit();
    const { t } = useTranslation('common');
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

    // Virtualization State
    const containerRef = useRef<HTMLDivElement>(null);
    const [scrollTop, setScrollTop] = useState(0);
    const [viewportHeight, setViewportHeight] = useState(600); // Default fallback

    useLayoutEffect(() => {
        if (containerRef.current) {
            setViewportHeight(containerRef.current.clientHeight);
        }
    }, []);

    useEffect(() => {
        const el = containerRef.current;
        if (!el) return;
        const ro = new ResizeObserver(entries => {
            for (const entry of entries) {
                setViewportHeight(entry.contentRect.height);
            }
        });
        ro.observe(el);
        return () => ro.disconnect();
    }, []);

    const handleScroll = (e: React.UIEvent<HTMLDivElement>) => {
        setScrollTop(e.currentTarget.scrollTop);
    };

    const BUFFER_PX = 300;
    const minVisY = Math.max(0, scrollTop - BUFFER_PX);
    const maxVisY = scrollTop + viewportHeight + BUFFER_PX;

    // Dimming Logic based on search
    const filteredNodes = useMemo(() => {
        if (!searchQuery) return nodes;

        const query = searchQuery.toLowerCase();
        return nodes.map(node => {
            const formattedDate = new Date(node.timestamp).toLocaleString('ja-JP', {
                year: 'numeric', month: '2-digit', day: '2-digit',
                hour: '2-digit', minute: '2-digit', second: '2-digit'
            });

            const match =
                node.id.toLowerCase().includes(query) ||
                node.message.toLowerCase().includes(query) ||
                formattedDate.startsWith(query); // Prefix match for date

            return {
                ...node,
                opacity: match ? 1 : 0.2
                // We keep edges attached to dimmed nodes consistent with node opacity or keep them dimmed?
                // Visual preference: if node is dimmed, edges connecting it likely dimmed too.
            };
        });
    }, [nodes, searchQuery]);

    const activeNodes = searchQuery ? filteredNodes : nodes;

    // Auto-scroll to first match
    useEffect(() => {
        if (!searchQuery) return;
        const firstMatch = activeNodes.find(n => n.opacity === 1);
        if (firstMatch && containerRef.current) {
            containerRef.current.scrollTo({
                top: firstMatch.y - viewportHeight / 2,
                behavior: 'smooth'
            });
        }
    }, [searchQuery, activeNodes, viewportHeight]);

    // Filter for visibility (Virtualization)
    const visibleNodes = useMemo(() =>
        activeNodes.filter(n => n.y >= minVisY && n.y <= maxVisY),
        [activeNodes, minVisY, maxVisY]
    );

    const visibleEdges = useMemo(() =>
        edges.filter(e => e.maxY >= minVisY && e.minY <= maxVisY).map(edge => {
            // If search is active, dim edges if their connected nodes are dimmed?
            // This requires mapping edges to nodes. Simplified: if searchQuery, dim all edges to 0.2?
            // Or better: keep original edges opacity but multiply by 0.2 if no match?
            // Too complex to map dynamically without edge source/target ref.
            // Let's just dim all edges slightly during search to emphasize nodes.
            if (searchQuery) {
                return { ...edge, opacity: 0.1 };
            }
            return edge;
        }),
        [edges, minVisY, maxVisY, searchQuery]
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
            <div data-testid="git-graph-empty" className="flex h-full items-center justify-center text-gray-500 font-mono text-sm">
                {t('visualization.emptyState', { defaultValue: 'Type git init to start.' })}
            </div>
        );
    }

    // Processed empty state (Converted from initialized check)
    if (state.initialized && nodes.length === 0) {
        return (
            <div data-testid="git-graph-empty-commits" className="flex h-full flex-col items-center justify-center text-gray-500 gap-4" style={{ height: '100%', display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', color: 'var(--text-tertiary)', gap: '16px' }}>
                <div style={{ fontSize: '48px', opacity: 0.2 }}>📭</div>
                <div style={{ fontSize: '14px', textAlign: 'center' }}>
                    <div style={{ fontWeight: 600, marginBottom: '4px' }}>{t('visualization.emptyRepo.title', { defaultValue: 'Repository is empty' })}</div>
                    <div style={{ fontSize: '12px', opacity: 0.7 }}>{t('visualization.emptyRepo.description', { defaultValue: 'Push your first commit to see the graph.' })}</div>
                </div>
            </div>
        );
    }

    return (
        <div
            ref={containerRef}
            onScroll={handleScroll}
            data-testid="git-graph-container"
            style={{
                height: '100%',
                overflow: 'auto',
                background: 'var(--bg-primary)',
                color: 'var(--text-primary)',
                fontFamily: 'Menlo, Monaco, Consolas, monospace',
                fontSize: '12px'
            }}
        >
            {title && <GraphTitle title={title} />}

            <div style={{ position: 'relative', height: Math.max(height, 500) }}>
                {/* SVG Layer: Edges and Nodes */}
                <svg
                    width="100%"
                    height={height}
                    style={{ position: 'absolute', left: 0, top: 0, pointerEvents: 'none' }}
                >
                    {visibleEdges.map(edge => (
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
                        {visibleNodes.map(node => (
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
                                initial={false}
                                animate={{ scale: 1, opacity: node.opacity }}
                                exit={{ scale: 0, opacity: 0 }}
                                transition={{ type: 'spring', stiffness: 300, damping: 20 }}
                            />
                        ))}
                    </AnimatePresence>

                    {/* HEAD Halo */}
                    {visibleNodes.map(node =>
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
                {visibleNodes.map(node => (
                    <CommitRow
                        key={node.id}
                        node={node}
                        badges={badgesMap[node.id] || []}
                        isSelected={node.id === selectedCommitId}
                        onClick={onSelect ? () => onSelect(node) : undefined}
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
