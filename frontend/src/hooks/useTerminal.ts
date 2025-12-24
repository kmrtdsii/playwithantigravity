import { useEffect, useRef } from 'react';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { useGit } from '../context/GitAPIContext';
import { useTheme } from '../context/ThemeContext';
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

export const useTerminal = (
    terminalRef: React.RefObject<HTMLDivElement | null>,
    xtermRef: React.MutableRefObject<Terminal | null>,
    fitAddonRef: React.MutableRefObject<FitAddon | null>
) => {
    const {
        runCommand,
        state,
        activeDeveloper,
        sessionId,
        appendToTranscript,
        getTranscript,
        clearTranscript
    } = useGit();

    const { theme } = useTheme();

    // Input buffer
    const currentLineRef = useRef('');

    // Refs to avoid stale closures in callbacks
    const runCommandRef = useRef(runCommand);
    const appendToTranscriptRef = useRef(appendToTranscript);
    const getTranscriptRef = useRef(getTranscript);
    const clearTranscriptRef = useRef(clearTranscript);
    const stateRef = useRef(state);

    // Command Syncing Refs
    const isLocalCommandRef = useRef(false);
    const lastOutputLengthRef = useRef(0);

    useEffect(() => {
        runCommandRef.current = runCommand;
        stateRef.current = state;
        appendToTranscriptRef.current = appendToTranscript;
        getTranscriptRef.current = getTranscript;
        clearTranscriptRef.current = clearTranscript;
    }, [runCommand, state, appendToTranscript, getTranscript, clearTranscript]);

    const writeAndRecord = (text: string, hasNewline: boolean = true) => {
        if (!xtermRef.current) return;

        if (hasNewline) {
            xtermRef.current.writeln(text);
        } else {
            xtermRef.current.write(text);
        }

        if (appendToTranscriptRef.current) {
            appendToTranscriptRef.current(text, hasNewline);
        }
    };

    // --- INITIALIZATION & REPLAY ---
    useEffect(() => {
        if (!xtermRef.current) return;

        xtermRef.current.write('\x1bc'); // Full Reset

        // Sync ref with current state length to avoid re-printing history handled by replay
        // We defer to Replay logic, and just set the index.
        const transcript = getTranscriptRef.current ? getTranscriptRef.current() : [];
        lastOutputLengthRef.current = transcript.length > 0 ? transcript.length : 0;

        if (transcript.length > 0) {
            transcript.forEach(line => {
                if (line.hasNewline) {
                    xtermRef.current?.writeln(line.text);
                } else {
                    xtermRef.current?.write(line.text);
                }
            });
        } else {
            const welcomeLines = [
                '\x1b[1;36mWelcome to GitGym!\x1b[0m 🚀',
                'To get started, please clone a repository using:',
                '  \x1b[33mgit clone <url>\x1b[0m',
                '',
                "Type \x1b[32m'git help'\x1b[0m to see available commands.",
                ''
            ];

            welcomeLines.forEach(line => writeAndRecord(line, true));
            const prompt = getPrompt(stateRef.current);
            writeAndRecord(prompt, false);
        }

        currentLineRef.current = '';
        setTimeout(() => fitAddonRef.current?.fit(), 50);

    }, [activeDeveloper, sessionId]);

    // --- SYNC EXTERNAL COMMANDS ---
    useEffect(() => {
        const currentLength = state.output.length;
        const prevLength = lastOutputLengthRef.current;

        if (currentLength > prevLength) {
            // New output detected!
            if (!isLocalCommandRef.current && xtermRef.current) {
                const newLines = state.output.slice(prevLength);

                xtermRef.current.write('\r\n'); // Move to new line

                newLines.forEach(line => {
                    xtermRef.current?.writeln(line);
                });

                // Re-render prompt with NEW state
                const prompt = getPrompt(state);
                xtermRef.current.write(prompt);
            }
            // Always update ref
            lastOutputLengthRef.current = currentLength;
        }
    }, [state.output, state.HEAD, state.currentPath]);

    // --- SETUP XTERM ---
    useEffect(() => {
        if (!terminalRef.current) return;

        const term = new Terminal({
            cursorBlink: true,
            theme: theme === 'light' ? {
                background: '#ffffff',
                foreground: '#24292f',
                cursor: '#1f883d',
                selectionBackground: 'rgba(31, 136, 61, 0.3)',
            } : {
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

        xtermRef.current = term;

        // --- COMMAND HANDLER ---
        term.onData((data) => {
            const code = data.charCodeAt(0);

            // 1. ENTER KEY
            if (code === 13) {
                const cmd = currentLineRef.current.trim();
                const rawInput = currentLineRef.current;

                term.write('\r\n');

                if (appendToTranscriptRef.current) {
                    appendToTranscriptRef.current(rawInput, true);
                }

                currentLineRef.current = '';

                if (!cmd) {
                    const prompt = getPrompt(stateRef.current);
                    writeAndRecord(prompt, false);
                    return;
                }

                setTimeout(async () => {
                    if (cmd === 'clear') {
                        term.write('\x1bc'); // Full reset
                        if (clearTranscriptRef.current) clearTranscriptRef.current();
                        const prompt = getPrompt(stateRef.current);
                        writeAndRecord(prompt, false);
                        return;
                    }

                    if (cmd) {
                        let fullCmd = cmd;
                        if (!cmd.startsWith('git ')) {
                            if (cmd === 'git') {
                                // just 'git' is fine
                            } else {
                                fullCmd = `git ${cmd}`;
                                writeAndRecord(`\x1b[90m(Auto-prefixed: ${fullCmd})\x1b[0m`, true);
                            }
                        }

                        isLocalCommandRef.current = true;

                        if (runCommandRef.current) {
                            try {
                                const outputLines = await runCommandRef.current(fullCmd);
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

                    setTimeout(() => {
                        const prompt = getPrompt(stateRef.current);
                        writeAndRecord(prompt, false);
                        isLocalCommandRef.current = false;
                    }, 50);

                }, 0);

            } else if (code === 127) { // BACKSPACE
                if (currentLineRef.current.length > 0) {
                    const charToRemove = currentLineRef.current.slice(-1);
                    const isWide = !!charToRemove.match(/[\u3000-\u303f\u3040-\u309f\u30a0-\u30ff\uff00-\uff9f\u4e00-\u9faf\u3400-\u4dbf]/);

                    if (isWide) term.write('\b\b  \b\b');
                    else term.write('\b \b');

                    currentLineRef.current = currentLineRef.current.slice(0, -1);
                }
            } else if (code === 3) { // CTRL+C
                currentLineRef.current = '';
                writeAndRecord('^C', true);
                const prompt = getPrompt(stateRef.current);
                writeAndRecord(prompt, false);
            } else if (code >= 32) { // TYPING
                currentLineRef.current += data;
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

    // Theme Update
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
};
