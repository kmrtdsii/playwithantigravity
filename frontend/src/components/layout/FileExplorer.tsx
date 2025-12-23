import { useState, useMemo } from 'react';
import { useGit } from '../../context/GitAPIContext';
import { GitBranch, Check } from 'lucide-react';
import type { SelectedObject } from '../../types/layoutTypes';

interface FileExplorerProps {
    onSelect: (obj: SelectedObject) => void;
}

interface FileNode {
    name: string;
    path: string;
    isDir: boolean;
    children?: FileNode[];
    status?: string; // XY status from git
}

const FileExplorer: React.FC<FileExplorerProps> = ({ onSelect }) => {
    const { state, runCommand } = useGit();
    const [expandedFolders, setExpandedFolders] = useState<Set<string>>(new Set());
    const [showBranches, setShowBranches] = useState(true);

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

    // Build Entry Tree from flat file list (Working Tree)
    const fileTree = useMemo(() => {
        if (isRoot) return [];

        const root: FileNode = { name: 'ROOT', path: '', isDir: true, children: [] };
        const allFiles = state.files || [];
        allFiles.forEach(filePath => {
            const parts = filePath.split('/');
            let currentLevel = root.children!;
            parts.forEach((part, index) => {
                if (!part) return;
                const isLast = index === parts.length - 1;
                const currentPath = parts.slice(0, index + 1).join('/');

                let existingNode = currentLevel.find(n => n.name === part);
                if (!existingNode) {
                    const newNode: FileNode = {
                        name: part,
                        path: currentPath,
                        isDir: !isLast || filePath.endsWith('/'),
                        children: (!isLast || filePath.endsWith('/')) ? [] : undefined,
                        status: (isLast && !filePath.endsWith('/')) ? state.fileStatuses[filePath] : undefined
                    };
                    currentLevel.push(newNode);
                    currentLevel.sort((a, b) => {
                        if (a.isDir && !b.isDir) return -1;
                        if (!a.isDir && b.isDir) return 1;
                        return a.name.localeCompare(b.name);
                    });
                    existingNode = newNode;
                }
                if (existingNode.isDir) {
                    currentLevel = existingNode.children!;
                }
            });
        });
        return root.children || [];
    }, [state.files, state.fileStatuses, isRoot]);

    const toggleFolder = (path: string, e: React.MouseEvent) => {
        e.stopPropagation();
        const newSet = new Set(expandedFolders);
        if (newSet.has(path)) newSet.delete(path);
        else newSet.add(path);
        setExpandedFolders(newSet);
    };

    const handleProjectClick = (projectName: string) => {
        if (activeProject === projectName) {
            runCommand(`cd /`);
        } else {
            runCommand(`cd /${projectName}`);
        }
    };

    const handleBranchClick = (branchName: string, e: React.MouseEvent) => {
        e.stopPropagation();
        if (branchName !== currentBranch) {
            runCommand(`git checkout ${branchName}`);
        }
    };

    const handleDirClick = (node: FileNode, e: React.MouseEvent) => {
        e.stopPropagation();
        toggleFolder(node.path, e);
    };

    const handleDeleteProject = (projectName: string, e: React.MouseEvent) => {
        e.stopPropagation();
        if (confirm(`Are you sure you want to delete project '${projectName}'? This cannot be undone.`)) {
            if (activeProject === projectName) {
                runCommand('cd /');
            }
            runCommand(`rm -rf ${projectName}`);
        }
    };

    const getStatusColor = (status?: string) => {
        if (!status) return 'var(--text-secondary)';
        if (status.includes('?')) return '#ff5f56';
        if (status.includes('M')) return '#ffbd2e';
        if (status.includes('A')) return '#27c93f';
        return 'var(--text-primary)';
    };

    const renderTree = (nodes: FileNode[], depth: number = 0) => {
        return nodes.map(node => {
            const isExpanded = expandedFolders.has(node.path);
            const paddingLeft = depth * 16 + 12;

            if (node.isDir) {
                return (
                    <div key={node.path}>
                        <div
                            className="explorer-row"
                            onClick={(e) => handleDirClick(node, e)}
                            style={{ paddingLeft }}
                        >
                            <span className="icon">{isExpanded ? '📂' : '📁'}</span>
                            <span className="name">{node.name}</span>
                        </div>
                        {isExpanded && node.children && (
                            <div>{renderTree(node.children, depth + 1)}</div>
                        )}
                    </div>
                );
            } else {
                return (
                    <div
                        key={node.path}
                        className="explorer-row"
                        onClick={() => onSelect({ type: 'file', id: node.path })}
                        style={{ paddingLeft }}
                    >
                        <span className="icon">📄</span>
                        <span className="name" style={{ color: getStatusColor(node.status) }}>{node.name}</span>
                        {node.status && <span className="status-badge">{node.status}</span>}
                    </div>
                );
            }
        });
    };

    const projects = state.projects || [];

    return (
        <div className="file-explorer" style={{ color: 'var(--text-primary)', fontSize: '13px', fontFamily: 'system-ui, sans-serif', userSelect: 'none', display: 'flex', width: '100%', height: '100%' }}>

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
                    WORKSPACES
                </div>

                <div className="tree-content" style={{ flex: 1, overflowY: 'auto' }}>
                    {projects.length === 0 ? (
                        <div style={{ padding: '12px', color: 'var(--text-tertiary)', fontStyle: 'italic' }}>
                            No projects found. Clone a repo!
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
                                            <span className="name">{project}</span>
                                            <span className="delete-btn" onClick={(e) => handleDeleteProject(project, e)} style={{ marginLeft: 'auto', cursor: 'pointer', opacity: 0.5 }}>🗑️</span>
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
                                                            <span>BRANCHES</span>
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
                                                                                fontWeight: isCurrent ? 600 : 400,
                                                                                cursor: isCurrent ? 'default' : 'pointer'
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

                                                {/* File Tree */}
                                                {renderTree(fileTree)}
                                            </div>
                                        )}
                                    </div>
                                );
                            })}
                        </div>
                    )}
                </div>
            </div>

            <style>{`
                .explorer-row { display: flex; align-items: center; padding-top: 5px; padding-bottom: 5px; cursor: pointer; border-radius: 4px; }
                .explorer-row:hover { background-color: var(--bg-button-inactive); }
                .branch-row:hover { background-color: var(--bg-tertiary); }
                .icon { margin-right: 6px; opacity: 0.9; }
                .name { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
                .status-badge { font-size: 10px; opacity: 0.7; margin-right: 8px; font-family: monospace; }
            `}</style>
        </div>
    );
};

export default FileExplorer;

