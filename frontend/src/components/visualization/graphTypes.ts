import type { Commit } from '../../types/gitTypes';

/**
 * Extended Commit type with visualization positioning data
 */
export interface VizNode extends Commit {
    /** X coordinate in SVG */
    x: number;
    /** Y coordinate in SVG */
    y: number;
    /** Lane index (column) for this commit */
    lane: number;
    /** Color assigned based on lane */
    color: string;
    /** Whether this is a simulated/potential commit */
    isGhost: boolean;
    /** Opacity for reachability visualization (0-1) */
    opacity: number;
}

/**
 * Edge connecting two commits in the graph
 */
export interface VizEdge {
    /** Unique identifier (child-parent format) */
    id: string;
    /** SVG path data string */
    path: string;
    /** Color matching the child node */
    color: string;
    /** Whether this edge involves ghost commits */
    isGhost: boolean;
    /** Opacity for reachability visualization */
    opacity: number;
    /** Minimum Y coordinate for visibility optimization */
    minY: number;
    /** Maximum Y coordinate for visibility optimization */
    maxY: number;
}

/**
 * Badge displayed next to a commit (branch, tag, HEAD)
 */
export interface Badge {
    /** Display text (e.g., "main", "v1.0", "HEAD") */
    text: string;
    /** Badge type for styling */
    type: 'branch' | 'head' | 'tag' | 'remote-branch';
    /** Whether this is the currently checked out branch */
    isActive?: boolean;
}

/**
 * Layout computation result
 */
export interface LayoutResult {
    /** Positioned commit nodes */
    nodes: VizNode[];
    /** Edges connecting commits */
    edges: VizEdge[];
    /** Total height of the graph in pixels */
    height: number;
    /** Map of commit ID to badges */
    badgesMap: Record<string, Badge[]>;
}
