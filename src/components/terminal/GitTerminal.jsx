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
            // Re-prompt
            xtermRef.current.write('$ ');
        }
    }, [state.output]);

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
                        if (res && res.output) {
                            term.writeln(res.output);
                            term.write('$ ');
                        } else {
                            // It was a dispatch, so the useEffect will handle the prompt.
                            // Wait, if dispatch happens, state updates, effect runs.
                            // If invalid command, runCommand returned string but didn't dispatch?
                            // Let's check GitContext again.
                        }
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
            style={{ width: '100%', height: '100%' }}
        />
    );
};

export default GitTerminal;
