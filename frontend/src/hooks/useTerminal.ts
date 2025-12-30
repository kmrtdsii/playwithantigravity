import { useEffect, useRef, useCallback, useState, type RefObject } from 'react';
import { useTranslation } from 'react-i18next';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { WebLinksAddon } from '@xterm/addon-web-links';
import { WebglAddon } from '@xterm/addon-webgl';
import { useGit } from '../context/GitAPIContext';
import { useTheme } from '../context/ThemeContext';
import { getPrompt } from '../components/terminal/getPrompt';
import { useTerminalOutput } from '../context/TerminalOutputContext';

/**
 * Hook to manage the Git terminal instance and its interaction with the global Git state.
 *
 * @param terminalRef - Ref to the DOM element where the terminal should be mounted.
 * @param xtermRef - Ref to the xterm.js instance.
 * @param fitAddonRef - Ref to the xterm.js fit addon instance.
 */
export const useTerminal = (
    terminalRef: RefObject<HTMLDivElement | null>,
    xtermRef: RefObject<Terminal | null>,
    fitAddonRef: RefObject<FitAddon | null>,
    allowEmptyCommit: boolean = true
) => {
    const { t } = useTranslation('common');
    const {
        runCommand,
        state,
        activeDeveloper,
        sessionId,
        appendToTranscript,
        terminalTranscripts,
        clearTranscript
    } = useGit();
    const { getOutput } = useTerminalOutput();

    // State Tracking Refs
    const lastOutputLengthRef = useRef(0);
    const lastPathRef = useRef(state.currentPath);
    const lastHeadRef = useRef(state.HEAD?.id);
    const isLocalCommandRef = useRef(false);
    const lastPromptTriggerRef = useRef(0);

    const allowEmptyCommitRef = useRef(allowEmptyCommit);
    useEffect(() => {
        allowEmptyCommitRef.current = allowEmptyCommit;
    }, [allowEmptyCommit]);

    const { theme } = useTheme();
    const [isReady, setIsReady] = useState(false);

    const currentLineRef = useRef('');
    const cursorPosRef = useRef(0);

    const inputPerDeveloperRef = useRef<Map<string, { text: string; cursor: number }>>(new Map());
    const prevDeveloperRef = useRef<string | null>(null);

    // Refs for input batching (rAF optimization)
    const pendingInputRef = useRef<string>('');
    const rafIdRef = useRef<number | null>(null);

    // Refs to avoid stale closures in callbacks
    const runCommandRef = useRef(runCommand);
    const appendToTranscriptRef = useRef(appendToTranscript);
    const clearTranscriptRef = useRef(clearTranscript);
    const stateRef = useRef(state);

    useEffect(() => {
        runCommandRef.current = runCommand;
        stateRef.current = state;
        appendToTranscriptRef.current = appendToTranscript;
        clearTranscriptRef.current = clearTranscript;
    }, [runCommand, state, appendToTranscript, clearTranscript]);

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

    // Track activeDeveloper for output syncing
    const activeDeveloperRef = useRef(activeDeveloper);
    useEffect(() => { activeDeveloperRef.current = activeDeveloper; }, [activeDeveloper]);

    const [promptTrigger, setPromptTrigger] = useState(0);


    // SYNC Output from Context
    // const output = getOutput(sessionId); // Unused here, we get it inside useEffect or init logic


    // --- INITIALIZATION & REPLAY ---
    useEffect(() => {
        if (!xtermRef.current || !isReady) return;

        xtermRef.current.write('\x1bc'); // Full Reset

        // Sync ref with current state level
        const transcript = terminalTranscripts[sessionId] || [];
        const currentOutput = getOutput(sessionId);
        const stateLen = currentOutput.length;
        lastOutputLengthRef.current = Math.max(transcript.length, stateLen);

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
                `\x1b[1;36m${t('terminal.welcome')}\x1b[0m`,
                t('terminal.instructions'),
                '  \x1b[33mmkdir project\x1b[0m',
                '  \x1b[33mcd project\x1b[0m',
                '  \x1b[33mgit init\x1b[0m',
                '  -- or --',
                '  \x1b[33mgit clone <url>\x1b[0m',
                '',
                `\x1b[32m${t('terminal.help')}\x1b[0m`,
                ''
            ];
            welcomeLines.forEach(line => {
                xtermRef.current?.writeln(line);
                if (appendToTranscriptRef.current) {
                    appendToTranscriptRef.current(line, true);
                }
            });
            // Write initial prompt
            const prompt = getPrompt(stateRef.current);
            xtermRef.current.write(`\x1b[2K\r${prompt}`);
            if (appendToTranscriptRef.current) {
                appendToTranscriptRef.current(`\x1b[2K\r${prompt}`, false);
            }
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

        // Note: terminalTranscripts intentionally excluded to prevent re-running on each transcript append
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [activeDeveloper, sessionId, fitAddonRef, xtermRef, isReady, t, getOutput]);

    // --- SYNC EXTERNAL & LOCAL COMMANDS ---
    useEffect(() => {
        const currentOutput = state.output; // Use state.output directly
        const currentLength = currentOutput.length;
        const prevLength = lastOutputLengthRef.current;
        const hasNewOutput = currentLength > prevLength;

        // Check for state changes relevant to prompt (Path, HEAD)
        const pathChanged = state.currentPath !== lastPathRef.current;
        const headChanged = state.HEAD?.id !== lastHeadRef.current;
        const promptTriggered = promptTrigger !== lastPromptTriggerRef.current;

        if (hasNewOutput || pathChanged || headChanged || promptTriggered) {
            // 1. Write New Output Lines
            if (hasNewOutput && xtermRef.current) {
                const newLines = currentOutput.slice(prevLength);
                // Optimized: Removed extra newline injection here to prevent layout gaps.

                newLines.forEach(line => {
                    let formatted = line;
                    if (line.includes('[dry-run]') || line.includes('[simulation]')) {
                        formatted = `\x1b[38;5;214m${line}\x1b[0m`;
                    }
                    xtermRef.current?.writeln(formatted);
                });
            }

            // 2. Write Prompt (ONLY IF NOT RUNNING LOCAL COMMAND)
            // If a local command is running, we wait for it to finish (promptTriggered) before writing prompt
            // This prevents the prompt from appearing while command output is streaming in or processing
            if (!isLocalCommandRef.current && xtermRef.current) {
                const prompt = getPrompt(state);

                // Write prompt to terminal (using clear line to update in place if valid)
                // We use \x1b[2K\r to clear the line and move caret to start
                xtermRef.current.write(`\x1b[2K\r${prompt}`);

                // Persist prompt to transcript
                if (appendToTranscriptRef.current) {
                    appendToTranscriptRef.current(`\x1b[2K\r${prompt}`, false);
                }

                // Restore user input (if any typed during refresh?)
                if (currentLineRef.current) {
                    xtermRef.current.write(currentLineRef.current);
                }
            }

            // Update Refs
            lastOutputLengthRef.current = currentLength;
            lastPathRef.current = state.currentPath;
            lastHeadRef.current = state.HEAD?.id;
            lastPromptTriggerRef.current = promptTrigger;
        }
    }, [sessionId, state.currentPath, state.HEAD, promptTrigger, state.output, state]);

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

        // Add WebLinksAddon for clickable URLs
        const webLinksAddon = new WebLinksAddon((_event, uri) => {
            window.open(uri, '_blank', 'noopener,noreferrer');
        });
        term.loadAddon(webLinksAddon);

        term.open(terminalRef.current);
        fitAddon.fit();

        // Load WebGL addon for GPU-accelerated rendering (with fallback for unsupported environments)
        try {
            const webglAddon = new WebglAddon();
            webglAddon.onContextLoss(() => {
                // Gracefully handle WebGL context loss by disposing and continuing with DOM renderer
                webglAddon.dispose();
            });
            term.loadAddon(webglAddon);
        } catch (e) {
            console.warn('WebGL addon failed to load, falling back to DOM renderer:', e);
        }

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

                // Optimized: Pre-increment output length to skip the "Echo" from context.
                // This prevents duplicate command display.
                lastOutputLengthRef.current += 1;

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
                        // Ideally clear output context too, but we don't have direct access here easily without exposing it.
                        // Assuming context clear is handled elsewhere or acceptable limitation.

                        // Add delay after reset to ensure terminal is ready
                        setTimeout(() => {
                            const prompt = getPrompt(stateRef.current);
                            writeAndRecord(prompt, false);
                        }, 50);
                        return;
                    }

                    let fullCmd = cmd;
                    let showAutoPrefixMsg = false;

                    const firstWord = cmd.split(' ')[0];
                    const shellCommands = ['ls', 'cd', 'pwd', 'touch', 'rm', 'mkdir', 'cat', 'echo', 'clear', 'help', 'version'];
                    const gitSubcommands = ['init', 'clone', 'add', 'commit', 'push', 'pull', 'fetch', 'branch', 'checkout', 'switch', 'merge', 'rebase', 'reset', 'restore', 'log', 'status', 'diff', 'remote', 'stash', 'tag', 'show', 'config', 'cherry-pick', 'reflog'];

                    if (!cmd.startsWith('git ')) {
                        if (shellCommands.includes(firstWord)) {
                            fullCmd = cmd;
                        } else if (gitSubcommands.includes(firstWord)) {
                            fullCmd = `git ${cmd}`;
                        } else {
                            fullCmd = cmd;
                        }
                    }

                    if (allowEmptyCommitRef.current) {
                        if (/^git\s+commit(\s|$)/.test(fullCmd)) {
                            if (!fullCmd.includes('--allow-empty')) {
                                fullCmd += ' --allow-empty';
                                showAutoPrefixMsg = true;
                            }
                        }
                    }

                    if (showAutoPrefixMsg) {
                        writeAndRecord(`\x1b[90m(Modified: ${fullCmd})\x1b[0m`, true);
                    }

                    isLocalCommandRef.current = true;

                    if (runCommandRef.current) {
                        try {
                            // We await command completion, BUT we don't handle output printing here anymore.
                            // Output is handled reactively by the effect observing TerminalOutputContext.
                            // This guarantees that as soon as output is available (from runCommand internal addOutput calls), it is displayed.
                            await runCommandRef.current(fullCmd);
                        } catch (e) {
                            writeAndRecord(`Error: ${e}`, true);
                        }
                    }

                    isLocalCommandRef.current = false;
                    setPromptTrigger(prev => prev + 1);

                }, 0);

            } else if (code === 127) { // BACKSPACE
                if (cursorPosRef.current > 0) {
                    const line = currentLineRef.current;
                    const pos = cursorPosRef.current;
                    const charToRemove = line.charAt(pos - 1);
                    const isWide = !!charToRemove.match(/[\u3000-\u303f\u3040-\u309f\u30a0-\u30ff\uff00-\uff9f\u4e00-\u9faf\u3400-\u4dbf]/);

                    currentLineRef.current = line.slice(0, pos - 1) + line.slice(pos);
                    cursorPosRef.current--;

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
                    currentLineRef.current = line.slice(0, pos) + line.slice(pos + 1);
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

                currentLineRef.current = line.slice(0, pos) + data + line.slice(pos);
                cursorPosRef.current += data.length;

                // Optimization: Batch end-of-line typing with requestAnimationFrame
                // For mid-line insertion, render immediately for proper cursor positioning
                if (pos === line.length) {
                    // Accumulate pending input for batched rendering
                    pendingInputRef.current += data;

                    if (rafIdRef.current === null) {
                        rafIdRef.current = requestAnimationFrame(() => {
                            if (pendingInputRef.current) {
                                term.write(pendingInputRef.current);
                                pendingInputRef.current = '';
                            }
                            rafIdRef.current = null;
                        });
                    }
                } else {
                    // Mid-line insertion: flush pending and render immediately
                    if (rafIdRef.current !== null) {
                        cancelAnimationFrame(rafIdRef.current);
                        rafIdRef.current = null;
                    }
                    if (pendingInputRef.current) {
                        term.write(pendingInputRef.current);
                        pendingInputRef.current = '';
                    }
                    const remaining = currentLineRef.current.slice(pos);
                    term.write(remaining + '\b'.repeat(remaining.length - data.length));
                }
            }
        });

        const resizeObserver = new ResizeObserver(() => fitAddon.fit());
        resizeObserver.observe(terminalRef.current);

        return () => {
            // Cancel any pending rAF to prevent memory leaks
            if (rafIdRef.current !== null) {
                cancelAnimationFrame(rafIdRef.current);
                rafIdRef.current = null;
            }
            resizeObserver.disconnect();
            term.dispose();
        };
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [terminalRef, writeAndRecord, xtermRef, fitAddonRef]);

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
