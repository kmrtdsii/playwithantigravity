import React, { createContext, useContext, useRef, useCallback, useState } from 'react';

interface TerminalOutputContextType {
    getOutput: (sessionId: string) => string[];
    addOutput: (sessionId: string, lines: string[]) => void;
    clearOutput: (sessionId: string) => void;
    // Version counter to trigger re-renders when output changes
    version: number;
}

const TerminalOutputContext = createContext<TerminalOutputContextType | undefined>(undefined);

export const TerminalOutputProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
    // Use ref to store outputs - avoids recreating getOutput on every state change
    const outputsRef = useRef<Record<string, string[]>>({});
    // Version counter to notify consumers of changes
    const [version, setVersion] = useState(0);

    const addOutput = useCallback((sessionId: string, lines: string[]) => {
        const current = outputsRef.current[sessionId] || [];
        outputsRef.current = {
            ...outputsRef.current,
            [sessionId]: [...current, ...lines]
        };
        // Increment version to trigger re-renders in consumers
        setVersion(v => v + 1);
    }, []);

    const clearOutput = useCallback((sessionId: string) => {
        outputsRef.current = {
            ...outputsRef.current,
            [sessionId]: []
        };
        setVersion(v => v + 1);
    }, []);

    // Stable reference - doesn't change when outputs change
    const getOutput = useCallback((sessionId: string) => {
        return outputsRef.current[sessionId] || [];
    }, []);

    // Memoize the context value to prevent unnecessary provider re-renders
    const contextValue = React.useMemo(() => ({
        getOutput,
        addOutput,
        clearOutput,
        version
    }), [getOutput, addOutput, clearOutput, version]);

    return (
        <TerminalOutputContext.Provider value={contextValue}>
            {children}
        </TerminalOutputContext.Provider>
    );
};

export const useTerminalOutput = () => {
    const context = useContext(TerminalOutputContext);
    if (!context) {
        throw new Error('useTerminalOutput must be used within a TerminalOutputProvider');
    }
    return context;
};
