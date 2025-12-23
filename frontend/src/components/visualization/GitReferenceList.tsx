
import React, { useMemo } from 'react';
import { useGit } from '../../context/GitAPIContext';
import type { Commit } from '../../types/gitTypes';
import { Cloud, GitBranch } from 'lucide-react';

interface GitReferenceListProps {
    type: 'branches' | 'tags';
    onSelect?: (commit: Commit) => void;
    selectedCommitId?: string;
}

const GitReferenceList: React.FC<GitReferenceListProps> = ({ type, onSelect, selectedCommitId }) => {
    const { state } = useGit();
    const references = type === 'branches' ? state.branches : state.tags;
    const { commits } = state;

    // Create a map of Commit ID -> Commit Object for easy lookup
    const commitMap = new Map(commits.map(c => [c.id, c]));

    const listItems = useMemo(() => {
        const items: { name: string; commitId: string; commit: Commit | undefined; isRemote: boolean }[] = [];

        // 1. Local Branches / Tags
        if (references) {
            Object.entries(references).forEach(([name, commitId]) => {
                const commit = commitMap.get(commitId);
                items.push({ name, commitId, commit, isRemote: false });
            });
        }

        // 2. Remote Branches (only if type is 'branches')
        if (type === 'branches' && state.remoteBranches) {
            Object.entries(state.remoteBranches).forEach(([name, commitId]) => {
                const commit = commitMap.get(commitId);
                items.push({ name, commitId, commit, isRemote: true });
            });
        }

        // Sort items
        return items.sort((a, b) => {
            if (type === 'tags') {
                const timeA = a.commit ? new Date(a.commit.timestamp).getTime() : 0;
                const timeB = b.commit ? new Date(b.commit.timestamp).getTime() : 0;
                return timeB - timeA;
            }
            return a.name.localeCompare(b.name);
        });
    }, [references, state.remoteBranches, commits, type]);

    if (!listItems.length) {
        return (
            <div className="flex h-full items-center justify-center text-gray-500 font-mono text-sm">
                No {type} found.
            </div>
        );
    }

    return (
        <div style={{
            height: '100%',
            overflow: 'auto',
            background: 'var(--bg-primary)',
            color: 'var(--text-primary)',
            fontFamily: 'Menlo, Monaco, Consolas, monospace',
            fontSize: '12px'
        }}>
            <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                <thead style={{ position: 'sticky', top: 0, background: 'var(--bg-tertiary)', borderBottom: '1px solid var(--border-subtle)' }}>
                    <tr>
                        <th style={{ textAlign: 'left', padding: '8px 16px', width: '150px' }}>Name</th>
                        <th style={{ textAlign: 'left', padding: '8px 16px', width: '80px' }}>Hash</th>
                        <th style={{ textAlign: 'left', padding: '8px 16px' }}>Message</th>
                        <th style={{ textAlign: 'right', padding: '8px 16px', width: '150px' }}>Date</th>
                    </tr>
                </thead>
                <tbody>
                    {listItems.map((item) => {
                        const isSelected = item.commitId === selectedCommitId;
                        return (
                            <tr
                                key={item.name}
                                onClick={() => item.commit && onSelect && onSelect(item.commit)}
                                style={{
                                    cursor: 'pointer',
                                    borderBottom: '1px solid var(--border-subtle)',
                                    backgroundColor: isSelected ? 'rgba(59, 130, 246, 0.1)' : 'transparent',
                                    // Use box-shadow for left accent border in table row
                                    boxShadow: isSelected ? 'inset 4px 0 0 var(--accent-primary)' : 'none',
                                    transition: 'background-color 0.2s',
                                    ':hover': { backgroundColor: 'var(--bg-secondary)' }
                                } as any}
                                className="hover:bg-opacity-10 hover:bg-white"
                            >
                                <td style={{ padding: '8px 16px', fontWeight: 'bold', color: type === 'branches' ? (item.isRemote ? 'var(--text-secondary)' : 'var(--accent-primary)') : 'var(--text-secondary)' }}>
                                    <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                                        {type === 'branches' && (item.isRemote ? <Cloud size={14} /> : <GitBranch size={14} />)}
                                        {item.name}
                                    </div>
                                </td>
                                <td style={{ padding: '8px 16px', color: 'var(--text-tertiary)' }}>
                                    {item.commitId.substring(0, 7)}
                                </td>
                                <td style={{ padding: '8px 16px', color: 'var(--text-secondary)', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis', maxWidth: '300px' }}>
                                    {item.commit?.message || '<unknown commit>'}
                                </td>
                                <td style={{ padding: '8px 16px', textAlign: 'right', color: 'var(--text-tertiary)' }}>
                                    {item.commit ? new Date(item.commit.timestamp).toLocaleString('ja-JP', {
                                        year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', second: '2-digit'
                                    }) : '-'}
                                </td>
                            </tr>
                        );
                    })}
                </tbody>
            </table>
        </div>
    );
};

export default GitReferenceList;
