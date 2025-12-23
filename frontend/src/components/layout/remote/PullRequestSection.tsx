import React, { useState, useEffect } from 'react';
import type { PullRequest } from '../../../types/gitTypes';
import { sectionLabelStyle, actionButtonStyle, prCardStyle, mergeButtonStyle, emptyStyle } from './remoteStyles';

interface PullRequestSectionProps {
    pullRequests: PullRequest[];
    branches: Record<string, string>;
    onCreatePR: (title: string, desc: string, source: string, target: string) => void;
    onMergePR: (id: number) => void;
}

/**
 * Pull Request section with list and creation UI.
 */
const PullRequestSection: React.FC<PullRequestSectionProps> = ({
    pullRequests,
    branches,
    onCreatePR,
    onMergePR,
}) => {
    const [isCompareMode, setIsCompareMode] = useState(false);
    const [compareBase, setCompareBase] = useState('main');
    const [compareCompare, setCompareCompare] = useState('');

    // Set default compare branch when branches load
    useEffect(() => {
        const branchNames = Object.keys(branches);
        if (branchNames.length > 0) {
            if (!branchNames.includes(compareBase)) setCompareBase(branchNames[0]);
            if (!compareCompare && branchNames.length > 1) {
                setCompareCompare(branchNames.find(b => b !== 'main') || branchNames[1]);
            } else if (!compareCompare) {
                setCompareCompare(branchNames[0]);
            }
        }
    }, [branches, compareBase, compareCompare]);

    const handleCreatePRSubmit = () => {
        const title = prompt('PR Title', `Merge ${compareCompare} into ${compareBase}`);
        if (title) {
            onCreatePR(title, '', compareCompare, compareBase);
            setIsCompareMode(false);
        }
    };

    return (
        <div style={{ padding: '16px' }}>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '12px' }}>
                <div style={sectionLabelStyle}>PULL REQUESTS</div>
                {!isCompareMode && (
                    <button
                        onClick={() => setIsCompareMode(true)}
                        style={{ ...actionButtonStyle, background: '#238636' }}
                    >
                        New Pull Request
                    </button>
                )}
            </div>

            {isCompareMode ? (
                <CompareView
                    branches={branches}
                    compareBase={compareBase}
                    compareCompare={compareCompare}
                    onBaseChange={setCompareBase}
                    onCompareChange={setCompareCompare}
                    onSubmit={handleCreatePRSubmit}
                    onCancel={() => setIsCompareMode(false)}
                />
            ) : (
                <PullRequestList pullRequests={pullRequests} onMerge={onMergePR} />
            )}
        </div>
    );
};

// --- Sub-components ---

interface CompareViewProps {
    branches: Record<string, string>;
    compareBase: string;
    compareCompare: string;
    onBaseChange: (value: string) => void;
    onCompareChange: (value: string) => void;
    onSubmit: () => void;
    onCancel: () => void;
}

