/**
 * GitGraphViz Constants
 * Layout and styling constants for the Git graph visualization
 */

/** Height of each commit row in pixels */
export const ROW_HEIGHT = 36;

/** Width of each lane (branch column) in pixels */
export const LANE_WIDTH = 18;

/** Radius of commit node circles in pixels */
export const CIRCLE_RADIUS = 5;

/** Top padding for the graph area */
export const PADDING_TOP = 24;

/** Left padding before the first lane */
export const GRAPH_LEFT_PADDING = 24;

/**
 * Color palette for branch lanes
 * Colors rotate based on lane index
 */
export const LANE_COLORS = [
    '#58a6ff', // Blue
    '#d2a8ff', // Purple
    '#3fb950', // Green
    '#ffa657', // Orange
    '#ff7b72', // Red
    '#79c0ff', // Light Blue
    '#f2cc60', // Yellow
    '#56d364', // Light Green
] as const;

/** Type for lane color values */
export type LaneColor = typeof LANE_COLORS[number];
