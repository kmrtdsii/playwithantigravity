import React, { useState, useEffect } from 'react';
import { GitPullRequest, CheckCircle, ChevronDown, ArrowLeft, GitMerge, Trash2, User, ArrowRight, RotateCw } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import Modal from '../../common/Modal';
import { Button } from '../../common/Button';
import type { PullRequest } from '../../../types/gitTypes';
import { sectionLabelStyle, actionButtonStyle, emptyStyle } from './remoteStyles';

interface PullRequestSectionProps {
    pullRequests: PullRequest[];
    branches: Record<string, string>;
    onCreatePR: (title: string, desc: string, source: string, target: string) => void;
    onMergePR: (id: number) => Promise<void>;
    onDeletePR: (id: number) => Promise<void>;
}

/**
 * Pull Request section with list and creation UI.
 */
const PullRequestSection: React.FC<PullRequestSectionProps> = ({
    pullRequests,
    branches,
    onCreatePR,
    onMergePR,
    onDeletePR,
}) => {
    const { t } = useTranslation('common');
    const [isCompareMode, setIsCompareMode] = useState(false);
    const [compareBase, setCompareBase] = useState('main');
    const [compareCompare, setCompareCompare] = useState('');

    // Delete Modal State
    const [deleteId, setDeleteId] = useState<number | null>(null);
    const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false);
    const [isDeletingPR, setIsDeletingPR] = useState(false);

    // Set default compare branch when branches load or change
    useEffect(() => {
        const branchNames = Object.keys(branches);
        if (branchNames.length > 0) {
            // 1. Validate 'compareBase'
            let newBase = compareBase;
            if (!branchNames.includes(compareBase)) {
                // If current base is invalid (e.g. stale from other repo), reset to default
                if (branchNames.includes('main')) newBase = 'main';
                else if (branchNames.includes('master')) newBase = 'master';
                else newBase = branchNames[0];
                // eslint-disable-next-line react-hooks/set-state-in-effect
                setCompareBase(newBase);
            }

            // 2. Validate 'compareCompare'
            // Must be valid AND different from base if possible
            if (!compareCompare || !branchNames.includes(compareCompare) || compareCompare === newBase) {
                // Try to find a different branch than newBase
                let candidate = branchNames.find(b => b !== newBase && b !== 'main' && b !== 'master');
                if (!candidate) {
                    candidate = branchNames.find(b => b !== newBase);
                }
                // If no other branch exists, just use the first available (even if same as base)
                if (!candidate) {
                    candidate = branchNames[0];
                }

                setCompareCompare(candidate);
            }
        }
    }, [branches, compareBase, compareCompare]);

    const handleCreatePRSubmit = (title: string) => {
        if (title) {
            onCreatePR(title, '', compareCompare, compareBase);
            setIsCompareMode(false);
        }
    };

    return (
        <div style={{ padding: '16px' }}>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '12px' }}>
                <div style={sectionLabelStyle}>{t('remote.pullRequests')}</div>
                {!isCompareMode && (
                    <button
                        type="button"
                        onClick={() => setIsCompareMode(true)}
                        style={{ ...actionButtonStyle, background: '#238636' }}
                    >
                        {t('remote.newPR')}
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
                <PullRequestList
                    pullRequests={pullRequests}
                    onMerge={onMergePR}
                    onDelete={(id) => {
                        setDeleteId(id);
                        setIsDeleteModalOpen(true);
                    }}
                />
            )}

            <Modal
                isOpen={isDeleteModalOpen}
                onClose={() => !isDeletingPR && setIsDeleteModalOpen(false)}
                title={t('remote.list.deletePRTitle')}
                hideCloseButton
            >
                <div style={{ padding: '8px 0' }}>
                    {t('remote.list.deletePRConfirm')}
                </div>
                <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '8px', marginTop: '16px' }}>
                    <Button
                        variant="ghost"
                        onClick={() => setIsDeleteModalOpen(false)}
                        disabled={isDeletingPR}
                    >
                        {t('remote.cancel')}
                    </Button>
                    <Button
                        variant="danger"
                        onClick={async () => {
                            if (deleteId) {
                                setIsDeletingPR(true);
                                await onDeletePR(deleteId);
                                setIsDeletingPR(false);
                                setIsDeleteModalOpen(false);
                                setDeleteId(null);
                            }
                        }}
                        isLoading={isDeletingPR}
                    >
                        {isDeletingPR ? t('remote.list.deleting') : t('remote.list.delete')}
                    </Button>
                </div>
            </Modal>
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
    onSubmit: (title: string) => void;
    onCancel: () => void;
}