const CompareView: React.FC<CompareViewProps> = ({
    branches,
    compareBase,
    compareCompare,
    onBaseChange,
    onCompareChange,
    onSubmit,
    onCancel,
}) => {
    const branchNames = Object.keys(branches);

    return (
        <div style={{
            background: 'var(--bg-secondary)',
            borderRadius: '6px',
            border: '1px solid var(--border-subtle)',
            marginBottom: '16px',
            overflow: 'hidden'
        }}>
            <div style={{
                padding: '12px',
                borderBottom: '1px solid var(--border-subtle)',
                background: 'var(--bg-primary)'
            }}>
                <div style={{ fontSize: '1.2rem', fontWeight: 600, marginBottom: '4px' }}>
                    Comparing changes
                </div>
                <div style={{ fontSize: '0.8rem', color: 'var(--text-secondary)' }}>
                    Choose two branches to see what's changed or to start a new pull request.
                </div>
            </div>

            <div style={{
                padding: '12px',
                display: 'flex',
                alignItems: 'center',
                gap: '8px',
                background: 'var(--bg-secondary)',
                borderBottom: '1px solid var(--border-subtle)'
            }}>
                <BranchSelector label="base" value={compareBase} onChange={onBaseChange} branches={branchNames} />
                <span style={{ color: 'var(--text-tertiary)' }}>←</span>
                <BranchSelector label="compare" value={compareCompare} onChange={onCompareChange} branches={branchNames} />
            </div>

            <div style={{
                padding: '12px',
                background: '#e6ffec',
                color: '#1a7f37',
                fontSize: '0.85rem',
                display: 'flex',
                alignItems: 'center',
                gap: '6px',
                borderBottom: '1px solid var(--border-subtle)'
            }}>
                <span>✓</span>
                <strong>Able to merge.</strong>
                <span>These branches can be automatically merged.</span>
            </div>

            <div style={{ padding: '12px', display: 'flex', justifyContent: 'flex-end', gap: '8px' }}>
                <button
                    onClick={onCancel}
                    style={{
                        padding: '6px 12px',
                        background: 'transparent',
                        border: 'none',
                        color: 'var(--text-secondary)',
                        cursor: 'pointer'
                    }}
                >
                    Cancel
                </button>
                <button
                    onClick={onSubmit}
                    style={{ ...actionButtonStyle, background: '#238636', fontSize: '0.9rem', padding: '6px 16px' }}
                >
                    Create pull request
                </button>
            </div>
        </div>
    );
};

interface BranchSelectorProps {
    label: string;
    value: string;
    onChange: (value: string) => void;
    branches: string[];
}

const BranchSelector: React.FC<BranchSelectorProps> = ({ label, value, onChange, branches }) => (
    <div style={{ display: 'flex', alignItems: 'center', gap: '4px', fontSize: '0.85rem' }}>
        <span style={{ color: 'var(--text-tertiary)' }}>{label}:</span>
        <select
            value={value}
            onChange={e => onChange(e.target.value)}
            style={{
                background: 'var(--bg-primary)',
                color: 'var(--text-primary)',
                border: '1px solid var(--border-subtle)',
                borderRadius: '6px',
                padding: '4px 8px'
            }}
        >
            {branches.map(b => <option key={b} value={b}>{b}</option>)}
        </select>
    </div>
);

interface PullRequestListProps {
    pullRequests: PullRequest[];
    onMerge: (id: number) => void;
}

const PullRequestList: React.FC<PullRequestListProps> = ({ pullRequests, onMerge }) => (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
        {pullRequests.length === 0 ? (
            <div style={emptyStyle}>No active PRs</div>
        ) : (
            pullRequests.map(pr => (
                <PullRequestCard key={pr.id} pr={pr} onMerge={() => onMerge(pr.id)} />
            ))
        )}
    </div>
);

interface PullRequestCardProps {
    pr: PullRequest;
    onMerge: () => void;
}

const PullRequestCard: React.FC<PullRequestCardProps> = ({ pr, onMerge }) => (
    <div style={prCardStyle}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
            <div style={{ fontWeight: 700, fontSize: '0.85rem' }}>
                #{pr.id} {pr.title}
            </div>
            <span style={{
                fontSize: '0.7rem',
                padding: '2px 6px',
                background: pr.status === 'OPEN' ? '#238636' : '#8957e5',
                color: 'white',
                borderRadius: '10px'
            }}>
                {pr.status}
            </span>
        </div>
        <div style={{ fontSize: '0.75rem', color: 'var(--text-secondary)' }}>
            {pr.sourceBranch} ➜ {pr.targetBranch}
        </div>
        <div style={{ fontSize: '0.7rem', color: 'var(--text-tertiary)' }}>
            opened by {pr.creator}
        </div>
        {pr.status === 'OPEN' && (
            <button onClick={onMerge} style={mergeButtonStyle}>
                Merge Pull Request
            </button>
        )}
    </div>
);

export default PullRequestSection;
