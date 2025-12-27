import React, { useEffect, useState } from 'react';
import { useGit } from '../../context/GitAPIContext';
import { FileCode, FilePlus, FileMinus, FileDiff, X } from 'lucide-react';

interface CommitDetailsProps {
    commitId: string;
    onClose: () => void;
}

interface FileChange {
    status: 'A' | 'M' | 'D' | 'R';
    path: string;
}

const CommitDetails: React.FC<CommitDetailsProps> = ({ commitId, onClose }) => {
    const { runCommand } = useGit();
    // const { t } = useTranslation('common');
    const [changes, setChanges] = useState<FileChange[]>([]);
    // const [commitInfo, setCommitInfo] = useState<any>(null); // To store basic info if needed
    const [loading, setLoading] = useState(false);

    useEffect(() => {
        const fetchDetails = async () => {
            if (!commitId) return;
            setLoading(true);
            try {
                // Fetch changes: git show <id> --name-status
                // We use our custom backend command logic
                const output = await runCommand(`git show ${commitId} --name-status`, { silent: true, skipRefresh: true });

                if (output) {
                    // output might be string or string[] depending on context, assuming string based on runCommand usage generally
                    // But error said it's string[]? Let's handle both.
                    const outputStr = Array.isArray(output) ? output.join('\n') : output;
                    const lines = outputStr.trim().split('\n');
                    const parsedChanges: FileChange[] = [];

                    lines.forEach((line: string) => {
                        const parts = line.split('\t');
                        if (parts.length >= 2) {
                            parsedChanges.push({
                                status: parts[0][0] as any,
                                path: parts[1]
                            });
                        }
                    });
                    setChanges(parsedChanges);
                }
            } catch (e) {
                console.error("Failed to fetch commit details", e);
            } finally {
                setLoading(false);
            }
        };

        fetchDetails();
    }, [commitId, runCommand]);

    const getIcon = (status: string) => {
        switch (status) {
            case 'A': return <FilePlus size={14} className="text-green-500" style={{ color: '#4ade80' }} />;
            case 'D': return <FileMinus size={14} className="text-red-500" style={{ color: '#f87171' }} />;
            case 'M': return <FileDiff size={14} className="text-yellow-500" style={{ color: '#fbbf24' }} />;
            default: return <FileCode size={14} className="text-gray-500" style={{ color: '#9ca3af' }} />;
        }
    };

    return (
        <div style={{
            display: 'flex',
            flexDirection: 'column',
            height: '100%',
            background: 'var(--bg-secondary)',
            borderLeft: '1px solid var(--border-subtle)'
        }}>
            {/* Header */}
            <div style={{
                padding: '12px',
                borderBottom: '1px solid var(--border-subtle)',
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center',
                background: 'var(--bg-tertiary)'
            }}>
                <div style={{ fontWeight: 600, fontSize: '12px' }}>
                    Commit Details
                </div>
                <button onClick={onClose} style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'var(--text-secondary)' }}>
                    <X size={14} />
                </button>
            </div>

            {/* Content */}
            <div style={{ flex: 1, overflowY: 'auto', padding: '12px' }}>
                <div style={{ marginBottom: '16px' }}>
                    <div style={{ fontSize: '14px', fontWeight: 'bold', fontFamily: 'monospace', color: 'var(--accent-primary)' }}>
                        {commitId.substring(0, 7)}
                    </div>
                </div>

                <div style={{ fontSize: '11px', fontWeight: 600, color: 'var(--text-tertiary)', marginBottom: '8px', textTransform: 'uppercase' }}>
                    Changed Files ({changes.length})
                </div>

                {loading ? (
                    <div style={{ padding: '8px', color: 'var(--text-tertiary)' }}>Loading...</div>
                ) : (
                    <div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
                        {changes.map((change, idx) => (
                            <div key={idx} style={{
                                display: 'flex',
                                alignItems: 'center',
                                padding: '4px 8px',
                                background: 'var(--bg-primary)',
                                borderRadius: '4px',
                                fontSize: '12px',
                                border: '1px solid var(--border-subtle)'
                            }}>
                                <span style={{ marginRight: '8px', display: 'flex' }}>
                                    {getIcon(change.status)}
                                </span>
                                <span style={{
                                    whiteSpace: 'nowrap',
                                    overflow: 'hidden',
                                    textOverflow: 'ellipsis',
                                    color: change.status === 'D' ? 'var(--text-tertiary)' : 'var(--text-primary)',
                                    textDecoration: change.status === 'D' ? 'line-through' : 'none'
                                }}>
                                    {change.path}
                                </span>
                                <span style={{ marginLeft: 'auto', fontSize: '10px', color: 'var(--text-tertiary)', fontFamily: 'monospace', opacity: 0.7 }}>
                                    {change.status}
                                </span>
                            </div>
                        ))}
                    </div>
                )}
            </div>
        </div>
    );
};

export default CommitDetails;
