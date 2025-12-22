import { useEffect, useRef } from 'react';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import '@xterm/xterm/css/xterm.css';
import { useGit } from '../../context/GitAPIContext';
import { useTheme } from '../../context/ThemeContext';

const GitTerminal = () => {
    const terminalRef = useRef<HTMLDivElement>(null);
    const xtermRef = useRef<Terminal | null>(null);
    const { runCommand, state, activeDeveloper, sessionId } = useGit();
    const runCommandRef = useRef(runCommand);
    const lastOutputLen = useRef(0);
    const lastCommandCount = useRef(0);
    const stateRef = useRef(state);
    const { theme } = useTheme();

    // Store processed state to detect user switches
    const lastActiveDeveloper = useRef(activeDeveloper);

    // Keep ref updated
    useEffect(() => {
        runCommandRef.current = runCommand;
        stateRef.current = state;
    }, [runCommand, state]);

    // Handle User Switch: Clear terminal and reset trackers
    useEffect(() => {
        if (lastActiveDeveloper.current !== activeDeveloper) {
            if (xtermRef.current) {
                xtermRef.current.clear(); // Clear visual buffer
                xtermRef.current.writeln(`\x1b[1;32mUser: ${activeDeveloper}\x1b[0m`);
            }
            lastOutputLen.current = 0; // Reset output to trigger replay
            lastCommandCount.current = -1; // Reset command count to ensure (current > last) triggers prompt

            lastActiveDeveloper.current = activeDeveloper;
        }
    }, [activeDeveloper]);

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

    // Effect to sync state output to terminal
    useEffect(() => {
        if (!xtermRef.current) return;

        // Safety Check: Ensure the state we are seeing belongs to the active session
        // This prevents displaying "stale" state from the previous user during async transition.
        const { _sessionId } = state;
        if (!_sessionId || _sessionId !== sessionId) return;

        // Write new output lines
        if (state.output.length > lastOutputLen.current) {
            const isFullReplay = lastOutputLen.current === 0;
            const newLines = state.output.slice(lastOutputLen.current);
            newLines.forEach(line => {
                // Skip command echo lines during incremental updates (already shown by xterm input)
                // Only write them during full replay (after tab switch when lastOutputLen was reset)
                if (!isFullReplay && line.startsWith('> ')) {
                    return;
                }

                let formattedLine = line;
                // Highlight simulation/dry-run
                if (line.includes('[dry-run]') || line.includes('[simulation]')) {
                    const ORANGE = '\x1b[38;5;214m';
                    const RESET = '\x1b[0m';
                    formattedLine = `${ORANGE}${line}${RESET}`;
                }
                xtermRef.current?.writeln(formattedLine);
            });
            lastOutputLen.current = state.output.length;
        }

        // Check if a command finished execution
        if (state.commandCount > lastCommandCount.current) {
            const prompt = getPrompt(state);
            xtermRef.current.write(`\r\n${prompt}`);
            lastCommandCount.current = state.commandCount;
        }
    }, [state.output, state.commandCount, state.HEAD, state.currentPath, state.initialized, state._sessionId, sessionId]);

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

        // Initial Theme
        if (theme === 'light') {
            term.options.theme = {
                background: '#ffffff',
                foreground: '#24292f',
                cursor: '#1f883d',
                selectionBackground: 'rgba(31, 136, 61, 0.3)',
            };
        }

        term.writeln('\x1b[1;32mGitGym Terminal\x1b[0m v1.0.0');
        term.writeln('Type "git clone <url>" to start.');

        term.write(getPrompt(stateRef.current));

        xtermRef.current = term;

        let currentLine = '';

        term.onData((data) => {
            const code = data.charCodeAt(0);

            // Enter key
            if (code === 13) {
                term.write('\r\n');
                const cmd = currentLine.trim();
                setTimeout(() => {
                    if (cmd) {
                        if (cmd === 'clear') {
                            term.clear();
                            term.write(getPrompt(stateRef.current));
                        } else {
                            // ENFORCE GIT COMMANDS ONLY
                            let fullCmd = cmd;
                            const parts = cmd.split(/\s+/);
                            const firstToken = parts[0];

                            // 1. If starts with "git", allow it (git commit, git status)
                            if (firstToken === 'git') {
                                // fullCmd is already good
                            }
                            // 2. If it is a known git subcommand (simple heuristic), prepend "git "
                            // Heuristic: If it's not a known shell command, assume it's a git subcommand target
                            // BUT user wants "Git Command Only".
                            // Let's TRY to run as "git <cmd>" if it doesn't start with git.
                            else {
                                // Auto-prefix
                                fullCmd = `git ${cmd}`;
                                // Optional logic: We could check if `cmd` is `ls` or `cd` and Block it?
                                // User said: "git コマンド以外のバックエンドコードはそのまま削除しないように"
                                // "ターミナルは... git コマンド限定にしたい"
                                // If I run `git ls`, git will complain "git: 'ls' is not a git command". This satisfies "Git commands only" (invalid ones are rejected by git).
                                term.writeln(`\r\n\x1b[90m(Auto-prefixed: ${fullCmd})\x1b[0m`);
                            }

                            if (runCommandRef.current) {
                                runCommandRef.current(fullCmd);
                            }
                        }
                    } else {
                        term.write(getPrompt(stateRef.current));
                    }
                }, 0);

                currentLine = '';
            }
            else if (code === 127) { // Backspace
                if (currentLine.length > 0) {
                    const charToRemove = currentLine.slice(-1);
                    const isWide = !!charToRemove.match(/[\u3000-\u303f\u3040-\u309f\u30a0-\u30ff\uff00-\uff9f\u4e00-\u9faf\u3400-\u4dbf]/);

                    if (isWide) {
                        term.write('\b\b  \b\b');
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
    }, []);

    // Theme Effect
    useEffect(() => {
        if (!xtermRef.current) return;
        if (theme === 'light') {
            xtermRef.current.options.theme = {
                background: '#ffffff',
                foreground: '#24292f',
                cursor: '#1f883d',
                selectionBackground: 'rgba(31, 136, 61, 0.3)',
            };
        } else {
            xtermRef.current.options.theme = {
                background: '#0d1117',
                foreground: '#c9d1d9',
                cursor: '#238636',
                selectionBackground: 'rgba(35, 134, 54, 0.3)',
            };
        }
    }, [theme]);

    return (
        <div style={{ width: '100%', height: '100%', display: 'flex', flexDirection: 'column' }}>
            <div
                ref={terminalRef}
                style={{ width: '100%', flex: 1, minHeight: 0 }}
            />
        </div>
    );
};

export default GitTerminal;
