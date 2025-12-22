import { useState, useMemo } from 'react';
import { useGit } from '../../context/GitAPIContext';
import type { SelectedObject } from './AppLayout';

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

    // Active Project Detection
    const currentPathClean = state.currentPath ? state.currentPath.replace(/^\//, '') : '';
    const isRoot = !currentPathClean;
    const activeProject = isRoot ? null : currentPathClean.split('/')[0];

    // Compute Staged Files
    const stagedFiles = useMemo(() => {
        if (!state.HEAD || state.HEAD.type === 'none') return [];
        return Object.entries(state.fileStatuses || {})
            .filter(([, status]) => {
                const x = status[0];
                return x !== ' ' && x !== '?';
            })
            .map(([path, status]) => ({ path, status }));
    }, [state.fileStatuses, state.HEAD]);

    // Build Entry Tree from flat file list (Working Tree)
    // For this split view, the LEFT side is the general file explorer (which implicitly shows unstaged changes via status colors).
    const fileTree = useMemo(() => {
        if (isRoot) return []; // Don't show files at root, relying on projects list

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

            {/* LEFT PANE: CHANGES & EXPLORER */}
            <div style={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column', borderRight: '1px solid var(--border-subtle)' }}>
                <div className="section-header" style={{ padding: '8px 12px', fontSize: '11px', fontWeight: 'bold', color: 'var(--text-tertiary)', background: 'var(--bg-secondary)', borderBottom: '1px solid var(--border-subtle)' }}>
                    {isRoot ? 'WORKSPACES' : 'FILES / CHANGES'}
                </div>

                <div className="tree-content" style={{ flex: 1, overflowY: 'auto' }}>
                    {projects.length === 0 && isRoot ? (
                        <div style={{ padding: '12px', color: 'var(--text-tertiary)', fontStyle: 'italic' }}>
                            No projects found. Clone a repo!
                        </div>
                    ) : (
                        <div>
                            {/* If Root: Show Projects */}
                            {isRoot && projects.map(project => {
                                const isActive = project === activeProject;
                                return (
                                    <div key={project} className="explorer-row" onClick={() => handleProjectClick(project)} style={{ padding: '4px 12px' }}>
                                        <span className="icon">📦</span>
                                        <span className="name" style={{ fontWeight: isActive ? 'bold' : 'normal' }}>{project}</span>
                                        <span className="delete-btn" onClick={(e) => handleDeleteProject(project, e)} style={{ marginLeft: 'auto', cursor: 'pointer' }}>🗑️</span>
                                    </div>
                                );
                            })}

                            {/* If In Project: Show File Tree */}
                            {!isRoot && (
                                renderTree(fileTree)
                            )}
                        </div>
                    )}
                </div>
            </div>

            {/* RIGHT PANE: STAGED */}
            <div style={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column', background: 'rgba(0,0,0,0.02)' }}>
                <div className="section-header" style={{ padding: '8px 12px', fontSize: '11px', fontWeight: 'bold', color: 'var(--text-tertiary)', background: 'var(--bg-secondary)', borderBottom: '1px solid var(--border-subtle)', display: 'flex', justifyContent: 'space-between' }}>
                    <span>STAGED CHANGES</span>
                    <span style={{ fontWeight: 'normal', opacity: 0.7 }}>{stagedFiles.length}</span>
                </div>
                <div className="tree-content" style={{ flex: 1, overflowY: 'auto' }}>
                    {stagedFiles.length === 0 ? (
                        <div style={{ padding: '20px', textAlign: 'center', color: 'var(--text-tertiary)', fontStyle: 'italic', fontSize: '12px' }}>
                            No staged changes.
                        </div>
                    ) : (
                        stagedFiles.map((file) => (
                            <div key={file.path} className="explorer-row" onClick={() => onSelect({ type: 'file', id: file.path })} style={{ padding: '4px 12px' }}>
                                <span className="icon">📄</span>
                                <span className="name" style={{ color: '#27c93f' }}>{file.path}</span>
                                <span className="status-badge" style={{ marginLeft: 'auto' }}>{file.status}</span>
                            </div>
                        ))
                    )}
                </div>
            </div>

            <style>{`
                .explorer-row { display: flex; alignItems: center; padding-top: 5px; padding-bottom: 5px; cursor: pointer; border-radius: 4px; }
                .explorer-row:hover { background-color: rgba(255, 255, 255, 0.05); }
                .icon { margin-right: 6px; opacity: 0.9; }
                .name { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
                .status-badge { font-size: 10px; opacity: 0.7; margin-right: 8px; font-family: monospace; }
            `}</style>
        </div>
    );
};

export default FileExplorer;