// --- SUB-COMPONENTS: Custom Select (Premium Design) ---
interface CustomSelectProps {
    value: string;
    onChange: (value: string) => void;
    options: string[];
    label: string;
}

const CustomSelect: React.FC<CustomSelectProps> = ({ value, onChange, options, label }) => {
    const [isOpen, setIsOpen] = useState(false);
    const containerRef = React.useRef<HTMLDivElement>(null);

    // Close on click outside
    useEffect(() => {
        const handleClickOutside = (event: MouseEvent) => {
            if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
                setIsOpen(false);
            }
        };
        document.addEventListener('mousedown', handleClickOutside);
        return () => document.removeEventListener('mousedown', handleClickOutside);
    }, []);

    return (
        <div ref={containerRef} style={{ position: 'relative', width: '100%' }}>
            <div
                onClick={() => setIsOpen(!isOpen)}
                aria-label={label}
                role="button"
                tabIndex={0}
                style={{
                    width: '100%',
                    padding: '8px 12px',
                    paddingRight: '32px', // Space for chevron
                    background: 'var(--bg-primary)',
                    border: isOpen ? '1px solid var(--accent-primary)' : '1px solid var(--border-subtle)',
                    borderRadius: '6px',
                    color: 'var(--text-primary)',
                    fontSize: '13px',
                    cursor: 'pointer',
                    height: '36px',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    boxShadow: isOpen ? '0 0 0 2px rgba(var(--accent-rgb), 0.1)' : 'none',
                    transition: 'all 0.2s ease',
                    userSelect: 'none'
                }}
            >
                <div style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {value}
                </div>
                <ChevronDown
                    size={14}
                    style={{
                        position: 'absolute',
                        right: 12,
                        top: 11,
                        color: 'var(--text-tertiary)',
                        transform: isOpen ? 'rotate(180deg)' : 'none',
                        transition: 'transform 0.2s ease'
                    }}
                />
            </div>

            {isOpen && (
                <div style={{
                    position: 'absolute',
                    top: '100%',
                    left: 0,
                    right: 0,
                    marginTop: '4px',
                    background: 'var(--bg-secondary)', // Slightly lighter than primary for dropdown
                    border: '1px solid var(--border-subtle)',
                    borderRadius: '8px',
                    boxShadow: '0 4px 20px rgba(0, 0, 0, 0.2)',
                    zIndex: 100,
                    maxHeight: '200px',
                    overflowY: 'auto',
                    padding: '4px'
                }}>
                    {options.map(option => (
                        <div
                            key={option}
                            onClick={() => {
                                onChange(option);
                                setIsOpen(false);
                            }}
                            style={{
                                padding: '8px 12px',
                                fontSize: '13px',
                                color: option === value ? 'var(--text-primary)' : 'var(--text-secondary)',
                                borderRadius: '4px',
                                cursor: 'pointer',
                                background: option === value ? 'var(--bg-active)' : 'transparent',
                                display: 'flex',
                                alignItems: 'center',
                                justifyContent: 'space-between',
                                fontWeight: option === value ? 500 : 400
                            }}
                            onMouseEnter={(e) => {
                                if (option !== value) e.currentTarget.style.background = 'var(--bg-hover)';
                            }}
                            onMouseLeave={(e) => {
                                if (option !== value) e.currentTarget.style.background = 'transparent';
                            }}
                        >
                            <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{option}</span>
                            {option === value && <CheckCircle size={12} className="text-accent" />}
                        </div>
                    ))}
                </div>
            )}
        </div>
    );
};

