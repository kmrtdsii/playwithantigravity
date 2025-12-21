import { useEffect, useRef } from 'react';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import '@xterm/xterm/css/xterm.css';
import { useGit } from '../../context/GitAPIContext';

const GitTerminal = () => {
    const terminalRef = useRef<HTMLDivElement>(null);
    const xtermRef = useRef<Terminal | null>(null);
    const { runCommand, state } = useGit();
    const runCommandRef = useRef(runCommand);
    const lastOutputLen = useRef(0);
    const lastCommandCount = useRef(0);
    const stateRef = useRef(state);

    // Keep ref updated
    useEffect(() => {
        runCommandRef.current = runCommand;
        stateRef.current = state;
    }, [runCommand, state]);

    // Helper to generate Powerline-style prompt
    const getPrompt = (currentState: typeof state) => {
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
        // Format path: if empty, show /
        const displayPath = (path === '') ? '/' : path;

        // Check if no project is selected (root path)
        if (displayPath === '/') {
            return `${RESET}${displayPath} > `;
        }

        const hasRepo = currentState.HEAD && currentState.HEAD.type !== 'none';

        let p = '';

        // SEGMENT 1: Path
        // Use Nerd Font folder icon (\uf07c) instead of Emoji to avoid width issues
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

    // Effect to sync state output to terminal
    useEffect(() => {
        if (!xtermRef.current) return;

        // Write new output lines
        if (state.output.length > lastOutputLen.current) {
            const newLines = state.output.slice(lastOutputLen.current);
            newLines.forEach(line => {
                xtermRef.current?.writeln(line);
            });
            lastOutputLen.current = state.output.length;
        }

        // Check if a command finished execution
        if (state.commandCount > lastCommandCount.current) {
            const prompt = getPrompt(state);
            xtermRef.current.write(`\r\n${prompt}`);
            lastCommandCount.current = state.commandCount;
        }

    }, [state.output, state.commandCount, state.HEAD, state.currentPath, state.initialized]);

    useEffect(() => {
        if (!terminalRef.current) return;

        // Initialize Xterm
        const term = new Terminal({
            cursorBlink: true,
            theme: {
                background: '#0d1117',
                foreground: '#c9d1d9',
                cursor: '#238636',
                selectionBackground: 'rgba(35, 134, 54, 0.3)',
            },
            fontFamily: '"MesloLGS NF", Menlo, Monaco, "Courier New", monospace',
            fontSize: 14,
            convertEol: true,
        });

        const fitAddon = new FitAddon();
        term.loadAddon(fitAddon);

        term.open(terminalRef.current);
        fitAddon.fit();

        term.writeln('\x1b[1;32mGitGym Terminal\x1b[0m v1.0.0');
        term.writeln('Type "git clone <url>" to start.');

        // Initial Prompt
        // Need to use current state, but effect might run before state is populated?
        // Use default initial state implied prompt.
        term.write(getPrompt(stateRef.current));

        xtermRef.current = term;

        let currentLine = '';

        term.onData((data) => {
            const code = data.charCodeAt(0);

            // Enter key
            if (code === 13) {
                term.write('\r\n');
                const cmd = currentLine.trim();
                setTimeout(() => { // Small delay to allow react command dispatch if needed, usually direct is fine
                    if (cmd) {
                        if (cmd === 'clear') {
                            term.clear();
                            term.write(getPrompt(stateRef.current));
                        } else {
                            // Run command
                            if (runCommandRef.current) {
                                runCommandRef.current(cmd);
                            }
                            // The prompt will be written by the useEffect when commandCount increases
                        }
                    } else {
                        // Empty command, just new prompt
                        term.write(getPrompt(stateRef.current));
                    }
                }, 0);

                currentLine = '';
            }
            else if (code === 127) { // Backspace
                if (currentLine.length > 0) {
                    const charToRemove = currentLine.slice(-1);
                    // Simple CJK detection (incomplete but covers most common cases)
                    // Regular expression for Full-width characters
                    const isWide = !!charToRemove.match(/[\u3000-\u303f\u3040-\u309f\u30a0-\u30ff\uff00-\uff9f\u4e00-\u9faf\u3400-\u4dbf]/);

                    if (isWide) {
                        term.write('\b\b  \b\b'); // Move back 2, clear 2, move back 2
                    } else {
                        term.write('\b \b');
                    }
                    currentLine = currentLine.slice(0, -1);
                }
            }
            else if (code === 3) { // Ctrl+C
                currentLine = '';
                term.write(`^C\r\n${getPrompt(stateRef.current)}`);
            }
            else if (code >= 32) {
                currentLine += data;
                term.write(data);
            }
        });

        const resizeObserver = new ResizeObserver(() => fitAddon.fit());
        resizeObserver.observe(terminalRef.current);

        return () => {
            resizeObserver.disconnect();
            term.dispose();
        };
    }, []); // Run once on mount

    return (
        <div
            ref={terminalRef}
            style={{ width: '100%', flex: 1, minHeight: 0 }}
        />
    );
};

export default GitTerminal;
