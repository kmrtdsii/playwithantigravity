import React, { useEffect, useState } from 'react';
import { useMission, type VerificationResult } from '../../context/MissionContext';
import { motion, AnimatePresence } from 'framer-motion';
import { useTranslation } from 'react-i18next';

interface MissionInfo {
    id: string;
    title: string;
    description: string;
    hints: string[];
}

const MissionPanel: React.FC = () => {
    const { activeMissionId, endMission, verifyMission } = useMission();
    const { t, i18n } = useTranslation();
    const [verificationResult, setVerificationResult] = useState<VerificationResult | null>(null);
    const [missionInfo, setMissionInfo] = useState<MissionInfo | null>(null);

    useEffect(() => {
        if (activeMissionId) {
            // Fetch mission details
            fetch('/api/mission/list', {
                headers: { 'Accept-Language': i18n.language }
            })
                .then(res => res.json())
                .then((missions: MissionInfo[]) => {
        const mission = missions.find(m => m.id === activeMissionId);
        if (mission) setMissionInfo(mission);
    })
    .catch(console.error);
    } else {
    setMissionInfo(null);
    setVerificationResult(null);
}
}, [activeMissionId]);

if (!activeMissionId) return null;

const handleVerify = async () => {
    const result = await verifyMission();
    if (result) {
        setVerificationResult(result);
    }
};

return (
    <AnimatePresence>
        <motion.div
            initial={{ x: 300, opacity: 0 }}
            animate={{ x: 0, opacity: 1 }}
            exit={{ x: 300, opacity: 0 }}
            style={{
                position: 'absolute',
                top: 80, // Below header
                right: 20,
                width: 320,
                backgroundColor: 'var(--bg-secondary)',
                border: '1px solid var(--border-subtle)',
                borderRadius: 8,
                boxShadow: '0 4px 20px rgba(0,0,0,0.2)',
                zIndex: 50,
                padding: 16,
                display: 'flex',
                flexDirection: 'column',
                gap: 12
            }}
        >
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <h3 style={{ margin: 0, fontSize: 16, fontWeight: 700 }}>
                    🎯 {missionInfo ? missionInfo.title : t('mission.active')}
                </h3>
                <button onClick={endMission} style={{ background: 'none', border: 'none', cursor: 'pointer', fontSize: 16 }}>✕</button>
            </div>

            <div style={{ background: 'var(--bg-default)', padding: 12, borderRadius: 6, fontSize: 13, lineHeight: 1.5 }}>
                <p style={{ margin: '0 0 8px 0', fontWeight: 600 }}>{t('mission.objective')}:</p>
                <p style={{ margin: 0 }}>
                    {missionInfo?.description || t('mission.loading')}
                </p>
            </div>

            <div style={{ display: 'flex', gap: 8 }}>
                <button
                    onClick={handleVerify}
                    style={{
                        flex: 1,
                        padding: '8px 16px',
                        background: 'var(--accent-primary)',
                        color: 'white',
                        border: 'none',
                        borderRadius: 6,
                        cursor: 'pointer',
                        fontWeight: 600
                    }}
                >
                    {t('mission.verify')}
                </button>
                <button
                    onClick={endMission}
                    style={{
                        padding: '8px 12px',
                        background: 'var(--bg-button-inactive)',
                        color: 'var(--text-secondary)',
                        border: '1px solid var(--border-subtle)',
                        borderRadius: 6,
                        cursor: 'pointer'
                    }}
                >
                    {t('mission.abort')}
                </button>
            </div>

            {verificationResult && (
                <div style={{
                    marginTop: 8,
                    padding: 8,
                    background: verificationResult.success ? 'rgba(34, 197, 94, 0.1)' : 'rgba(239, 68, 68, 0.1)',
                    border: `1px solid ${verificationResult.success ? 'rgba(34, 197, 94, 0.3)' : 'rgba(239, 68, 68, 0.3)'}`,
                    borderRadius: 4,
                    color: verificationResult.success ? 'var(--success-fg)' : 'var(--error-fg)',
                    fontSize: 12
                }}>
                    {verificationResult.success ? (
                        <strong>🎉 {t('mission.success')}</strong>
                    ) : (
                        <div>
                            <strong>❌ {t('mission.failed')}:</strong>
                            <ul style={{ paddingLeft: 16, margin: '4px 0 0 0' }}>
                                {verificationResult.progress.filter(p => !p.passed).map((p, i) => (
                                    <li key={i}>{p.description}</li>
                                ))}
                            </ul>
                        </div>
                    )}
                </div>
            )}
        </motion.div>
    </AnimatePresence>
);
};

export default MissionPanel;
