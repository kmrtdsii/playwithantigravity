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

    // Keep ref updated
    useEffect(() => {
        runCommandRef.current = runCommand;
    }, [runCommand]);

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
            xtermRef.current.write('\r\n$ ');
            lastCommandCount.current = state.commandCount;
        }

    }, [state.output, state.commandCount]);

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
                    if (cmd === 'clear') {
                        term.clear();
                        term.write('$ ');
                    } else {
                        // Run command - result handling is in the other useEffect
                        console.log("GitTerminal: Invoking runCommand with:", cmd);
                        if (runCommandRef.current) {
                            runCommandRef.current(cmd);
                        }
                    }
                } else {
                    term.write('$ ');
                }
                currentLine = '';
            }
            else if (code === 127) { // Backspace
                if (currentLine.length > 0) {
                    term.write('\b \b');
                    currentLine = currentLine.slice(0, -1);
                }
            }
            else if (code === 3) { // Ctrl+C
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
    }, []); // Run once on mount

    return (
        <div
            ref={terminalRef}
            style={{ width: '100%', flex: 1, minHeight: 0 }}
        />
    );
};

export default GitTerminal;
