import React from 'react';
import { useGit } from '../../context/GitAPIContext';
import { useTranslation } from 'react-i18next'; // Add import
import { FileCode } from 'lucide-react';

export const StagedFilesPanel: React.FC = () => {
    const { t } = useTranslation('common'); // Hook
    const { state } = useGit();
    const stagedFiles = state.staging || [];
    const statuses = state.fileStatuses || {};

    if (stagedFiles.length === 0) return null;

    return (
        <div data-testid="staged-files-panel" style={{
            width: '250px',
            display: 'flex',
            flexDirection: 'column',
            borderLeft: '1px solid var(--border-subtle)',
            background: 'var(--bg-secondary)',
            height: '100%',
            overflow: 'hidden'
        }}>
            <div className="section-header" style={{
                height: 'var(--header-height)',
                display: 'flex',
                alignItems: 'center',
                padding: '0 12px',
                fontSize: '11px',
                fontWeight: 700,
                color: 'var(--text-secondary)',
                borderBottom: '1px solid var(--border-subtle)',
                backgroundColor: 'var(--bg-tertiary)',
                textTransform: 'uppercase',
                letterSpacing: '0.05em'
            }}>
                {t('workspace.stagedChanges')}
                <span style={{
                    marginLeft: '8px',
                    backgroundColor: 'var(--accent-primary)',
                    color: 'white',
                    borderRadius: '10px',
                    padding: '1px 6px',
                    fontSize: '10px',
                    fontWeight: 600
                }}>
                    {stagedFiles.length}
                </span>
            </div>

            <div style={{ flex: 1, overflowY: 'auto' }}>
                {stagedFiles.map(file => {
                    // Status logic
                    const fullStatus = statuses[file] || '  ';
                    const indexStatus = fullStatus[0]; // 'A', 'M', 'D', 'R'

                    let badgeColor = 'var(--text-secondary)';
                    const label = indexStatus;
                    let title = 'Modified';

                    switch (indexStatus) {
                        case 'A':
                            badgeColor = '#60a5fa'; // Blue
                            title = 'Added';
                            break;
                        case 'M':
                            badgeColor = '#facc15'; // Yellow
                            title = 'Modified';
                            break;
                        case 'D':
                            badgeColor = '#f87171'; // Red
                            title = 'Deleted';
                            break;
                        case 'R':
                            badgeColor = '#c084fc'; // Purple
                            title = 'Renamed';
                            break;
                    }

                    return (
                        <div key={file} style={{
                            padding: '6px 12px',
                            display: 'flex',
                            alignItems: 'center',
                            fontSize: '12px',
                            borderBottom: '1px solid var(--border-subtle)',
                            cursor: 'default',
                            backgroundColor: 'var(--bg-primary)'
                        }} title={`${title}: ${file}`}>
                            <FileCode size={14} style={{ marginRight: '8px', opacity: 0.7, flexShrink: 0 }} />

                            <span style={{
                                flex: 1,
                                overflow: 'hidden',
                                textOverflow: 'ellipsis',
                                whiteSpace: 'nowrap',
                                marginRight: '8px'
                            }}>
                                {file.split('/').pop()}
                            </span>

                            <span style={{
                                color: badgeColor,
                                fontWeight: 'bold',
                                fontFamily: 'monospace',
                                fontSize: '11px',
                                flexShrink: 0
                            }}>
                                {label}
                            </span>
                        </div>
                    );
                })}
            </div>
        </div>
    );
};
