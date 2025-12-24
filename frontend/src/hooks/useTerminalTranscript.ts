import { useState, useRef, useEffect } from 'react';

export interface TranscriptLine {
    text: string;
    hasNewline: boolean;
}

export const useTerminalTranscript = (sessionId: string) => {
    // Terminal Transcript Store
    const [terminalTranscripts, setTerminalTranscripts] = useState<Record<string, TranscriptLine[]>>({});

    // Ref to access latest transcripts in callbacks without stale closures
    const terminalTranscriptsRef = useRef<Record<string, TranscriptLine[]>>({});

    useEffect(() => {
        terminalTranscriptsRef.current = terminalTranscripts;
    }, [terminalTranscripts]);

    const appendToTranscript = (text: string, hasNewline: boolean = true) => {
        if (!sessionId) return;

        const line: TranscriptLine = { text, hasNewline };

        setTerminalTranscripts(prev => {
            const current = prev[sessionId] || [];
            return {
                ...prev,
                [sessionId]: [...current, line]
            };
        });
    };

    const getTranscript = (): TranscriptLine[] => {
        // Use ref to access latest state immediately
        return terminalTranscriptsRef.current[sessionId] || [];
    };

    const clearTranscript = () => {
        if (!sessionId) return;
        setTerminalTranscripts(prev => ({
            ...prev,
            [sessionId]: []
        }));
    };

    return {
        appendToTranscript,
        getTranscript,
        clearTranscript
    };
};
