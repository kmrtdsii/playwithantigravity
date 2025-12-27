import { useCallback } from 'react';
import { gitService } from '../services/gitService';
import type { GitDataHook } from './useGitData';

interface UseGitCommandProps {
    sessionId: string;
    gitData: GitDataHook;
}

export const useGitCommand = ({ sessionId, gitData }: UseGitCommandProps) => {
    const {
        fetchState,
        fetchServerState,
        serverState,
        setState,
        updateCommandOutput,
        incrementCommandCount,
        state
    } = gitData;

    const runCommand = useCallback(async (cmd: string, options?: { silent?: boolean; skipRefresh?: boolean }): Promise<string[]> => {
        if (!sessionId) {
            console.error("No session ID");
            return [];
        }

        console.log(`Executing command: ${cmd} (Session: ${sessionId})`);

        // 1. Echo Command
        const commandEcho = `> ${cmd}`;
        const currentOutput = state.output;
        // Note: state.output is from the current session view, which should match sessionId if synced.

        let newOutput = [...currentOutput];

        if (!options?.silent) {
            newOutput.push(commandEcho);
            // Optimistic update
            setState(prev => ({
                ...prev,
                output: newOutput
            }));
            updateCommandOutput(sessionId, newOutput);
        }

        try {
            const data = await gitService.executeCommand(sessionId, cmd);
            let responseLines: string[] = [];
            let isError = false;

            if (data.error) {
                responseLines = [`Error: ${data.error}`];
                isError = true;
            } else if (data.output) {
                responseLines = [data.output];
            }

            // 2. Append Output
            if (!options?.silent || isError) {
                newOutput = [...newOutput, ...responseLines];
                updateCommandOutput(sessionId, newOutput);
                setState(prev => ({
                    ...prev,
                    output: newOutput
                }));
            }

            // 3. Refresh State
            if (!options?.skipRefresh) {
                await fetchState(sessionId);
            }

            // 4. Increment Count & Trigger Prompt
            incrementCommandCount(sessionId);
            setState(prev => ({
                ...prev,
                commandCount: prev.commandCount + 1
            }));

            // 5. Auto-refresh Server State
            if (!options?.skipRefresh) {
                if (serverState && serverState.remotes?.length === 0) {
                    await fetchServerState('origin');
                } else if (serverState) {
                    // Default fallback refresh
                    await fetchServerState('origin');
                }
            }

            return responseLines;

        } catch (e) {
            console.error(e);
            const errorLine = "Network error";
            const finalOutput = [...newOutput, errorLine];
            updateCommandOutput(sessionId, finalOutput);
            setState(prev => ({ ...prev, output: finalOutput, commandCount: prev.commandCount + 1 }));
            incrementCommandCount(sessionId);
            return [errorLine];
        }
    }, [sessionId, state.output, fetchState, fetchServerState, serverState, setState, updateCommandOutput, incrementCommandCount]);

    return { runCommand };
};
