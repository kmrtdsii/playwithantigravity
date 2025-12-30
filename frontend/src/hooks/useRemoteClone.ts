import { useState, useRef, useEffect, useCallback } from 'react';
import { useGit } from '../context/GitAPIContext';
import { useTranslation } from 'react-i18next';
import type { CloneStatus } from '../components/layout/remote/CloneProgress';
import { gitService } from '../services/gitService';

export interface RepoInfo {
    name: string;
    sizeDisplay: string;
    message: string;
}

export const useRemoteClone = () => {
    const { ingestRemote, fetchServerState, sessionId } = useGit();
    const { t } = useTranslation('common');

    const [cloneStatus, setCloneStatus] = useState<CloneStatus>('idle');
    const [estimatedSeconds, setEstimatedSeconds] = useState<number>(0);
    const [elapsedSeconds, setElapsedSeconds] = useState(0);
    const [repoInfo, setRepoInfo] = useState<RepoInfo | undefined>(undefined);
    const [errorMessage, setErrorMessage] = useState<string | undefined>(undefined);

    // Timer ref
    const timerRef = useRef<number | null>(null);

    // Cleanup timer on unmount
    useEffect(() => {
        return () => {
            if (timerRef.current) {
                window.clearInterval(timerRef.current);
            }
        };
    }, []);

    const validateUrl = (url: string): string | null => {
        if (!url.trim()) return t('remote.validation.urlRequired');
        // Accept https:// URLs and git@ URLs (SSH)
        if (!url.startsWith('https://') && !url.startsWith('git@')) {
            return t('remote.validation.invalidUrl');
        }
        return null;
    };

    const performClone = useCallback(async (url: string, depth?: number) => {
        // Clear any existing timer first
        if (timerRef.current) {
            clearInterval(timerRef.current);
            timerRef.current = null;
        }

        // Reset state
        setErrorMessage(undefined);
        setElapsedSeconds(0);
        setEstimatedSeconds(0);
        setRepoInfo(undefined);

        const validationError = validateUrl(url);
        if (validationError) {
            setCloneStatus('error');
            setErrorMessage(validationError);
            return;
        }

        try {
            setCloneStatus('fetching_info');

            const info = await gitService.getRemoteInfo(url);
            setRepoInfo({
                name: info.repoInfo.name,
                sizeDisplay: info.sizeDisplay,
                message: info.message,
            });
            setEstimatedSeconds(info.estimatedSeconds);

            setCloneStatus('cloning');

            const startTime = Date.now();
            timerRef.current = window.setInterval(() => {
                setElapsedSeconds((Date.now() - startTime) / 1000);
            }, 500);

            // Derive a friendly name from the URL for the list
            // e.g. https://github.com/user/repo.git -> repo
            // e.g. https://github.com/user/repo/ -> repo
            let remoteName = 'origin';
            try {
                // Remove trailing slash if present
                const cleanUrl = url.replace(/\/$/, '');
                const parts = cleanUrl.split('/');
                const lastPart = parts[parts.length - 1];
                // Remove .git extension if present
                const candidate = lastPart.replace(/\.git$/, '');
                if (candidate && candidate.trim() !== '') {
                    remoteName = candidate;
                }
            } catch (e) {
                console.warn('Failed to parse remote name from URL', e);
            }

            console.log('Ingesting remote with name:', remoteName, 'URL:', url);

            // Use derived name instead of 'origin' so it shows up in the backend list (which filters out 'origin')
            await ingestRemote(remoteName, url, depth);
            await fetchServerState(remoteName);

            if (timerRef.current) {
                clearInterval(timerRef.current);
                timerRef.current = null;
            }
            setCloneStatus('complete');

            // Auto reset
            setTimeout(() => setCloneStatus('idle'), 2000);

        } catch (err) {
            if (timerRef.current) {
                clearInterval(timerRef.current);
                timerRef.current = null;
            }
            setCloneStatus('error');

            const errorMsg = err instanceof Error ? err.message : 'Unknown error';
            if (errorMsg.includes('404') || errorMsg.includes('not found') || errorMsg.toLowerCase().includes('failed')) {
                setErrorMessage(t('remote.status.invalidRepo', { error: errorMsg }));
            } else {
                setErrorMessage(errorMsg);
            }
            console.error('Clone failed:', err);
        }
    }, [ingestRemote, fetchServerState]);

    const performCreate = useCallback(async (name: string) => {
        // Reset state
        setErrorMessage(undefined);
        setElapsedSeconds(0);
        setEstimatedSeconds(0);
        setRepoInfo(undefined);

        if (!name.trim()) {
            setCloneStatus('error');
            setErrorMessage(t('remote.empty.repoNameRequired'));
            return;
        }

        try {
            setCloneStatus('creating');
            // Mock timer for UX
            const startTime = Date.now();
            timerRef.current = window.setInterval(() => {
                setElapsedSeconds((Date.now() - startTime) / 1000);
            }, 100);

            await gitService.createRemote(name, sessionId);

            // Fetch state to reflect the directory switch on backend
            // We use the actual repo name because 'origin' alias is not set in SharedRemotes for created repos
            await fetchServerState(name);

            if (timerRef.current) {
                clearInterval(timerRef.current);
                timerRef.current = null;
            }
            setCloneStatus('complete');

            // Auto reset
            setTimeout(() => setCloneStatus('idle'), 2000);

        } catch (err) {
            if (timerRef.current) {
                clearInterval(timerRef.current);
                timerRef.current = null;
            }
            setCloneStatus('error');
            const msg = err instanceof Error ? err.message : 'Failed to create repository';
            // Translate generic fetch errors to something friendlier
            if (msg.includes('Failed to fetch') || msg.includes('Network request failed')) {
                setErrorMessage(t('remote.status.serverError'));
            } else {
                setErrorMessage(msg);
            }
        }
    }, [fetchServerState, sessionId]);


    const cancelClone = useCallback(() => {
        if (timerRef.current) {
            clearInterval(timerRef.current);
            timerRef.current = null;
        }
        setCloneStatus('idle');
        setErrorMessage(undefined);
    }, []);

    return {
        cloneStatus,
        setCloneStatus, // Exposed for external reset if needed
        estimatedSeconds,
        elapsedSeconds,
        repoInfo,
        errorMessage,
        performClone,
        performCreate,
        cancelClone
    };
};
