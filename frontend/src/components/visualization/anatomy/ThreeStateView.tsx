import React, { useMemo } from 'react';
import { useGit } from '../../../context/GitAPIContext';
import FileCard from './FileCard';

const ThreeStateView: React.FC = () => {
    const { state, stageFile, unstageFile } = useGit();

    // 1. Categorize Files
    const { worktreeFiles, indexFiles, headFiles } = useMemo(() => {
        const wt: { name: string; status: string }[] = [];
        const idx: { name: string; status: string }[] = [];
        const hd: { name: string; status: string }[] = []; // In a real implementation, this would be populated from HEAD tree

        // Iterate through all known files in the status map
        // Note: state.files contains ALL workspace files. state.fileStatuses contains CHANGED files.
        // We want to show:
        // - Worktree: Modified in worktree (XY: _M, M_, MM, ??) -> Actually X=Index, Y=Worktree. 
        //   gogit status codes: 
        //   ' ' = Unmodified
        //   'M' = Modified
        //   'A' = Added
        //   'D' = Deleted
        //   '?' = Untracked

        // Mapping (X = Staging, Y = Worktree)
        // ?? -> Worktree (Untracked)
        //  M -> Worktree (Modified)
        // MM -> Worktree (Modified) AND Index (Staged)
        // M  -> Index (Staged)
        // A  -> Index (Staged)
        // D  -> Index (Deleted)
        //  D -> Worktree (Deleted)

        Object.entries(state.fileStatuses).forEach(([path, status]) => {
            const x = status[0];
            const y = status[1];

            // Worktree Logic
            if (y === 'M' || y === 'D' || status === '??') {
                wt.push({ name: path, status: status });
            }

            // Index Logic
            if (x === 'M' || x === 'A' || x === 'D' || x === 'R') {
                idx.push({ name: path, status: status });
            }
        });

        // For HEAD, we ideally list all files in HEAD. 
        // Since we don't have that list easily separate from "Files" (which is workspace), 
        // we can approximate or just show "Everything that is NOT Added/New" from `state.files`.
        // Or cleaner: Show "Committable Snapshots". 
        // For this visualizer, let's just list files that are "clean" or "staged" (as they will be in HEAD).
        // Actually, HEAD column usually represents "Last Commit". 
        // So it should show files as they were in HEAD.
        // If file is "A ", it is NOT in HEAD.
        // If file is "M ", it IS in HEAD (old version).
        // If "??", NOT in HEAD.

        // Let's iterate `state.files` (which is current disk). 
        // If status is 'A' or '??', skip for HEAD.
        state.files.forEach(f => {
            const status = state.fileStatuses[f];
            if (!status || (status[0] !== 'A' && status !== '??')) {
                hd.push({ name: f, status: 'Committed' });
            }
        });

        return { worktreeFiles: wt, indexFiles: idx, headFiles: hd };
    }, [state.fileStatuses, state.files]);

    const handleDragEnd = (targetCol: 'worktree' | 'index' | 'head', filename: string) => {
        if (targetCol === 'index') {
            console.log(`Staging ${filename}`);
            if (stageFile) stageFile(filename);
        } else if (targetCol === 'worktree') {
            console.log(`Unstaging ${filename}`);
            if (unstageFile) unstageFile(filename);
        }
    };

    return (
        <div style={{ display: 'flex', height: '100%', gap: '16px', padding: '16px', background: 'var(--bg-primary)' }}>

            {/* WORKTREE */}
            <div style={columnStyle}>
                <div style={headerStyle('var(--accent-secondary)')}>
                    Worktree <span style={subHeaderStyle}>(Actual Files)</span>
                </div>
                <div style={listStyle}>
                    {worktreeFiles.map(f => (
                        <FileCard
                            key={f.name}
                            filename={f.name}
                            status={f.status}
                            column="worktree"
                            onDragEnd={handleDragEnd}
                        />
                    ))}
                    {worktreeFiles.length === 0 && <div style={emptyStyle}>Clean</div>}
                </div>
            </div>

            {/* INDEX (STAGE) */}
            <div style={columnStyle}>
                <div style={headerStyle('var(--accent-primary)')}>
                    Index <span style={subHeaderStyle}>(Staging Area)</span>
                </div>
                <div style={listStyle}>
                    {indexFiles.map(f => (
                        <FileCard
                            key={f.name}
                            filename={f.name}
                            status={f.status}
                            column="index"
                            onDragEnd={handleDragEnd}
                        />
                    ))}
                    {indexFiles.length === 0 && <div style={emptyStyle}>Empty</div>}
                </div>
            </div>

            {/* HEAD */}
            <div style={columnStyle}>
                <div style={headerStyle('var(--text-secondary)')}>
                    HEAD <span style={subHeaderStyle}>(Last Commit)</span>
                </div>
                <div style={listStyle}>
                    {headFiles.map(f => (
                        <FileCard
                            key={f.name}
                            filename={f.name}
                            status={f.status}
                            column="head"
                        />
                    ))}
                </div>
            </div>

        </div>
    );
};

const columnStyle: React.CSSProperties = {
    flex: 1,
    display: 'flex',
    flexDirection: 'column',
    background: 'var(--bg-tertiary)',
    borderRadius: '8px',
    overflow: 'hidden',
    border: '1px solid var(--border-subtle)'
};

const headerStyle = (color: string): React.CSSProperties => ({
    padding: '12px',
    borderBottom: '1px solid var(--border-subtle)',
    background: 'var(--bg-secondary)',
    fontWeight: 700,
    color: color,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    gap: '8px'
});

const subHeaderStyle: React.CSSProperties = {
    fontSize: '0.7em',
    color: 'var(--text-tertiary)',
    fontWeight: 400
};

const listStyle: React.CSSProperties = {
    flex: 1,
    padding: '12px',
    overflowY: 'auto'
};

const emptyStyle: React.CSSProperties = {
    textAlign: 'center',
    color: 'var(--text-tertiary)',
    marginTop: '20px',
    fontStyle: 'italic'
};

export default ThreeStateView;
