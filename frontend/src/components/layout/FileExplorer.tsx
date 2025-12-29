import { useState, useMemo, useEffect } from 'react';
import { useTranslation, Trans } from 'react-i18next';
import { useGit } from '../../context/GitAPIContext';
import { Folder, FileCode } from 'lucide-react';
import type { SelectedObject } from '../../types/layoutTypes';
import Modal from '../common/Modal';
import { Button } from '../common/Button';
import WorkspaceTree from '../workspace/WorkspaceTree';

import { ChevronRight, ChevronDown } from 'lucide-react';
interface FileExplorerProps {
    onSelect: (obj: SelectedObject) => void;
}

// Tree Node Structure
// Tree Node Structure
interface TreeNode {
    name: string;
    path: string;
    isDir: boolean;
    children: Record<string, TreeNode>;
    status?: string; // 'M', '??', 'A', 'MM', etc.
    hasChanges?: boolean; // True if self or children have status
}

const buildTree = (files: string[], statuses: Record<string, string>): TreeNode => {
    const root: TreeNode = { name: 'root', path: '', isDir: true, children: {} };

    // Helper to calculate hasChanges recursively
    const computeChanges = (node: TreeNode): boolean => {
        let changed = false;

        // Check self status (if file)
        if (!node.isDir && node.status && node.status.trim() !== '') {
            // Only count changes if they are visual (Untracked or Worktree Modified)
            const status = node.status;
            // Untracked
            if (status === '??') changed = true;
            // Worktree Modified (2nd char)
            else if (status.length > 1 && status[1] !== ' ' && status[1] !== '?') changed = true;
        }

        // Check children
        Object.values(node.children).forEach(child => {
            if (computeChanges(child)) {
                changed = true;
            }
        });

        node.hasChanges = changed;
        return changed;
    };


    files.forEach(filePath => {
        const parts = filePath.split('/');
        let current = root;

        parts.forEach((part, index) => {
            if (!part) return;
            const currentPath = parts.slice(0, index + 1).join('/');

            if (!current.children[part]) {
                current.children[part] = {
                    name: part,
                    path: currentPath,
                    isDir: false,
                    children: {}
                };
            }

            // If we are not at the leaf, this node MUST be a dir
            if (index < parts.length - 1) {
                current.children[part].isDir = true;
                current = current.children[part];
            } else {
                // Leaf node: Assign status
                current.children[part].status = statuses[filePath];
            }
        });
    });

    computeChanges(root);
    return root;
};

