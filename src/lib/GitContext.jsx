import React, { createContext, useContext, useReducer } from 'react';
import { gitReducer, initialState } from './git-simulation';

const GitContext = createContext(null);

export const GitProvider = ({ children }) => {
    const [state, dispatch] = useReducer(gitReducer, initialState);

    const runCommand = (cmdString) => {
        const parts = cmdString.trim().split(/\s+/);
        const cmd = parts[0];

        if (cmd !== 'git') {
            return { output: `Command not found: ${cmd}` };
        }

        // Basic argument parsing (can be replaced by a library if needed, but manual is fine for limited set)
        const args = parts.slice(1);
        const subCmd = args[0];
        const flags = args.filter(a => a.startsWith('-'));
        const params = args.filter(a => !a.startsWith('-') && a !== subCmd);

        switch (subCmd) {
            case 'init':
                dispatch({ type: 'INIT' });
                break;

            case 'status':
                dispatch({ type: 'STATUS' });
                break;

            case 'add':
                if (params.length === 0) {
                    // If dot?
                    if (args.includes('.')) {
                        dispatch({ type: 'ADD', payload: { files: ['file.txt', 'style.css'] } }); // Simulator magic
                    } else {
                        return { output: 'Nothing specified, nothing added.' };
                    }
                } else {
                    dispatch({ type: 'ADD', payload: { files: params } });
                }
                break;

            case 'commit':
                const mIndex = args.indexOf('-m');
                let message = '';
                if (mIndex !== -1 && args[mIndex + 1]) {
                    // Reconstruct message from parts after -m
                    // This is tricky with split(/\s+/) because it eats internal spaces.
                    // Better to parse raw string ? 
                    // Quick fix: find "-m" in original string?
                    const match = cmdString.match(/-m\s+"([^"]+)"/) || cmdString.match(/-m\s+([^" ]+)/);
                    if (match) {
                        message = match[1];
                    } else {
                        return { output: 'Aborting commit due to empty commit message.' };
                    }
                } else if (flags.includes('-m')) {
                    return { output: 'error: switch `m` requires a value' };
                }

                // If user didn't modify anything? (Simulated environment)
                // For now, if staging is empty, maybe we reject in reducer.
                dispatch({ type: 'COMMIT', payload: { message } });
                break;

            case 'branch':
                if (params.length > 0) {
                    dispatch({ type: 'BRANCH', payload: { name: params[0] } });
                } else {
                    // TODO: List branches
                    return {
                        output: Object.keys(state.branches).map(b =>
                            (state.HEAD.type === 'branch' && state.HEAD.ref === b ? '* ' : '  ') + b
                        ).join('\n')
                    };
                }
                break;

            case 'checkout':
                if (flags.includes('-b')) {
                    if (params.length > 0) {
                        dispatch({ type: 'CHECKOUT', payload: { ref: params[0], isNewBranch: true } });
                    } else {
                        return { output: "fatal: 'checkout' -b requires a branch name" };
                    }
                } else {
                    if (params.length > 0) {
                        dispatch({ type: 'CHECKOUT', payload: { ref: params[0] } });
                    }
                }
                break;

            case 'merge':
                if (params.length > 0) {
                    dispatch({ type: 'MERGE', payload: { source: params[0] } });
                } else {
                    return { output: 'fatal: No branch specified.' };
                }
                break;

            case 'log':
                // Return log string directly? Or dispatch LOG?
                // Let's just generate log from state here for simplicity or dispatch to add to output
                // Reducer is cleaner for "action history".
                // But formatting logic might be complex for reducer.
                // Let's do it here? No, stick to reducer for output consistency.
                // Wait, I didn't verify LOG in reducer.

                // Let's just format it here and return { output: ... } ?
                // No, that bypasses the "terminal history" being part of state?
                // Currently state.output accumulates ALL history. 
                // So we should dispatch.
                // Wait, I removed LOG case from my reducer update above? 
                // Ah, I added ACTION_TYPES.LOG but didn't implement the case in the reducer block I replaced.
                // I should add LOG case to reducer.

                // For now, let's just make LOG return output directly to terminal, 
                // BUT current terminal implementation relies on `state.output` changes to print!
                // If I just return { output }, my GitTerminal code handles it?
                // Yes, I added logic: `if (res && res.output) term.writeln...`

                const logOutput = state.commits.slice().sort((a, b) => new Date(b.timestamp) - new Date(a.timestamp))
                    .map(c => `\x1b[33m${c.id}\x1b[0m ${c.message}`).join('\n');
                return { output: logOutput || '(No commits yet)' };

            default:
                return { output: `git: '${subCmd}' is not a git command.` };
        }
    };

    return (
        <GitContext.Provider value={{ state, runCommand }}>
            {children}
        </GitContext.Provider>
    );
};

export const useGit = () => {
    const context = useContext(GitContext);
    if (!context) {
        throw new Error('useGit must be used within a GitProvider');
    }
    return context;
};
