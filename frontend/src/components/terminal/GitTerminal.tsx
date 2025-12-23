import { useEffect, useRef } from 'react';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import '@xterm/xterm/css/xterm.css';
import { useGit } from '../../context/GitAPIContext';
import { useTheme } from '../../context/ThemeContext';

const GitTerminal = () => {
    const terminalRef = useRef<HTMLDivElement>(null);
    const xtermRef = useRef<Terminal | null>(null);
    const {
        runCommand,
        state,
        activeDeveloper,
        sessionId,
        appendToTranscript,
        getTranscript,
        clearTranscript
    } = useGit();

    const runCommandRef = useRef(runCommand);
    const appendToTranscriptRef = useRef(appendToTranscript);
    const getTranscriptRef = useRef(getTranscript);
    const clearTranscriptRef = useRef(clearTranscript);
    const stateRef = useRef(state);
    const { theme } = useTheme();
    const fitAddonRef = useRef<FitAddon | null>(null);



    // Input buffer
    const currentLineRef = useRef('');

    // Update refs
    useEffect(() => {
        runCommandRef.current = runCommand;
        stateRef.current = state;
        appendToTranscriptRef.current = appendToTranscript;
        getTranscriptRef.current = getTranscript;
        clearTranscriptRef.current = clearTranscript;
    }, [runCommand, state, appendToTranscript, getTranscript, clearTranscript]);

    // --- RECORDER PATTERN ---
    // Single source of truth for writing to terminal AND saving history.
    const writeAndRecord = (text: string, hasNewline: boolean = true) => {
        if (!xtermRef.current) return;

        // 1. Write to visual terminal
        if (hasNewline) {
            xtermRef.current.writeln(text);
        } else {
            xtermRef.current.write(text);
        }

        // 2. Record to transcript (if available)
        if (appendToTranscriptRef.current) {
            appendToTranscriptRef.current(text, hasNewline);
        }
    };

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

    // --- INITIALIZATION & REPLAY ---
    // When developer switches or component mounts, replay history or show welcome.
    useEffect(() => {
        if (!xtermRef.current) return;

        // Reset Terminal Visually
        xtermRef.current.write('\x1bc'); // Full Reset

        // Fetch History
        const transcript = getTranscriptRef.current ? getTranscriptRef.current() : [];

        if (transcript.length > 0) {
            // REPLAY MODE: Exact reproduction of history
            transcript.forEach(line => {
                if (line.hasNewline) {
                    xtermRef.current?.writeln(line.text);
                } else {
                    xtermRef.current?.write(line.text);
                }
            });
        } else {
            // FRESH SESSION MODE: Show Welcome & Initial Prompt
            // Note: We intentionally record these to the transcript so they persist!
            const welcomeLines = [
                '\x1b[1;36mWelcome to GitGym!\x1b[0m 🚀',
                'To get started, please clone a repository using:',
                '  \x1b[33mgit clone <url>\x1b[0m',
                '',
                "Type \x1b[32m'git help'\x1b[0m to see available commands.",
                ''
            ];

            welcomeLines.forEach(line => writeAndRecord(line, true));

            // Initial Prompt
            const prompt = getPrompt(stateRef.current);
            writeAndRecord(prompt, false);
        }

        // Reset local trackers
        currentLineRef.current = '';

        // Refit
        setTimeout(() => fitAddonRef.current?.fit(), 50);

    }, [activeDeveloper, sessionId]);

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
            fontFamily: '"JetBrains Mono", "Fira Code", "MesloLGS NF", Menlo, Monaco, "Courier New", monospace',
            fontSize: 13,
            lineHeight: 1.1,
            convertEol: true,
            allowProposedApi: true,
        });

        const fitAddon = new FitAddon();
        fitAddonRef.current = fitAddon;
        term.loadAddon(fitAddon);
        term.open(terminalRef.current);
        fitAddon.fit();

        // Save instance
        xtermRef.current = term;

        // Apply Theme (Initial)
        if (theme === 'light') {
            term.options.theme = {
                background: '#ffffff',
                foreground: '#24292f',
                cursor: '#1f883d',
                selectionBackground: 'rgba(31, 136, 61, 0.3)',
            };
        }

        // --- COMMAND HANDLER (REPL) ---
        term.onData((data) => {
            const code = data.charCodeAt(0);

            // 1. ENTER KEY
            if (code === 13) {
                // Determine command
                const cmd = currentLineRef.current.trim();
                const rawInput = currentLineRef.current;

                // Visual Echo
                term.write('\r\n');

                // Record Input to Transcript
                if (appendToTranscriptRef.current) {
                    appendToTranscriptRef.current(rawInput, true);
                }

                // Clear Buffer
                currentLineRef.current = '';

                // Handle empty command immediately
                if (!cmd) {
                    const prompt = getPrompt(stateRef.current);
                    writeAndRecord(prompt, false);
                    return;
                }

                // Execute Command Async
                setTimeout(async () => {
                    if (cmd === 'clear') {
                        // Special Handling for Clear
                        term.write('\x1bc'); // Visual Reset

                        // Clear Transcript History
                        if (clearTranscriptRef.current) {
                            clearTranscriptRef.current();
                        }

                        // Write Fresh Prompt
                        const prompt = getPrompt(stateRef.current);
                        writeAndRecord(prompt, false);
                        return;
                    }

                    if (cmd) {
                        // Enforce Git & Auto-prefix
                        let fullCmd = cmd;
                        if (!cmd.startsWith('git ')) {
                            if (cmd === 'git') {
                                // just 'git' is fine
                            } else {
                                fullCmd = `git ${cmd}`;
                                writeAndRecord(`\x1b[90m(Auto-prefixed: ${fullCmd})\x1b[0m`, true);
                            }
                        }

                        // Run Command
                        if (runCommandRef.current) {
                            try {
                                const outputLines = await runCommandRef.current(fullCmd); // Returns string[]
                                // Output lines are plain text usually (or ANSI)
                                outputLines.forEach(line => {
                                    let formatted = line;
                                    if (line.includes('[dry-run]') || line.includes('[simulation]')) {
                                        formatted = `\x1b[38;5;214m${line}\x1b[0m`;
                                    }
                                    writeAndRecord(formatted, true);
                                });
                            } catch (e) {
                                writeAndRecord(`Error: ${e}`, true);
                            }
                        }
                    }

                    // Write New Prompt (using LATEST state after command)
                    setTimeout(() => {
                        const prompt = getPrompt(stateRef.current);
                        writeAndRecord(prompt, false);
                    }, 50);

                }, 0);

                // 2. BACKSPACE
            } else if (code === 127) {
                if (currentLineRef.current.length > 0) {
                    const charToRemove = currentLineRef.current.slice(-1);
                    const isWide = !!charToRemove.match(/[\u3000-\u303f\u3040-\u309f\u30a0-\u30ff\uff00-\uff9f\u4e00-\u9faf\u3400-\u4dbf]/);

                    if (isWide) term.write('\b\b  \b\b');
                    else term.write('\b \b');

                    currentLineRef.current = currentLineRef.current.slice(0, -1);
                }

                // 3. CTRL+C
            } else if (code === 3) {
                currentLineRef.current = '';
                writeAndRecord('^C', true);
                const prompt = getPrompt(stateRef.current);
                writeAndRecord(prompt, false);

                // 4. TYPING
            } else if (code >= 32) {
                currentLineRef.current += data;
                term.write(data); // Echo back (Visualization only, no record until Enter)
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

    const currentBranch = state.HEAD?.ref || (state.HEAD?.id ? state.HEAD.id.substring(0, 7) : 'DETACHED');
    const isDetached = !state.HEAD?.ref && !!state.HEAD?.id;

    return (
        <div style={{ width: '100%', height: '100%', display: 'flex', flexDirection: 'column', boxSizing: 'border-box', background: 'var(--bg-primary)' }}>
            {/* Persistent Terminal Status Bar */}
            <div style={{
                height: 'var(--header-height)',
                display: 'flex',
                alignItems: 'center',
                padding: '0 var(--space-3)',
                background: 'var(--bg-secondary)',
                borderBottom: '1px solid var(--border-subtle)',
                fontSize: 'var(--text-xs)',
                gap: 'var(--space-4)',
                flexShrink: 0
            }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--radius-md)' }}>
                    <span style={{ color: 'var(--text-secondary)', fontWeight: 'var(--font-extrabold)', fontSize: 'var(--text-xs)', letterSpacing: '0.05em' }}>User:</span>
                    <span style={{ color: 'var(--accent-primary)', fontWeight: 'var(--font-semibold)' }}>{activeDeveloper}</span>
                </div>
                <div style={{ width: '1px', height: '12px', background: 'var(--border-subtle)' }} />
                <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--radius-md)' }}>
                    <span style={{ color: 'var(--text-secondary)', fontWeight: 'var(--font-extrabold)', fontSize: 'var(--text-xs)', letterSpacing: '0.05em' }}>Branch:</span>
                    <span style={{
                        color: isDetached ? 'var(--text-warning)' : 'var(--text-secondary)',
                        fontFamily: 'monospace'
                    }}>
                        {currentBranch}
                    </span>
                </div>
            </div>

            <div style={{ flex: 1, minHeight: 0, paddingLeft: 'var(--space-4)', paddingTop: 'var(--space-3)', paddingBottom: 'var(--space-3)' }}>
                <div
                    ref={terminalRef}
                    style={{ width: '100%', height: '100%' }}
                />
            </div>
        </div>
    );
};

export default GitTerminal;
