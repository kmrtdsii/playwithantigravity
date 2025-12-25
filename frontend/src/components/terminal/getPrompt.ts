import type { GitState } from '../../types/gitTypes';

// ANSI Color Codes
const ANSI_RESET = '\x1b[0m';
const ANSI_BLUE_BG = '\x1b[44m';
const ANSI_BLUE_FG = '\x1b[34m';
const ANSI_WHITE_FG = '\x1b[97m';
const ANSI_YELLOW_BG = '\x1b[43m';
const ANSI_YELLOW_FG = '\x1b[33m';
const ANSI_BLACK_FG = '\x1b[30m';

const SEP = '\ue0b0';
const BRANCH_ICON = '\ue0a0';

/**
 * Generates a Powerline-style terminal prompt string based on the current Git state.
 * 
 * It constructs the prompt with:
 * 1. Current path (Blue background)
 * 2. Git branch/HEAD info (Yellow background) - only if inside a repo
 * 
 * @param currentState - The full Git state object
 * @returns ANSI-escaped string for the terminal prompt
 */
export function getPrompt(currentState: GitState): string {
    const path = currentState.currentPath || '/';
    const displayPath = (path === '') ? '/' : path;

    if (displayPath === '/') {
        return `${ANSI_RESET}${displayPath} > `;
    }

    const hasRepo = currentState.HEAD && currentState.HEAD.type !== 'none';

    let p = '';

    // SEGMENT 1: Path
    p += `${ANSI_BLUE_BG}${ANSI_WHITE_FG} \uf07c ${displayPath} `;

    if (hasRepo) {
        // TRANSITION 1: Blue -> Yellow
        p += `${ANSI_YELLOW_BG}${ANSI_BLUE_FG}${SEP}`;

        // SEGMENT 2: Git Info
        const branch = currentState.HEAD.ref || currentState.HEAD.id?.substring(0, 7) || 'DETACHED';
        p += `${ANSI_BLACK_FG} ${BRANCH_ICON} ${branch} `;

        // END: Yellow -> Default
        p += `${ANSI_RESET}${ANSI_YELLOW_FG}${SEP}${ANSI_RESET} `;
    } else {
        // END: Blue -> Default
        p += `${ANSI_RESET}${ANSI_BLUE_FG}${SEP}${ANSI_RESET} `;
    }

    return p;
}
