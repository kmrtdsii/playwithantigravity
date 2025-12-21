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
    const [expandedFolders, setExpandedFolders] = useState<Set<string>>(new Set(['STAGED']));

    // Active Project Detection
    // currentPath is from backend, e.g. "my-project" or "/my-project"
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
    // Only used when inside a project
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
            // If already active, maybe go to root?
            runCommand(`cd /`);
        } else {
            // Switch project
            runCommand(`cd /${projectName}`);
        }
    };

    const handleDirClick = (node: FileNode, e: React.MouseEvent) => {
        e.stopPropagation();
        // Inside a project, normal toggle
        toggleFolder(node.path, e);
    };

    const handleDeleteProject = (projectName: string, e: React.MouseEvent) => {
        e.stopPropagation();
        if (confirm(`Are you sure you want to delete project '${projectName}'? This cannot be undone.`)) {
            // Need to change directory if we are inside the deleted project
            if (activeProject === projectName) {
                runCommand('cd /');
            }
            // Execute Delete
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
        <div className="file-explorer" style={{ color: 'var(--text-primary)', fontSize: '13px', fontFamily: 'system-ui, sans-serif', userSelect: 'none' }}>
            {/* STAGED CHANGES SECTION */}
            {stagedFiles.length > 0 && (
                <>
                    <div className="section-header"
                        style={{ padding: '8px 12px', fontSize: '11px', fontWeight: 'bold', color: 'var(--text-tertiary)', display: 'flex', alignItems: 'center', cursor: 'pointer' }}
                        onClick={(e) => toggleFolder('STAGED', e)}
                    >
                        <span style={{ transform: expandedFolders.has('STAGED') ? 'rotate(90deg)' : 'none', marginRight: '4px', display: 'inline-block', transition: 'transform 0.1s' }}>▶</span>
                        STAGED CHANGES
                        <span style={{ marginLeft: 'auto', fontWeight: 'normal', opacity: 0.7 }}>{stagedFiles.length}</span>
                    </div>
                    {expandedFolders.has('STAGED') && (
                        <div className="tree-content" style={{ marginBottom: '8px' }}>
                            {stagedFiles.map((file) => (
                                <div key={file.path} className="explorer-row" onClick={() => onSelect({ type: 'file', id: file.path })} style={{ paddingLeft: '12px' }}>
                                    <span className="icon">📄</span>
                                    <span className="name" style={{ color: '#27c93f' }}>{file.path}</span>
                                    <span className="status-badge">{file.status}</span>
                                </div>
                            ))}
                        </div>
                    )}
                </>
            )}

            {/* WORKSPACES HEAD */}
            <div className="section-header" style={{ padding: '8px 12px', fontSize: '11px', fontWeight: 'bold', color: 'var(--text-tertiary)', display: 'flex', alignItems: 'center' }}>
                WORKSPACES
            </div>

            <div className="tree-content">
                {projects.length === 0 ? (
                    <div style={{ padding: '12px', color: 'var(--text-tertiary)', fontStyle: 'italic' }}>
                        No projects found. Clone a repo!
                    </div>
                ) : (
                    projects.map(project => {
                        const isActive = project === activeProject;
                        // For UX, treat active project as having an expanded tree.
                        return (
                            <div key={project}>
                                <div
                                    className="explorer-row"
                                    onClick={() => handleProjectClick(project)}
                                    style={{ paddingLeft: '12px', backgroundColor: isActive ? 'rgba(255,255,255,0.08)' : undefined }}
                                >
                                    <span style={{
                                        marginRight: '6px',
                                        opacity: 0.8,
                                        display: 'inline-block',
                                        width: '16px',
                                        textAlign: 'center',
                                        transform: isActive ? 'rotate(90deg)' : 'none',
                                        transition: 'transform 0.2s'
                                    }}>▶</span>
                                    <span className="icon">📦</span>
                                    <span className="name" style={{ fontWeight: isActive ? 'bold' : 'normal' }}>{project}</span>

                                    <span
                                        className="delete-btn"
                                        title="Delete Project"
                                        onClick={(e) => handleDeleteProject(project, e)}
                                        style={{ marginLeft: '8px', opacity: 0.5, cursor: 'pointer', fontSize: '12px' }}
                                    >
                                        🗑️
                                    </span>
                                </div>
                                {isActive && (
                                    <div style={{ borderLeft: '1px solid rgba(255,255,255,0.1)', marginLeft: '19px' }}>
                                        {fileTree.length === 0 ? (
                                            <div style={{ padding: '4px 0 4px 12px', opacity: 0.5, fontStyle: 'italic' }}>Empty (or loading...)</div>
                                        ) : (
                                            renderTree(fileTree)
                                        )}
                                    </div>
                                )}
                            </div>
                        );
                    })
                )}
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
