import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Folder, FolderOpen, GitBranch, ChevronRight, ChevronDown } from 'lucide-react';

export interface DirectoryNode {
    path: string;
    name: string;
    isDir: boolean;
    isRepo: boolean;
    branch?: string;
    children?: DirectoryNode[];
}

interface WorkspaceTreeProps {
    tree: DirectoryNode[];
    currentPath: string;
    currentRepo: string;
    onNavigate: (path: string) => void;
}

interface TreeNodeProps {
    node: DirectoryNode;
    depth: number;
    currentPath: string;
    currentRepo: string;
    onNavigate: (path: string) => void;
}

const TreeNode: React.FC<TreeNodeProps> = ({ node, depth, currentPath, currentRepo, onNavigate }) => {
    // Top-level items (repos) start expanded, nested folders start collapsed
    const [isExpanded, setIsExpanded] = useState(depth === 0);
    const hasChildren = node.children && node.children.length > 0;
    const isCurrentPath = currentPath === node.path || currentPath.startsWith(node.path + '/');
    const isActiveRepo = currentRepo === node.path;

    const handleClick = () => {
        if (hasChildren) {
            setIsExpanded(!isExpanded);
        }
        onNavigate(node.path);
    };

    // Skip root node display, show children directly
    if (node.path === '/') {
        return (
            <>
                {node.children?.map((child) => (
                    <TreeNode
                        key={child.path}
                        node={child}
                        depth={0}
                        currentPath={currentPath}
                        currentRepo={currentRepo}
                        onNavigate={onNavigate}
                    />
                ))}
            </>
        );
    }

    return (
        <div>
            <div
                onClick={handleClick}
                style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: '4px',
                    padding: '6px 8px',
                    paddingLeft: `${depth * 16 + 8}px`,
                    cursor: 'pointer',
                    borderRadius: 'var(--radius-sm)',
                    background: isCurrentPath
                        ? 'var(--bg-selection)'
                        : isActiveRepo ? 'var(--accent-primary-alpha-10)' : 'transparent',
                    borderLeft: isActiveRepo ? '2px solid var(--accent-primary)' : '2px solid transparent',
                    transition: 'background 0.15s',
                }}
                onMouseOver={(e) => {
                    if (!isActiveRepo) {
                        e.currentTarget.style.background = 'var(--bg-tertiary)';
                    }
                }}
                onMouseOut={(e) => {
                    if (!isActiveRepo) {
                        e.currentTarget.style.background = 'transparent';
                    }
                }}
            >
                {/* Expand/Collapse Icon */}
                {hasChildren ? (
                    isExpanded ? (
                        <ChevronDown size={14} style={{ color: 'var(--text-tertiary)', flexShrink: 0 }} />
                    ) : (
                        <ChevronRight size={14} style={{ color: 'var(--text-tertiary)', flexShrink: 0 }} />
                    )
                ) : (
                    <span style={{ width: 14, flexShrink: 0 }} />
                )}

                {/* Folder Icon */}
                {node.isRepo ? (
                    <GitBranch size={14} style={{ color: 'var(--accent-primary)', flexShrink: 0 }} />
                ) : isExpanded ? (
                    <FolderOpen size={14} style={{ color: 'var(--text-secondary)', flexShrink: 0 }} />
                ) : (
                    <Folder size={14} style={{ color: 'var(--text-secondary)', flexShrink: 0 }} />
                )}

                {/* Name */}
                <span style={{
                    fontSize: '13px',
                    color: node.isRepo ? 'var(--text-primary)' : 'var(--text-secondary)',
                    fontWeight: node.isRepo ? 500 : 400,
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                    whiteSpace: 'nowrap',
                }}>
                    {node.name}
                </span>

                {/* Branch indicator for repos */}
                {node.isRepo && node.branch && (
                    <span style={{
                        fontSize: '11px',
                        color: 'var(--accent-primary)',
                        background: 'var(--accent-primary-alpha-10)',
                        padding: '1px 6px',
                        borderRadius: 'var(--radius-sm)',
                        marginLeft: 'auto',
                        flexShrink: 0,
                    }}>
                        {node.branch}
                    </span>
                )}
            </div>

            {/* Children */}
            {isExpanded && hasChildren && (
                <div>
                    {node.children!.map((child) => (
                        <TreeNode
                            key={child.path}
                            node={child}
                            depth={depth + 1}
                            currentPath={currentPath}
                            currentRepo={currentRepo}
                            onNavigate={onNavigate}
                        />
                    ))}
                </div>
            )}
        </div>
    );
};

const WorkspaceTree: React.FC<WorkspaceTreeProps> = ({
    tree,
    currentPath,
    currentRepo,
    onNavigate
}) => {
    const { t } = useTranslation('common');

    if (!tree || tree.length === 0) {
        return (
            <div style={{
                padding: '20px',
                textAlign: 'center',
                color: 'var(--text-tertiary)',
                fontSize: '13px',
            }}>
                {t('workspace.empty', { defaultValue: 'ワークスペースが空です' })}
            </div>
        );
    }

    // Check if workspace has any content
    const rootNode = tree[0];
    const hasContent = rootNode.children && rootNode.children.length > 0;

    if (!hasContent) {
        return (
            <div style={{
                padding: '12px 16px',
                textAlign: 'left',
                color: 'var(--text-tertiary)',
                fontSize: '13px',
                lineHeight: '1.6'
            }}>
                <div style={{ marginBottom: '8px', color: 'var(--text-secondary)', fontWeight: 500 }}>
                    {t('workspace.noProjects', { defaultValue: 'ワークスペースは空です' })}
                </div>
                <div style={{ fontSize: '12px', opacity: 0.8, lineHeight: '1.5' }}>
                    {t('workspace.createHint', { defaultValue: 'mkdir でフォルダを作るか、git clone しましょう。' })}
                </div>
            </div>
        );
    }

    return (
        <div style={{ padding: '4px 0' }}>
            {tree.map((node) => (
                <TreeNode
                    key={node.path}
                    node={node}
                    depth={0}
                    currentPath={currentPath}
                    currentRepo={currentRepo}
                    onNavigate={onNavigate}
                />
            ))}
        </div>
    );
};

export default WorkspaceTree;
