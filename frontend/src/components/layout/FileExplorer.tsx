import { useState, useMemo } from 'react';
import { useTranslation, Trans } from 'react-i18next';
import { useGit } from '../../context/GitAPIContext';
import { GitBranch, Check, Folder, FileCode } from 'lucide-react';
import type { SelectedObject } from '../../types/layoutTypes';
import Modal from '../common/Modal';
import { Button } from '../common/Button';

interface FileExplorerProps {
    onSelect: (obj: SelectedObject) => void;
}


const FileExplorer: React.FC<FileExplorerProps> = () => {
    const { t } = useTranslation('common');
    const { state, runCommand } = useGit();
    const [showBranches, setShowBranches] = useState(true);

    // Modal State
    const [projectToDelete, setProjectToDelete] = useState<string | null>(null);
    const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false);

    // Active Project Detection
    const currentPathClean = state.currentPath ? state.currentPath.replace(/^\//, '') : '';
    const isRoot = !currentPathClean;
    const activeProject = isRoot ? null : currentPathClean.split('/')[0];

    // Get current branch from HEAD
    const currentBranch = state.HEAD?.ref || null;

    // Get local branches
    const localBranches = useMemo(() => {
        return Object.keys(state.branches || {}).sort();
    }, [state.branches]);

    const handleProjectClick = (projectName: string) => {
        if (activeProject === projectName) {
            runCommand(`cd /`, { silent: true });
        } else {
            runCommand(`cd /${projectName}`, { silent: true });
        }
    };

    const handleBranchClick = (branchName: string, e: React.MouseEvent) => {
        e.stopPropagation();
        if (branchName !== currentBranch) {
            runCommand(`git checkout ${branchName}`);
        }
    };

    const handleDeleteClick = (projectName: string, e: React.MouseEvent) => {
        e.stopPropagation();
        setProjectToDelete(projectName);
        setIsDeleteModalOpen(true);
    };

    const confirmDelete = async () => {
        if (!projectToDelete) return;

        if (activeProject === projectToDelete) {
            await runCommand('cd /', { silent: true });
        }
        await runCommand(`rm -rf /${projectToDelete}`, { silent: true });

        setIsDeleteModalOpen(false);
        setProjectToDelete(null);
    };

    const projects = state.projects || [];
    const files = state.files || [];

    return (
        <div className="file-explorer" data-testid="file-explorer" style={{ color: 'var(--text-primary)', fontSize: '13px', fontFamily: 'system-ui, sans-serif', userSelect: 'none', display: 'flex', width: '100%', height: '100%' }}>

            {/* SINGLE PANE: WORKSPACES & FILES */}
            <div style={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column' }}>
                <div className="section-header" style={{
                    height: 'var(--header-height)',
                    display: 'flex',
                    alignItems: 'center',
                    padding: '0 var(--space-3)',
                    fontSize: 'var(--text-xs)',
                    fontWeight: 'var(--font-extrabold)',
                    color: 'var(--text-secondary)',
                    background: 'var(--bg-secondary)',
                    borderBottom: '1px solid var(--border-subtle)',
                    letterSpacing: '0.05em',
                    flexShrink: 0
                }}>
                    {t('workspace.title')}
                </div>

                <div className="tree-content" style={{ flex: 1, overflowY: 'auto' }}>
                    {projects.length === 0 ? (
                        <div style={{ padding: '12px', color: 'var(--text-tertiary)', fontStyle: 'italic' }}>
                            {t('workspace.noProjects')}
                        </div>
                    ) : (
                        <div>
                            {projects.map(project => {
                                const isActive = project === activeProject;
                                return (
                                    <div key={project}>
                                        {/* Project Row */}
                                        <div
                                            className="explorer-row"
                                            onClick={() => handleProjectClick(project)}
                                            style={{
                                                padding: '4px 12px',
                                                background: isActive ? 'var(--bg-button-inactive)' : 'transparent',
                                                fontWeight: isActive ? 600 : 400
                                            }}
                                        >
                                            <span className="icon">{isActive ? '📂' : '📦'}</span>
                                            <span className="name" style={{ marginRight: '8px', flex: '0 1 auto' }}>{project}</span>

                                            {/* ACTIVE BRANCH BADGE */}
                                            {isActive && currentBranch && (
                                                <div style={{
                                                    display: 'flex',
                                                    alignItems: 'center',
                                                    padding: '2px 8px',
                                                    backgroundColor: 'rgba(74, 222, 128, 0.15)',
                                                    border: '1px solid rgba(74, 222, 128, 0.3)',
                                                    borderRadius: '12px',
                                                    color: 'var(--accent-primary)',
                                                    fontSize: '10px',
                                                    fontFamily: 'monospace',
                                                    whiteSpace: 'nowrap',
                                                    flexShrink: 0
                                                }}>
                                                    <GitBranch size={10} style={{ marginRight: '4px' }} />
                                                    <span style={{ fontWeight: 600 }}>{currentBranch}</span>
                                                </div>
                                            )}

                                            {/* Spacer to push delete button to right */}
                                            <div style={{ flex: 1 }} />

                                            <span
                                                className="delete-btn"
                                                onClick={(e) => handleDeleteClick(project, e)}
                                                style={{ marginLeft: '8px', cursor: 'pointer', opacity: 0.5 }}
                                                title={t('workspace.deleteTitle')}
                                            >
                                                🗑️
                                            </span>
                                        </div>

                                        {/* Expanded Content (Only if active) */}
                                        {isActive && (
                                            <div style={{ marginLeft: '0px' }}>
                                                {/* Branches Section */}
                                                {localBranches.length > 0 && (
                                                    <div style={{ borderBottom: '1px solid var(--border-subtle)', marginBottom: '4px' }}>
                                                        <div
                                                            className="explorer-row"
                                                            onClick={() => setShowBranches(!showBranches)}
                                                            style={{ padding: '4px 24px', fontSize: '11px', fontWeight: 600, color: 'var(--text-secondary)' }}
                                                        >
                                                            <GitBranch size={12} style={{ marginRight: '6px', opacity: 0.8 }} />
                                                            <span>{t('workspace.branches')}</span>
                                                            <span style={{ marginLeft: 'auto', fontSize: '10px', opacity: 0.6 }}>{showBranches ? '▼' : '▶'}</span>
                                                        </div>
                                                        {showBranches && (
                                                            <div style={{ paddingBottom: '4px' }}>
                                                                {localBranches.map(branch => {
                                                                    const isCurrent = branch === currentBranch;
                                                                    return (
                                                                        <div
                                                                            key={branch}
                                                                            className="explorer-row branch-row"
                                                                            onClick={(e) => handleBranchClick(branch, e)}
                                                                            style={{
                                                                                padding: '3px 36px',
                                                                                color: isCurrent ? 'var(--accent-primary)' : 'var(--text-secondary)',
                                                                                fontWeight: isCurrent ? 700 : 400,
                                                                                backgroundColor: isCurrent ? 'rgba(74, 222, 128, 0.1)' : 'transparent', // var(--accent-primary) with opacity
                                                                                cursor: isCurrent ? 'default' : 'pointer',
                                                                                borderRadius: '4px',
                                                                                margin: '0 4px'
                                                                            }}
                                                                            title={isCurrent ? 'Current branch' : `Checkout ${branch}`}
                                                                        >
                                                                            {isCurrent && <Check size={12} style={{ marginRight: '4px', color: 'var(--accent-primary)' }} />}
                                                                            <span>{branch}</span>
                                                                        </div>
                                                                    );
                                                                })}
                                                            </div>
                                                        )}
                                                    </div>
                                                )}
                                            </div>
                                        )}
                                    </div>
                                );
                            })}
                        </div>
                    )}
                </div>
            </div>

            {/* SEPARATOR */}
            <div style={{ height: '1px', background: 'var(--border-subtle)' }} />

            {/* PANE 2: FILES */}
            <div style={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column' }}>
                <div className="section-header" style={{
                    height: 'var(--header-height)',
                    display: 'flex',
                    alignItems: 'center',
                    padding: '0 var(--space-3)',
                    fontSize: 'var(--text-xs)',
                    fontWeight: 'var(--font-extrabold)',
                    color: 'var(--text-secondary)',
                    background: 'var(--bg-secondary)',
                    borderBottom: '1px solid var(--border-subtle)',
                    letterSpacing: '0.05em',
                    flexShrink: 0
                }}>
                    FILES
                </div>

                <div className="tree-content" style={{ flex: 1, overflowY: 'auto' }}>
                    {files.length === 0 ? (
                        <div style={{ padding: '12px', color: 'var(--text-tertiary)', fontStyle: 'italic' }}>
                            {t('workspace.noFiles')}
                        </div>
                    ) : (
                        <div>
                            {files.map(file => {
                                const isDir = file.endsWith('/');
                                // Clean name for display (remove trailing slash for dirs)
                                const displayName = isDir ? file.slice(0, -1) : file;

                                return (
                                    <div
                                        key={file}
                                        className="explorer-row"
                                        style={{
                                            padding: '4px 24px', // Matches indented branch padding roughly
                                        }}
                                    >
                                        <span className="icon" style={{ display: 'flex', alignItems: 'center' }}>
                                            {isDir ?
                                                <Folder size={14} style={{ color: '#60a5fa' }} /> : // blue-400
                                                <FileCode size={14} style={{ color: '#fbbf24' }} /> // amber-400 (using FileCode as generic code icon for visual pop)
                                            }
                                        </span>
                                        <span className="name" style={{ fontSize: '12px' }}>{displayName}</span>
                                    </div>
                                );
                            })}
                        </div>
                    )}
                </div>
            </div>

            <Modal
                isOpen={isDeleteModalOpen}
                onClose={() => setIsDeleteModalOpen(false)}
                title={t('workspace.deleteTitle')}
            >
                <div>
                    <Trans t={t} i18nKey="workspace.deleteConfirm" values={{ name: projectToDelete }} components={{ 1: <strong></strong> }} />
                </div>
                <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '8px', marginTop: '16px' }}>
                    <Button variant="ghost" onClick={() => setIsDeleteModalOpen(false)}>
                        {t('remote.cancel')}
                    </Button>
                    <Button variant="danger" onClick={confirmDelete}>
                        {t('workspace.deleteTitle').split(' ')[0]}
                    </Button>
                </div>
            </Modal>

            <style>{`
                .explorer-row { display: flex; align-items: center; padding-top: 5px; padding-bottom: 5px; cursor: pointer; border-radius: 4px; }
                .explorer-row:hover { background-color: var(--bg-button-inactive); }
                .branch-row:hover { background-color: var(--bg-tertiary); }
                .icon { margin-right: 6px; opacity: 0.9; }
                .name { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
                .status-badge { font-size: 10px; opacity: 0.7; margin-right: 8px; font-family: monospace; }
                .delete-btn:hover { opacity: 1 !important; color: var(--text-danger); transform: scale(1.1); transition: all 0.2s; }
            `}</style>
        </div>
    );
};

export default FileExplorer;