// --- RENDER: Create PR Form (Design 2025) ---
const CompareView: React.FC<CompareViewProps> = ({
    branches,
    compareBase,
    compareCompare,
    onBaseChange,
    onCompareChange,
    onSubmit,
    onCancel,
}) => {
    const { t } = useTranslation('common');
    const branchNames = Object.keys(branches);
    const [title, setTitle] = useState(t('remote.compare.defaultTitle', { source: compareCompare, target: compareBase }));

    // Update default title when branches change
    useEffect(() => {
        // eslint-disable-next-line react-hooks/set-state-in-effect
        setTitle(t('remote.compare.defaultTitle', { source: compareCompare, target: compareBase }));
    }, [compareBase, compareCompare, t]);

    return (
        <div style={{
            background: 'var(--bg-secondary)',
            borderRadius: '12px',
            border: '1px solid var(--border-subtle)',
            padding: '20px', // Reduced padding closer to "Standard" size content
            marginTop: '16px',
            boxShadow: '0 4px 12px rgba(0, 0, 0, 0.08)'
        }}>
            {/* Header */}
            <div style={{ marginBottom: '16px' }}>
                <h3 style={{
                    margin: 0,
                    fontSize: '16px', // Slightly smaller for better fit
                    fontWeight: 600,
                    color: 'var(--text-primary)',
                    display: 'flex',
                    alignItems: 'center',
                    gap: '8px'
                }}>
                    <GitPullRequest size={18} className="text-accent" />
                    {t('remote.compare.title')}
                </h3>
                <p style={{
                    margin: '4px 0 0 0',
                    fontSize: '12px',
                    color: 'var(--text-tertiary)'
                }}>
                    {t('remote.compare.desc')}
                </p>
            </div>

            {/* Branch Selection (Visual Flow) */}
            <div style={{
                background: 'var(--bg-tertiary)',
                padding: '16px',
                borderRadius: '8px',
                border: '1px solid var(--border-subtle)',
                marginBottom: '16px'
            }}>
                <div style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: '12px',
                    flexWrap: 'wrap' // Allow wrapping on very small screens
                }}>
                    {/* Base */}
                    <div style={{ flex: '1 1 200px' }}> {/* Minimum width 200px */}
                        <label style={{
                            display: 'block',
                            fontSize: '11px',
                            fontWeight: 600,
                            color: 'var(--text-tertiary)',
                            marginBottom: '6px',
                            textTransform: 'uppercase',
                            letterSpacing: '0.05em'
                        }}>
                            {t('remote.compare.base')}
                        </label>
                        <CustomSelect
                            value={compareBase}
                            onChange={onBaseChange}
                            options={branchNames}
                            label={t('remote.compare.base')}
                        />
                    </div>

                    {/* Arrow Icon */}
                    <div style={{
                        color: 'var(--text-tertiary)',
                        paddingTop: '18px',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center'
                    }}>
                        <ArrowLeft size={18} strokeWidth={2.5} />
                    </div>

                    {/* Compare */}
                    <div style={{ flex: '1 1 200px' }}>
                        <label style={{
                            display: 'block',
                            fontSize: '11px',
                            fontWeight: 600,
                            color: 'var(--text-tertiary)',
                            marginBottom: '6px',
                            textTransform: 'uppercase',
                            letterSpacing: '0.05em'
                        }}>
                            {t('remote.compare.compare')}
                        </label>
                        <CustomSelect
                            value={compareCompare}
                            onChange={onCompareChange}
                            options={branchNames}
                            label={t('remote.compare.compare')}
                        />
                    </div>
                </div>
            </div>

            {/* Title Input */}
            <div style={{ marginBottom: '20px' }}>
                <input
                    value={title}
                    onChange={(e) => setTitle(e.target.value)}
                    placeholder={t('remote.compare.titlePlaceholder')}
                    style={{
                        width: '100%',
                        padding: '10px 14px',
                        background: 'var(--bg-primary)',
                        border: '1px solid var(--border-subtle)',
                        borderRadius: '6px',
                        color: 'var(--text-primary)',
                        fontSize: '14px',
                        outline: 'none',
                        transition: 'border-color 0.15s'
                    }}
                    onFocus={(e) => e.target.style.borderColor = 'var(--accent-primary)'}
                    onBlur={(e) => e.target.style.borderColor = 'var(--border-subtle)'}
                />
            </div>

            {/* Status & Actions - Responsive Footer */}
            <div style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                flexWrap: 'wrap', // Allow wrapping
                gap: '16px'
            }}>
                <div style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: '8px',
                    color: 'var(--text-success)',
                    fontSize: '13px',
                    whiteSpace: 'nowrap' // Keep status on one line if possible, otherwise it wraps as a block
                }}>
                    <CheckCircle size={16} />
                    <span style={{ fontWeight: 500 }}>{t('remote.compare.ableToMerge')}</span>
                    <span style={{ color: 'var(--text-tertiary)', fontSize: '12px', display: 'inline-block' }}>— {t('remote.compare.ableToMergeDesc')}</span>
                </div>

                <div style={{ display: 'flex', gap: '10px', marginLeft: 'auto' }}> {/* marginLeft auto pushes to right when wrapping */}
                    <button
                        type="button"
                        onClick={onCancel}
                        style={{
                            padding: '8px 16px',
                            background: 'transparent',
                            border: '1px solid transparent',
                            color: 'var(--text-secondary)',
                            fontWeight: 500,
                            fontSize: '13px',
                            cursor: 'pointer',
                            borderRadius: '6px'
                        }}
                    >
                        {t('remote.cancel')}
                    </button>
                    <button
                        type="button"
                        onClick={() => {
                            if (title.trim()) onSubmit(title);
                        }}
                        disabled={!title.trim()}
                        style={{
                            padding: '8px 20px',
                            background: title.trim() ? '#238636' : 'var(--bg-button-inactive)',
                            color: 'white',
                            border: 'none',
                            borderRadius: '6px',
                            fontWeight: 600,
                            fontSize: '13px',
                            cursor: title.trim() ? 'pointer' : 'not-allowed',
                            display: 'flex',
                            alignItems: 'center',
                            gap: '8px',
                            boxShadow: title.trim() ? '0 2px 4px rgba(0,0,0,0.1)' : 'none',
                            whiteSpace: 'nowrap'
                        }}
                    >
                        <GitPullRequest size={14} />
                        {t('remote.compare.create')}
                    </button>
                </div>
            </div>
        </div>
    );
};


