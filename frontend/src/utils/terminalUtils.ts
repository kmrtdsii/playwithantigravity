import type { GitState } from '../types/gitTypes';

// Helper to generate Powerline-style prompt
export const getPrompt = (currentState: GitState) => {
    const RESET = '\x1b[0m';
    const BLUE_BG = '\x1b[44m';
    const BLUE_FG = '\x1b[34m';
    const WHITE_FG = '\x1b[97m';
    const YELLOW_BG = '\x1b[43m';
    const YELLOW_FG = '\x1b[33m';
    const BLACK_FG = '\x1b[30m';

    const SEP = '\ue0b0';
    const BRANCH_ICON = '\ue0a0';

    const path = currentState.currentPath || '/';
    const displayPath = (path === '') ? '/' : path;

    if (displayPath === '/') {
        return `${RESET}${displayPath} > `;
    }

    const hasRepo = currentState.HEAD && currentState.HEAD.type !== 'none';

    let p = '';

    // SEGMENT 1: Path
    p += `${BLUE_BG}${WHITE_FG} \uf07c ${displayPath} `;

    if (hasRepo) {
        // TRANSITION 1: Blue -> Yellow
        p += `${YELLOW_BG}${BLUE_FG}${SEP}`;

        // SEGMENT 2: Git Info
        const branch = currentState.HEAD.ref || currentState.HEAD.id?.substring(0, 7) || 'DETACHED';
        p += `${BLACK_FG} ${BRANCH_ICON} ${branch} `;

        // END: Yellow -> Default
        p += `${RESET}${YELLOW_FG}${SEP}${RESET} `;
    } else {
        // END: Blue -> Default
        p += `${RESET}${BLUE_FG}${SEP}${RESET} `;
    }

    return p;
};