// ... (TreeItem)
const TreeItem: React.FC<{ node: TreeNode, depth: number }> = ({ node, depth }) => {
    const [isOpen, setIsOpen] = useState(false);

    const hasChildren = Object.keys(node.children).length > 0;
    const isDir = node.isDir || hasChildren;

    // Use sorted children: Dirs first, then Files
    const sortedChildren = useMemo(() => {
        return Object.values(node.children).sort((a, b) => {
            if (a.isDir === b.isDir) return a.name.localeCompare(b.name);
            return a.isDir ? -1 : 1;
        });
    }, [node.children]);

    const handleToggle = (e: React.MouseEvent) => {
        e.stopPropagation();
        setIsOpen(!isOpen);
    };

    // --- Status Logic ---
    // Status is usually 2 chars: [Index, Worktree] e.g "MM", "A ", " M", "??"
    // We explicitly IGNORE index status (char 0) for the file explorer visualization as per requirements.
    // We only care about Worktree status (char 1) or Untracked ('??').

    let textColor = 'var(--text-secondary)';
    const badges: { label: string, color: string }[] = [];

    const status = node.status || '';
    // const indexStatus = status.length > 0 ? status[0] : ' '; // Ignored
    const worktreeStatus = status.length > 1 ? status[1] : ' ';

    // 1. Untracked
    if (status === '??') {
        textColor = '#73c990'; // Green
        badges.push({ label: 'U', color: '#73c990' });
    } else {
        // 2. Worktree Status (Modified)
        // Only show if modified in worktree. Staged 'A' or 'M' (indexStatus) are ignored here.
        if (worktreeStatus !== ' ' && worktreeStatus !== '?') {
            const color = '#e2c08d'; // Light Orange
            badges.push({ label: worktreeStatus, color });
            textColor = color;
        }
    }

    // Is Folder modified?
    // Note: computeChanges (buildTree) has proven to only set hasChanges=true for Untracked/WorktreeModified.
    const showDot = isDir && node.hasChanges;

    return (
        <div>
            <div
                className="explorer-row"
                onClick={isDir ? handleToggle : undefined}
                style={{
                    padding: `4px 12px 4px ${12 + depth * 12}px`, // Indentation
                    cursor: isDir ? 'pointer' : 'default',
                    display: 'flex',
                    alignItems: 'center',
                    color: !isDir ? textColor : 'var(--text-secondary)'
                }}
            >
                {/* Indent Icon */}
                <span style={{ width: '16px', display: 'flex', justifyContent: 'center', marginRight: '4px' }}>
                    {isDir && (
                        isOpen ? <ChevronDown size={12} /> : <ChevronRight size={12} />
                    )}
                </span>

                <span className="icon" style={{ display: 'flex', alignItems: 'center', marginRight: '6px' }}>
                    {isDir ?
                        <Folder size={14} style={{ color: showDot ? '#e2c08d' : '#60a5fa' }} /> :
                        <FileCode size={14} style={{ color: !isDir ? textColor : '#fbbf24' }} />
                    }
                </span>

                <span className="name" style={{ fontSize: '12px', flex: '0 1 auto', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {node.name}
                </span>

                {/* Right Aligned Area */}
                <div style={{ marginLeft: 'auto', display: 'flex', alignItems: 'center', paddingLeft: '8px' }}>

                    {/* Folder Dot */}
                    {showDot && (
                        <span style={{
                            width: '8px', height: '8px',
                            borderRadius: '50%',
                            backgroundColor: '#e2c08d', // Dot Color
                            marginRight: '4px'
                        }} />
                    )}

                    {/* File Badges */}
                    {!isDir && badges.map((b, i) => (
                        <span
                            key={i}
                            className="status-badge"
                            style={{
                                color: b.color,
                                marginLeft: '4px',
                                minWidth: '14px',
                                textAlign: 'center'
                            }}
                        >
                            {b.label}
                        </span>
                    ))}
                </div>
            </div>

            {isOpen && hasChildren && (
                <div>
                    {sortedChildren.map(child => (
                        <TreeItem key={child.path} node={child} depth={depth + 1} />
                    ))}
                </div>
            )}
        </div>
    );
};


const FileExplorer: React.FC<FileExplorerProps> = () => {
    const { t } = useTranslation('common');
    const {
        state,
        runCommand,
        sessionId,
        workspaceTree,
        currentRepo,
        fetchWorkspaceTree
    } = useGit();

    // Modal State (for delete confirmation)
    const [projectToDelete, setProjectToDelete] = useState<string | null>(null);
    const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false);

    // Fetch tree on mount and when state changes
    useEffect(() => {
        if (sessionId) {
            fetchWorkspaceTree(sessionId);
        }
    }, [fetchWorkspaceTree, sessionId, state.commandCount]);

    // Active Project Detection (for delete logic)
    const currentPathClean = state.currentPath ? state.currentPath.replace(/^\//, '') : '';
    const isRoot = !currentPathClean;
    const activeProject = isRoot ? null : currentPathClean.split('/')[0];

    const confirmDelete = async () => {
        if (!projectToDelete) return;

        if (activeProject === projectToDelete) {
            await runCommand('cd /', { silent: true });
        }
        await runCommand(`rm -rf /${projectToDelete}`, { silent: true });

        setIsDeleteModalOpen(false);
        setProjectToDelete(null);
    };

    const handleUpDir = () => {
        runCommand('cd ..');
    };

    const handleNavigate = async (path: string) => {
        await runCommand(`cd ${path}`, { silent: true });
        if (sessionId) {
            await fetchWorkspaceTree(sessionId);
        }
    };

    const files = state.files || [];
    const fileStatuses = state.fileStatuses || {};

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
                    <WorkspaceTree
                        tree={workspaceTree}
                        currentPath={state.currentPath || '/'}
                        currentRepo={currentRepo}
                        onNavigate={handleNavigate}
                    />
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
                    {t('workspace.files')}
                </div>

                <div className="tree-content" style={{ flex: 1, overflowY: 'auto' }}>

                    {/* UP DIR ENTRY */}
                    {!isRoot && (
                        <div
                            className="explorer-row"
                            onClick={handleUpDir}
                            style={{ padding: '4px 24px' }}
                        >
                            <span className="icon" style={{ display: 'flex', alignItems: 'center' }}>
                                <Folder size={14} style={{ color: 'var(--text-secondary)' }} />
                            </span>
                            <span className="name" style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>..</span>
                        </div>
                    )}

                    {files.length === 0 ? (
                        <div style={{ padding: '12px', color: 'var(--text-tertiary)', fontStyle: 'italic' }}>
                            {t('workspace.noFiles')}
                        </div>
                    ) : (
                        <div style={{ paddingBottom: '20px' }}>
                            {(() => {
                                const root = buildTree(files, fileStatuses);
                                const children = Object.values(root.children).sort((a, b) => {
                                    if (a.isDir === b.isDir) return a.name.localeCompare(b.name);
                                    return a.isDir ? -1 : 1;
                                });

                                return children.map(child => (
                                    <TreeItem key={child.path} node={child} depth={0} />
                                ));
                            })()}
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
                .status-badge { font-size: 10px; font-weight: 700; opacity: 0.9; margin-left: 4px; font-family: monospace; }
                .delete-btn:hover { opacity: 1 !important; color: var(--text-danger); transform: scale(1.1); transition: all 0.2s; }
            `}</style>
        </div>
    );
};

export default FileExplorer;

