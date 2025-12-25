import { useState, useCallback } from 'react';

export interface TranscriptLine {
    text: string;
    hasNewline: boolean;
}

export const useTerminalTranscript = (sessionId: string) => {
    // Terminal Transcript Store
    const [terminalTranscripts, setTerminalTranscripts] = useState<Record<string, TranscriptLine[]>>({});

    const appendToTranscript = useCallback((text: string, hasNewline: boolean = true) => {
        if (!sessionId) return;

        const line: TranscriptLine = { text, hasNewline };

        setTerminalTranscripts(prev => {
            const current = prev[sessionId] || [];
            return {
                ...prev,
                [sessionId]: [...current, line]
            };
        });
    }, [sessionId]);

    const clearTranscript = useCallback(() => {
        if (!sessionId) return;
        setTerminalTranscripts(prev => ({
            ...prev,
            [sessionId]: []
        }));
    }, [sessionId]);

    return {
        terminalTranscripts, // Return state directly
        appendToTranscript,
        clearTranscript
    };
};