interface PullRequestListProps {
    pullRequests: PullRequest[];
    onMerge: (id: number) => Promise<void>;
    onDelete: (id: number) => void;
}

const PullRequestList: React.FC<PullRequestListProps> = ({ pullRequests, onMerge, onDelete }) => {
    const { t } = useTranslation('common');
    return (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
            {pullRequests.length === 0 ? (
                <div style={emptyStyle}>{t('remote.list.empty')}</div>
            ) : (
                pullRequests.map(pr => (
                    <PullRequestCard key={pr.id} pr={pr} onMerge={() => onMerge(pr.id)} onDelete={() => onDelete(pr.id)} />
                ))
            )}
        </div>
    );
};

interface PullRequestCardProps {
    pr: PullRequest;
    onMerge: () => Promise<void>;
    onDelete: () => void;
}

const PullRequestCard: React.FC<PullRequestCardProps> = ({ pr, onMerge, onDelete }) => {
    const { t } = useTranslation('common');
    const [isMerging, setIsMerging] = useState(false);
    const [mergeError, setMergeError] = useState<string | null>(null);
    const [isHovered, setIsHovered] = useState(false);

    const handleMergeClick = async () => {
        setIsMerging(true);
        setMergeError(null);
        try {
            await onMerge();
        } catch (error) {
            console.error('Merge failed:', error);
            const msg = error instanceof Error ? error.message : 'Unknown error';
            setMergeError(msg);
        } finally {
            setIsMerging(false);
        }
    };

    return (
        <div
            style={{
                background: 'var(--bg-secondary)',
                borderRadius: '8px',
                border: '1px solid var(--border-subtle)',
                padding: '16px',
                transition: 'all 0.2s ease',
                boxShadow: isHovered ? '0 4px 12px rgba(0,0,0,0.08)' : 'none',
                transform: isHovered ? 'translateY(-1px)' : 'none',
                position: 'relative'
            }}
            onMouseEnter={() => setIsHovered(true)}
            onMouseLeave={() => setIsHovered(false)}
        >
            {/* Header: Status & Title */}
            <div style={{ display: 'flex', alignItems: 'flex-start', gap: '12px', marginBottom: '12px' }}>
                <div style={{
                    marginTop: '2px',
                    color: pr.status === 'OPEN' ? '#238636' : '#8957e5'
                }}>
                    <GitPullRequest size={18} />
                </div>
                <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{
                        fontSize: '15px',
                        fontWeight: 600,
                        color: 'var(--text-primary)',
                        marginBottom: '4px',
                        lineHeight: '1.4'
                    }}>
                        {pr.title}
                        <span style={{
                            fontWeight: 400,
                            color: 'var(--text-tertiary)',
                            marginLeft: '8px',
                            fontSize: '14px'
                        }}>
                            #{pr.id}
                        </span>
                    </div>
                </div>
                <div>
                    <span style={{
                        fontSize: '11px',
                        padding: '2px 8px',
                        background: pr.status === 'OPEN' ? '#238636' : '#8957e5',
                        color: 'white',
                        borderRadius: '12px',
                        fontWeight: 600,
                        letterSpacing: '0.02em'
                    }}>
                        {pr.status}
                    </span>
                </div>
            </div>

            {/* Meta Info */}
            <div style={{
                display: 'flex',
                alignItems: 'center',
                flexWrap: 'wrap',
                gap: '16px',
                fontSize: '12px',
                color: 'var(--text-secondary)',
                marginBottom: '16px',
                paddingBottom: '16px',
                borderBottom: '1px solid var(--border-subtle)'
            }}>
                {/* Branches */}
                <div style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: '6px',
                    background: 'var(--bg-tertiary)',
                    padding: '4px 8px',
                    borderRadius: '4px',
                    fontFamily: 'monospace'
                }}>
                    <span style={{ color: 'var(--accent-primary)' }}>{pr.sourceBranch}</span>
                    <ArrowRight size={12} style={{ color: 'var(--text-tertiary)' }} />
                    <span style={{ fontWeight: 600 }}>{pr.targetBranch}</span>
                </div>

                {/* User */}
                <div style={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
                    <User size={12} style={{ color: 'var(--text-tertiary)' }} />
                    <span>{t('remote.list.openedBy', { name: pr.creator })}</span>
                </div>
            </div>

            {/* Error Message */}
            {mergeError && (
                <div style={{
                    marginBottom: '16px',
                    padding: '10px 12px',
                    background: 'rgba(207, 34, 46, 0.1)',
                    border: '1px solid rgba(207, 34, 46, 0.2)',
                    borderRadius: '6px',
                    color: '#cf222e',
                    fontSize: '12px',
                    display: 'flex',
                    alignItems: 'center',
                    gap: '8px'
                }}>
                    <div style={{ flexShrink: 0 }}>⚠️</div>
                    <div>{mergeError}</div>
                    <button
                        onClick={() => setMergeError(null)}
                        style={{ marginLeft: 'auto', background: 'none', border: 'none', color: '#cf222e', cursor: 'pointer', fontSize: '14px' }}
                    >
                        ×
                    </button>
                </div>
            )}

            {/* Actions */}
            <div style={{ display: 'flex', gap: '8px', justifyContent: 'flex-end' }}>
                <button
                    type="button"
                    onClick={() => onDelete()}
                    disabled={isMerging}
                    style={{
                        padding: '8px 12px',
                        background: 'transparent',
                        color: 'var(--text-tertiary)',
                        border: '1px solid transparent', // Invisible border for alignment
                        borderRadius: '6px',
                        fontSize: '13px',
                        fontWeight: 500,
                        cursor: 'pointer',
                        display: 'flex',
                        alignItems: 'center',
                        gap: '6px',
                        transition: 'all 0.15s'
                    }}
                    onMouseEnter={(e) => {
                        e.currentTarget.style.color = '#cf222e';
                        e.currentTarget.style.background = 'rgba(207, 34, 46, 0.05)';
                    }}
                    onMouseLeave={(e) => {
                        e.currentTarget.style.color = 'var(--text-tertiary)';
                        e.currentTarget.style.background = 'transparent';
                    }}
                >
                    <Trash2 size={14} />
                    {t('remote.list.delete')}
                </button>

                {pr.status === 'OPEN' && (
                    <button
                        type="button"
                        onClick={handleMergeClick}
                        disabled={isMerging}
                        style={{
                            padding: '8px 20px',
                            background: isMerging ? 'var(--bg-button-inactive)' : '#8957e5', // Purple for merge
                            color: 'white',
                            border: 'none',
                            borderRadius: '6px',
                            fontSize: '13px',
                            fontWeight: 600,
                            cursor: isMerging ? 'not-allowed' : 'pointer',
                            display: 'flex',
                            alignItems: 'center',
                            gap: '8px',
                            boxShadow: isMerging ? 'none' : '0 2px 4px rgba(137, 87, 229, 0.2)',
                            transition: 'all 0.15s'
                        }}
                        onMouseEnter={(e) => !isMerging && (e.currentTarget.style.background = '#7a4bc4')} // Darker purple
                        onMouseLeave={(e) => !isMerging && (e.currentTarget.style.background = '#8957e5')}
                    >
                        {isMerging ? (
                            <RotateCw size={14} className="spin" />
                        ) : (
                            <GitMerge size={14} />
                        )}
                        {isMerging ? t('remote.list.merging') : t('remote.list.merge')}
                    </button>
                )}
            </div>
        </div>
    );
};

export default PullRequestSection;
