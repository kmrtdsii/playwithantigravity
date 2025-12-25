import { useState, useRef, useEffect, useCallback } from 'react';
import { useGit } from '../context/GitAPIContext';
import type { CloneStatus } from '../components/layout/remote/CloneProgress';
import { gitService } from '../services/gitService';

export interface RepoInfo {
    name: string;
    sizeDisplay: string;
    message: string;
}

export const useRemoteClone = () => {
    const { ingestRemote, fetchServerState } = useGit();

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
        if (!url.trim()) return 'URLを入力してください';
        if (!url.startsWith('https://github.com')) return 'https://github.com で始まる必要があります';
        return null;
    };

    const performClone = useCallback(async (url: string) => {
        // Reset
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

            await ingestRemote('origin', url);
            await fetchServerState('origin');

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
                setErrorMessage(`無効なリポジトリです。URLが正しいか確認してください (${errorMsg})`);
            } else {
                setErrorMessage(errorMsg);
            }
            console.error('Clone failed:', err);
        }
    }, [ingestRemote, fetchServerState]);

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
        cancelClone
    };
};
