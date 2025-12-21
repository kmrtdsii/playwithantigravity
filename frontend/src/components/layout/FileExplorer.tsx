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
    const { state } = useGit();
    const [expandedFolders, setExpandedFolders] = useState<Set<string>>(new Set(['GITGYM', 'STAGED']));

    // Compute Staged Files
    const stagedFiles = useMemo(() => {
        return Object.entries(state.fileStatuses || {})
            .filter(([, status]) => {
                // Status is XY. X=Staged, Y=Worktree.
                // If X is not ' ' and not '?', it's staged.
                const x = status[0];
                return x !== ' ' && x !== '?';
            })
            .map(([path, status]) => ({ path, status }));
    }, [state.fileStatuses]);

    // Build Entry Tree from flat file list (Working Tree)
    const fileTree = useMemo(() => {
        const root: FileNode = { name: 'ROOT', path: '', isDir: true, children: [] };

        // Files from `ls` (all files)
        const allFiles = state.files || [];

        // Merge with untracked files if not present? 
        // Actually state.files should contain everything if `ls` works.
        // But `ls` in our backend implementation might walk everything.
        // Let's rely on state.files.

        allFiles.forEach(filePath => {
            const parts = filePath.split('/');
            let currentLevel = root.children!;

            parts.forEach((part, index) => {
                const isLast = index === parts.length - 1;
                const currentPath = parts.slice(0, index + 1).join('/');

                let existingNode = currentLevel.find(n => n.name === part);

                if (!existingNode) {
                    const newNode: FileNode = {
                        name: part,
                        path: currentPath,
                        isDir: !isLast,
                        children: !isLast ? [] : undefined,
                        status: isLast ? state.fileStatuses[filePath] : undefined
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
    }, [state.files, state.fileStatuses]);

    const toggleFolder = (path: string, e: React.MouseEvent) => {
        e.stopPropagation();
        const newSet = new Set(expandedFolders);
        if (newSet.has(path)) {
            newSet.delete(path);
        } else {
            newSet.add(path);
        }
        setExpandedFolders(newSet);
    };

    // Toggle specifically for sections (using same set for simplicity)
    // const isSectionOpen = (key: string) => !expandedFolders.has(key); // Default open implies we track closed? 
    // Actually current logic `isExpanded` meant "is open". 
    // Let's stick to "Set contains Open items".
    // But for sections maybe we want them open by default?
    // Let's Initialize expandedFolders with 'GITGYM' and 'STAGED'.

    // Changing initialization in component:
    // const [expandedFolders, setExpandedFolders] = useState<Set<string>>(new Set(['GITGYM', 'STAGED']));

    const getStatusColor = (status?: string) => {
        if (!status) return 'var(--text-secondary)';
        if (status.includes('?')) return '#ff5f56'; // Untracked (Red)
        if (status.includes('M')) return '#ffbd2e'; // Modified (Yellow)
        if (status.includes('A')) return '#27c93f'; // Added (Green) 
        return 'var(--text-primary)';
    };

    const renderTree = (nodes: FileNode[], depth: number = 0) => {
        return nodes.map(node => {
            const isExpanded = expandedFolders.has(node.path);
            const paddingLeft = depth * 16 + 12; // Indentation

            if (node.isDir) {
                return (
                    <div key={node.path}>
                        <div
                            className="explorer-row"
                            onClick={(e) => toggleFolder(node.path, e)}
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
                        <span
                            className="name"
                            style={{ color: getStatusColor(node.status) }}
                        >
                            {node.name}
                        </span>
                        {node.status && (
                            <span className="status-badge">{node.status}</span>
                        )}
                    </div>
                );
            }
        });
    };

    return (
        <div className="file-explorer" style={{
            color: 'var(--text-primary)',
            fontSize: '13px',
            fontFamily: 'system-ui, sans-serif',
            userSelect: 'none'
        }}>
            {/* STAGED CHANGES SECTION */}
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
                    {stagedFiles.length === 0 ? (
                        <div style={{ padding: '4px 12px', fontStyle: 'italic', opacity: 0.5, fontSize: '12px' }}>No staged changes</div>
                    ) : (
                        stagedFiles.map((file) => (
                            <div
                                key={file.path}
                                className="explorer-row"
                                onClick={() => onSelect({ type: 'file', id: file.path })}
                                style={{ paddingLeft: '12px' }}
                            >
                                <span className="icon">📄</span>
                                <span className="name" style={{ color: '#27c93f' }}>{/* Green for staged usually? Or just text? Let's use green. */ file.path}</span>
                                <span className="status-badge">{file.status}</span>
                            </div>
                        ))
                    )}
                </div>
            )}

            {/* PROJECT ROOT SECTION */}
            <div className="section-header"
                style={{ padding: '8px 12px', fontSize: '11px', fontWeight: 'bold', color: 'var(--text-tertiary)', display: 'flex', alignItems: 'center', cursor: 'pointer' }}
                onClick={(e) => toggleFolder('GITGYM', e)}
            >
                <span style={{ transform: expandedFolders.has('GITGYM') ? 'rotate(90deg)' : 'none', marginRight: '4px', display: 'inline-block', transition: 'transform 0.1s' }}>▶</span>
                GITGYM
            </div>

            {expandedFolders.has('GITGYM') && (
                <div className="tree-content">
                    {fileTree.length === 0 ? (
                        <div style={{ padding: '12px', color: 'var(--text-tertiary)', fontStyle: 'italic' }}>
                            Empty workspace
                        </div>
                    ) : (
                        renderTree(fileTree)
                    )}
                </div>
            )}

            <style>{`
                .explorer-row {
                    display: flex;
                    alignItems: center;
                    padding-top: 4px;
                    padding-bottom: 4px;
                    cursor: pointer;
                }
                .explorer-row:hover {
                    background-color: rgba(255, 255, 255, 0.05);
                }
                .icon {
                    margin-right: 6px;
                    opacity: 0.8;
                    display: inline-block;
                    width: 16px;
                    text-align: center;
                }
                .name {
                    flex: 1;
                    overflow: hidden;
                    text-overflow: ellipsis;
                    white-space: nowrap;
                }
                .status-badge {
                    font-size: 10px;
                    opacity: 0.7;
                    margin-right: 8px;
                    font-family: monospace;
                }
            `}</style>
        </div>
    );
};

export default FileExplorer;
