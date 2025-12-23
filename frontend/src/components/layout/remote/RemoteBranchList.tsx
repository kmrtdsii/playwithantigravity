import React from 'react';
import { sectionLabelStyle, emptyStyle } from './remoteStyles';

interface RemoteBranchListProps {
    remoteBranches: Record<string, string>;
}

/**
 * Displays the list of remote branches with their commit hashes.
 */
const RemoteBranchList: React.FC<RemoteBranchListProps> = ({ remoteBranches }) => {
    const branches = Object.entries(remoteBranches);

    return (
        <div style={{ padding: '0 16px 16px 16px' }}>
            <div style={{ ...sectionLabelStyle, marginBottom: '12px' }}>REMOTE BRANCHES</div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
                {branches.length === 0 ? (
                    <div style={emptyStyle}>No branches found</div>
                ) : (
                    branches.map(([name, hash]) => (
                        <div
                            key={name}
                            style={{
                                display: 'flex',
                                justifyContent: 'space-between',
                                padding: '6px 10px',
                                background: 'var(--bg-secondary)',
                                borderRadius: '6px',
                                fontSize: '0.8rem',
                                border: '1px solid var(--border-subtle)'
                            }}
                        >
                            <span style={{ fontFamily: 'monospace' }}>{name}</span>
                            <span style={{
                                color: 'var(--text-tertiary)',
                                fontSize: '0.7rem',
                                fontFamily: 'monospace'
                            }}>
                                {hash.substring(0, 7)}
                            </span>
                        </div>
                    ))
                )}
            </div>
        </div>
    );
};

export default RemoteBranchList;
