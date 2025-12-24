import { useEffect, useRef, useCallback, useState } from 'react';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { useGit } from '../context/GitAPIContext';
import { useTheme } from '../context/ThemeContext';
import { getPrompt } from '../utils/terminalUtils';

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
    const [isReady, setIsReady] = useState(false);

    // Input buffer and cursor position
    const currentLineRef = useRef('');
    const cursorPosRef = useRef(0); // Position within currentLine (0 = start)

    // Per-developer input persistence (survives tab switches)
    const inputPerDeveloperRef = useRef<Map<string, { text: string; cursor: number }>>(new Map());
    const prevDeveloperRef = useRef<string | null>(null);

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

    const writeAndRecord = useCallback((text: string, hasNewline: boolean = true) => {
        if (!xtermRef.current) return;

        if (hasNewline) {
            xtermRef.current.writeln(text);
        } else {
            xtermRef.current.write(text);
        }

        if (appendToTranscriptRef.current) {
            appendToTranscriptRef.current(text, hasNewline);
        }
    }, [xtermRef]);

    // --- INITIALIZATION & REPLAY ---
    useEffect(() => {
        if (!xtermRef.current || !isReady) return;

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

        // Save previous developer's input before switching
        if (prevDeveloperRef.current && prevDeveloperRef.current !== activeDeveloper) {
            inputPerDeveloperRef.current.set(prevDeveloperRef.current, {
                text: currentLineRef.current,
                cursor: cursorPosRef.current
            });
        }

        // Restore input for current developer (if any)
        const savedInput = inputPerDeveloperRef.current.get(activeDeveloper);
        if (savedInput) {
            currentLineRef.current = savedInput.text;
            cursorPosRef.current = savedInput.cursor;
            // Write the restored input to terminal
            xtermRef.current?.write(savedInput.text);
        } else {
            currentLineRef.current = '';
            cursorPosRef.current = 0;
        }

        prevDeveloperRef.current = activeDeveloper;
        setTimeout(() => fitAddonRef.current?.fit(), 50);

    }, [activeDeveloper, sessionId, clearTranscript, getTranscript, writeAndRecord, fitAddonRef, xtermRef, isReady]);

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
    }, [state, state.output, state.HEAD, state.currentPath, writeAndRecord, xtermRef]);

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
        // Defer readiness to avoid synchronous state update warning
        setTimeout(() => setIsReady(true), 0);

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
                cursorPosRef.current = 0;

                if (!cmd) {
                    const prompt = getPrompt(stateRef.current);
                    writeAndRecord(prompt, false);
                    return;
                }

                setTimeout(async () => {
                    // Handle 'clear' command
                    if (cmd === 'clear') {
                        term.write('\x1bc'); // Full reset
                        if (clearTranscriptRef.current) clearTranscriptRef.current();
                        const prompt = getPrompt(stateRef.current);
                        writeAndRecord(prompt, false);
                        return;
                    }

                    let fullCmd = cmd;
                    let showAutoPrefixMsg = false;

                    const firstWord = cmd.split(' ')[0];
                    const shellCommands = ['ls', 'cd', 'pwd', 'touch', 'rm'];

                    // Determine if we need auto-prefix
                    if (!cmd.startsWith('git')) {
                        if (shellCommands.includes(firstWord)) {
                            // Shell commands: Pass through as-is, NO auto-prefix message
                            fullCmd = cmd;
                        } else {
                            // Unknown commands: Auto-prefix with git and SHOW message
                            fullCmd = `git ${cmd}`;
                            showAutoPrefixMsg = true;
                        }
                    }

                    // Display auto-prefix message if needed
                    if (showAutoPrefixMsg) {
                        writeAndRecord(`\x1b[90m(Auto-prefixed: ${fullCmd})\x1b[0m`, true);
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

                    setTimeout(() => {
                        const prompt = getPrompt(stateRef.current);
                        writeAndRecord(prompt, false);
                        isLocalCommandRef.current = false;
                    }, 50);

                }, 0);

            } else if (code === 127) { // BACKSPACE
                if (cursorPosRef.current > 0) {
                    const line = currentLineRef.current;
                    const pos = cursorPosRef.current;
                    const charToRemove = line.charAt(pos - 1);
                    const isWide = !!charToRemove.match(/[\u3000-\u303f\u3040-\u309f\u30a0-\u30ff\uff00-\uff9f\u4e00-\u9faf\u3400-\u4dbf]/);

                    // Remove char from buffer at cursor position
                    currentLineRef.current = line.slice(0, pos - 1) + line.slice(pos);
                    cursorPosRef.current--;

                    // Redraw: move back, print rest of line + space, move cursor back
                    const remaining = currentLineRef.current.slice(cursorPosRef.current);
                    if (isWide) {
                        term.write('\b\b' + remaining + '  \b\b' + '\b'.repeat(remaining.length));
                    } else {
                        term.write('\b' + remaining + ' ' + '\b'.repeat(remaining.length + 1));
                    }
                }
            } else if (data === '\x1b[3~') { // DELETE key
                const line = currentLineRef.current;
                const pos = cursorPosRef.current;
                if (pos < line.length) {
                    // Remove char at cursor position
                    currentLineRef.current = line.slice(0, pos) + line.slice(pos + 1);

                    // Redraw: print rest of line + space, move cursor back
                    const remaining = currentLineRef.current.slice(pos);
                    term.write(remaining + ' ' + '\b'.repeat(remaining.length + 1));
                }
            } else if (data === '\x1b[D') { // ARROW LEFT
                if (cursorPosRef.current > 0) {
                    cursorPosRef.current--;
                    term.write('\x1b[D'); // Move cursor left
                }
            } else if (data === '\x1b[C') { // ARROW RIGHT
                if (cursorPosRef.current < currentLineRef.current.length) {
                    cursorPosRef.current++;
                    term.write('\x1b[C'); // Move cursor right
                }
            } else if (data === '\x1b[H' || data === '\x1b[1~') { // HOME key
                if (cursorPosRef.current > 0) {
                    term.write('\x1b[' + cursorPosRef.current + 'D');
                    cursorPosRef.current = 0;
                }
            } else if (data === '\x1b[F' || data === '\x1b[4~') { // END key
                const moveRight = currentLineRef.current.length - cursorPosRef.current;
                if (moveRight > 0) {
                    term.write('\x1b[' + moveRight + 'C');
                    cursorPosRef.current = currentLineRef.current.length;
                }
            } else if (code === 3) { // CTRL+C
                currentLineRef.current = '';
                cursorPosRef.current = 0;
                writeAndRecord('^C', true);
                const prompt = getPrompt(stateRef.current);
                writeAndRecord(prompt, false);
            } else if (code >= 32) { // TYPING (printable chars)
                const line = currentLineRef.current;
                const pos = cursorPosRef.current;

                // Insert char at cursor position
                currentLineRef.current = line.slice(0, pos) + data + line.slice(pos);
                cursorPosRef.current += data.length;

                // If cursor is at end, just write the char
                if (pos === line.length) {
                    term.write(data);
                } else {
                    // Otherwise, redraw rest of line and move cursor back
                    const remaining = currentLineRef.current.slice(pos);
                    term.write(remaining + '\b'.repeat(remaining.length - data.length));
                }
            }
        });

        const resizeObserver = new ResizeObserver(() => fitAddon.fit());
        resizeObserver.observe(terminalRef.current);

        return () => {
            resizeObserver.disconnect();
            term.dispose();
        };
    }, [appendToTranscript, clearTranscript, fitAddonRef, runCommand, state, theme, terminalRef, writeAndRecord, xtermRef]);

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
    }, [theme, xtermRef]);
};
