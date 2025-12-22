import React from 'react';
import { motion } from 'framer-motion';

interface FileCardProps {
    filename: string;
    status: string;
    column: 'worktree' | 'index' | 'head';
    onDragEnd?: (column: 'worktree' | 'index' | 'head', filename: string) => void;
}

const FileCard: React.FC<FileCardProps> = ({ filename, status, column, onDragEnd }) => {
    return (
        <motion.div
            layout // Enable layout animation for smooth sorting/moving
            layoutId={filename} // layoutId is key for shared layout animations across lists
            drag={column !== 'head'} // HEAD items not draggable (for now)
            dragSnapToOrigin
            whileDrag={{ scale: 1.05, zIndex: 100 }}
            onDragEnd={(_, info) => {
                // Simple heuristic for dropping in columns
                // Drag to Right -> Promote (Worktree -> Index)
                // Drag to Left -> Demote (Index -> Worktree)

                if (column === 'worktree' && info.offset.x > 100) {
                    onDragEnd?.('index', filename);
                } else if (column === 'index' && info.offset.x < -100) {
                    onDragEnd?.('worktree', filename);
                }
            }}
            style={{
                background: 'var(--bg-secondary)',
                padding: '8px 12px',
                marginBottom: '8px',
                borderRadius: '6px',
                border: '1px solid var(--border-subtle)',
                cursor: column === 'head' ? 'default' : 'grab',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                boxShadow: '0 1px 2px rgba(0,0,0,0.05)'
            }}
        >
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                <span style={{ fontSize: '16px' }}>
                    {getIcon(filename)}
                </span>
                <span style={{ fontSize: '0.9rem', color: 'var(--text-primary)' }}>
                    {filename}
                </span>
            </div>

            {status && column !== 'head' && (
                <span style={{
                    fontSize: '0.7rem',
                    fontWeight: 700,
                    color: getStatusColor(status),
                    background: getStatusBg(status),
                    padding: '2px 6px',
                    borderRadius: '4px'
                }}>
                    {status}
                </span>
            )}
        </motion.div>
    );
};

const getIcon = (filename: string) => {
    if (filename.endsWith('.go')) return '🐹';
    if (filename.endsWith('.ts') || filename.endsWith('.tsx')) return '📘';
    if (filename.endsWith('.js') || filename.endsWith('.jsx')) return '📒';
    if (filename.endsWith('.md')) return '📝';
    if (filename.endsWith('.json')) return '📦';
    if (filename.endsWith('.css')) return '🎨';
    return '📄';
};

const getStatusColor = (status: string) => {
    if (status.includes('?')) return '#d97706'; // Untracked (Yellow/Orange)
    if (status.includes('M')) return '#2563eb'; // Modified (Blue)
    if (status.includes('A')) return '#16a34a'; // Added (Green)
    if (status.includes('D')) return '#dc2626'; // Deleted (Red)
    return '#6b7280';
};

const getStatusBg = (status: string) => {
    // Lighter versions
    if (status.includes('?')) return '#fffbeb';
    if (status.includes('M')) return '#eff6ff';
    if (status.includes('A')) return '#f0fdf4';
    if (status.includes('D')) return '#fef2f2';
    return '#f3f4f6';
}

export default FileCard;
