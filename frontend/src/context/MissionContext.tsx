import React, { createContext, useContext, useState } from 'react';
import { useGit } from './GitAPIContext';



export interface CheckResult {
    description: string;
    passed: boolean;
}

export interface VerificationResult {
    success: boolean;
    missionId: string;
    progress: CheckResult[];
    message?: string; // For errors or summary
}

interface MissionContextType {
    activeMissionId: string | null;
    startMission: (missionId: string) => Promise<void>;
    endMission: () => void;
    verifyMission: () => Promise<VerificationResult | undefined>;
}

const MissionContext = createContext<MissionContextType | undefined>(undefined);

export const MissionProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
    const { sessionId, setSessionId } = useGit();
    const [activeMissionId, setActiveMissionId] = useState<string | null>(null);
    const [originalSessionId, setOriginalSessionId] = useState<string | null>(null);

    const startMission = async (missionId: string) => {
        // Save current session
        if (!activeMissionId && sessionId) {
            setOriginalSessionId(sessionId);
        }

        try {
            const res = await fetch('/api/mission/start', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ missionId }),
            });
            if (!res.ok) throw new Error(await res.text());

            const data = await res.json();
            setActiveMissionId(missionId);
            // Switch Global Git Context to Mission Session
            setSessionId(data.sessionId);
        } catch (e) {
            console.error("Failed to start mission", e);
            alert("Failed to start mission: " + e);
        }
    };

    const endMission = () => {
        setActiveMissionId(null);
        if (originalSessionId) {
            setSessionId(originalSessionId);
            setOriginalSessionId(null);
        } else {
            // Fallback if original lost
            // initSession(); 
            // Logic depends on how persistent session is handled
        }
    };

    const verifyMission = async () => {
        if (!activeMissionId || !sessionId) return;
        const res = await fetch('/api/mission/verify', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ sessionId, missionId: activeMissionId }),
        });
        return await res.json() as VerificationResult;
    };

    return (
        <MissionContext.Provider value={{ activeMissionId, startMission, endMission, verifyMission }}>
            {children}
        </MissionContext.Provider>
    );
};

export const useMission = () => {
    const context = useContext(MissionContext);
    if (!context) throw new Error('useMission must be used within MissionProvider');
    return context;
};
