import { useEffect } from 'react';
import { useGit } from '../context/GitAPIContext';
import type { CloneStatus } from '../components/layout/remote/CloneProgress';

interface UseAutoDiscoveryProps {
    setupUrl: string;
    setSetupUrl: (url: string) => void;
    cloneStatus: CloneStatus;
    performClone: (url: string) => void;
}

export const useAutoDiscovery = ({
    setupUrl,
    setSetupUrl,
    cloneStatus,
    performClone
}: UseAutoDiscoveryProps) => {
    const { state, serverState } = useGit();

    useEffect(() => {
        const localOrigin = state.remotes?.find(r => r.name === 'origin');
        if (localOrigin && localOrigin.urls.length > 0) {
            const detectedUrl = localOrigin.urls[0];

            // Skip internal URLs (created repositories) - they don't need cloning
            if (detectedUrl.startsWith('remote://')) {
                return;
            }

            // Only auto-configure if disconnected, no setup url manually typed, and not currently cloning
            if (!serverState && !setupUrl && cloneStatus === 'idle') {
                console.log('Auto-detected remote origin:', detectedUrl);
                setSetupUrl(detectedUrl);
                performClone(detectedUrl);
            }
        }
    }, [state.remotes, serverState, setupUrl, cloneStatus, performClone, setSetupUrl]);
};
