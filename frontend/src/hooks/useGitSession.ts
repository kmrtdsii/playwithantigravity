import { useState, useCallback, useEffect } from 'react';
import { gitService } from '../services/gitService';

export interface GitSessionHook {
    sessionId: string;
    developers: string[];
    activeDeveloper: string;
    developerSessions: Record<string, string>;
    switchDeveloper: (name: string) => Promise<void>;
    addDeveloper: (name: string) => Promise<void>;
    removeDeveloper: (name: string) => Promise<void>;
    setSessionId: (id: string) => void;
}

export const useGitSession = (): GitSessionHook => {
    const [sessionId, setSessionId] = useState<string>('');
    const [developers, setDevelopers] = useState<string[]>([]);
    const [developerSessions, setDeveloperSessions] = useState<Record<string, string>>({});
    const [activeDeveloper, setActiveDeveloper] = useState<string>('');

    const switchDeveloper = useCallback(async (name: string) => {
        const sid = developerSessions[name];
        if (sid) {
            setActiveDeveloper(name);
            setSessionId(sid);
        }
    }, [developerSessions]);

    const addDeveloper = useCallback(async (name: string) => {
        try {
            if (developers.includes(name)) return;
            const data = await gitService.initSession();
            setDevelopers(prev => {
                if (prev.includes(name)) return prev;
                return [...prev, name];
            });
            setDeveloperSessions(prev => ({ ...prev, [name]: data.sessionId }));
            if (!activeDeveloper) {
                setActiveDeveloper(name);
                setSessionId(data.sessionId);
            }
        } catch (e) {
            console.error("Failed to add developer", e);
        }
    }, [developers, activeDeveloper]);

    const removeDeveloper = useCallback(async (name: string) => {
        if (name === 'Alice' || name === 'Bob') return;

        setDevelopers(prev => prev.filter(d => d !== name));

        if (activeDeveloper === name) {
            // Logic to switch to Alice
            // We assume Alice's session ID is available in current developerSessions state
            // or we optimistically update state.
            const aliceSid = developerSessions['Alice'];
            if (aliceSid) {
                setActiveDeveloper('Alice');
                setSessionId(aliceSid);
            }
        }

        const sid = developerSessions[name];
        if (sid) {
            setDeveloperSessions(prev => {
                const newSessions = { ...prev };
                delete newSessions[name];
                return newSessions;
            });
        }
    }, [activeDeveloper, developerSessions]);

    // Init Alice and Bob
    useEffect(() => {
        const init = async () => {
            await addDeveloper('Alice');
            await addDeveloper('Bob');
        };
        if (developers.length === 0) {
            init();
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    return {
        sessionId,
        developers,
        activeDeveloper,
        developerSessions,
        switchDeveloper,
        addDeveloper,
        removeDeveloper,
        setSessionId
    };
};
