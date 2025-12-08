import React, { createContext, useContext, useReducer } from 'react';
import { gitReducer, initialState } from './git-simulation';

const GitContext = createContext(null);

export const GitProvider = ({ children }) => {
    const [state, dispatch] = useReducer(gitReducer, initialState);

    const runCommand = (cmdString) => {
        const parts = cmdString.trim().split(' ');
        const cmd = parts[0];
        const subCmd = parts[1];

        if (cmd !== 'git') {
            return { output: `Command not found: ${cmd}` };
        }

        switch (subCmd) {
            case 'init':
                dispatch({ type: 'INIT' });
                break;
            case 'commit':
                // parse -m "message"
                const mIndex = parts.indexOf('-m');
                let message = 'update';
                if (mIndex !== -1 && parts[mIndex + 1]) {
                    // simple join for now, robust parsing later
                    message = parts.slice(mIndex + 1).join(' ').replace(/"/g, '');
                }
                dispatch({ type: 'COMMIT', payload: { message } });
                break;
            case 'branch':
                if (parts[2]) {
                    dispatch({ type: 'BRANCH', payload: { name: parts[2] } });
                } else {
                    // list branches logic needed in reducer or here
                    // For now just error
                }
                break;
            case 'checkout':
                if (parts[2]) {
                    dispatch({ type: 'CHECKOUT', payload: { ref: parts[2] } });
                }
                break;
            default:
                // do nothing or return error
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
