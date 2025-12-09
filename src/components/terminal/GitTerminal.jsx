import React, { useEffect, useRef } from 'react';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import '@xterm/xterm/css/xterm.css';
import { useGit } from '../../lib/GitContext';

const GitTerminal = () => {
    const terminalRef = useRef(null);
    const xtermRef = useRef(null);
    const { runCommand, state } = useGit();
    const lastOutputLen = useRef(0);

    // Effect to sync state output to terminal
    useEffect(() => {
        if (!xtermRef.current) return;

        if (state.output.length > lastOutputLen.current) {
            const newLines = state.output.slice(lastOutputLen.current);
            newLines.forEach(line => {
                xtermRef.current.writeln(line);
            });
            lastOutputLen.current = state.output.length;
        }

        // Re-prompt (always, if state changed)
        // Check if we already prompted for this state version?
        // Simple heuristic: If output length changed OR lastUpdated changed.
        // But initial mount shouldn't double prompt.
        // We can just call write('$ ') here if it's not the very first render?
        // Actually, we printed initial $ manually.

        // Let's just write prompt if we processed something.
        if (state.lastUpdated > 0 || state.output.length > 0) {
            xtermRef.current.write('\r\n$ ');
        }

    }, [state.output, state.lastUpdated]);

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
            fontFamily: 'Menlo, Monaco, "Courier New", monospace',
            fontSize: 14,
            convertEol: true,
        });

        const fitAddon = new FitAddon();
        term.loadAddon(fitAddon);

        term.open(terminalRef.current);
        fitAddon.fit();

        term.writeln('\x1b[1;32mGitForge Terminal\x1b[0m v1.0.0');
        term.writeln('Type "git init" to start.');
        term.write('$ ');

        xtermRef.current = term;

        let currentLine = '';

        term.onData((data) => {
            const code = data.charCodeAt(0);

            // Enter key
            if (code === 13) {
                term.write('\r\n');
                const cmd = currentLine.trim();
                if (cmd) {
                    // Check for clear command locally
                    if (cmd === 'clear') {
                        term.clear();
                        term.write('$ ');
                    } else {
                        // Note: We don't write '$ ' here immediately because we wait for output effect
                        // But GitContext might not produce output for unknown commands if I didn't handle it. 
                        // Currently GitContext returns object for unknown commands, need to handle that.
                        // Actually my GitContext implementation for unknown subcmd returns { output: ... }
                        // creating a discrepancy. I should make runCommand ALWAYS update state or handle return.

                        // Let's modify logic: if runCommand returns an object with output, we print it manually? 
                        const res = runCommand(cmd);

                        // Always write prompt back, even if no output
                        if (res && res.output) {
                            term.writeln(res.output);
                        }

                        // If output didn't change (silent command), we need to write prompt here.
                        // But if output DID change, the effect will write prompt.
                        // We need to coordinate.

                        // Check if state output length will change? 
                        // Actually, since dispatch is async, 'state' here is old.
                        // We can't know easily.

                        // Better strategy: Have runCommand return { output, silent: true/false } ?
                        // Or just let the effect handle EVERYTHING, and ensure 'silent' commands
                        // still append a dummy empty line or we use a transaction ID in state?

                        // Quick fix: Set a timeout to check if output changed? No, flaky.
                        // Reliable fix: Add a 'lastCommandId' to git state.

                        // For now, let's just write the prompt IF runCommand return value suggests no dispatch
                        // OR if we assume all dispatches update output.

                        // Wait, `git add` in reducer does: output: state.output
                        // So length never changes. 
                        // So effect never fires.
                        // So we MUST write prompt here if we detect no output change intention?

                        if (!res && state.output.length === lastOutputLen.current) {
                            // This assumes synchronous update? No, state update is async.
                            // If we dispatched, state WILL change eventually... but content might be identical?
                            // React won't re-render if state is identical.

                            // So we MUST return something from reducer.
                            // OR handle it here.
                        }

                        // Simplest fix: Force prompt here if we know the command is 'add'?
                        // No, generic.

                        // Case A: Command returns { output: ... } (handled below)
                        // Case B: Command dispatches. Loop relies on state change.

                        // If I change `GitTerminal` to NOT use effect for prompt, but manual?
                        // But `state.output` comes asynchronously.

                        // Best Fix: Update `GitContext/git-simulation.js` to ALWAYS return a new output array 
                        // even for silent commands (e.g. append null or empty string, or filter in UI).
                        // OR add a 'timestamp' to state.

                    }
                } else {
                    term.write('$ ');
                }
                currentLine = '';
            }
            else if (code === 127) {
                if (currentLine.length > 0) {
                    term.write('\b \b');
                    currentLine = currentLine.slice(0, -1);
                }
            }
            else if (code === 3) {
                currentLine = '';
                term.write('^C\r\n$ ');
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

    return (
        <div
            ref={terminalRef}
            style={{ width: '100%', flex: 1, minHeight: 0 }}
        />
    );
};

export default GitTerminal;
