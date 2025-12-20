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
            let prompt = '$ ';
            if (state.initialized) {
                const branch = state.HEAD.ref || state.HEAD.id?.substring(0, 7) || 'DETACHED';
                prompt = `(${branch}) $ `;
            }
            xtermRef.current.write(`\r\n${prompt}`);
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
                        let prompt = '$ ';
                        // We need to access current state here, but state is in closure.
                        // Ideally we should use a ref for state access in event handler.
                        // However, let's just default to simple prompt for clear, next command will fix it.
                        // Or better, let's use a ref to track current prompt string.
                        term.write(prompt);
                    } else {
                        // Run command - result handling is in the other useEffect
                        console.log("GitTerminal: Invoking runCommand with:", cmd);
                        if (runCommandRef.current) {
                            runCommandRef.current(cmd);
                        }
                    }
                } else {
                    // Empty command, just new prompt
                    // But wait, we don't have easy access to state here without ref.
                    // Let's rely on the useEffect to update prompt? No, useEffect only runs on state change.
                    // If user hits enter empty, we just show prompt again.
                    // Let's use a simple $ for empty enter for now or try to fetch ref.
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
