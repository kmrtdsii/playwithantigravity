import React from 'react';
import { Tag, Cloud } from 'lucide-react';

export interface CommitBadgeProps {
    badge: { text: string; type: string; isActive?: boolean };
    color: string;
}

export const CommitBadge: React.FC<CommitBadgeProps> = ({ badge, color }) => (
    <span style={{
        fontSize: '10px',
        padding: '1px 6px',
        borderRadius: '10px',
        fontWeight: badge.isActive ? 'bold' : 'normal',
        backgroundColor: 'transparent',
        border: `1px solid ${color}`,
        color: color,
        opacity: 0.9,
        display: 'flex',
        alignItems: 'center',
        gap: '4px'
    }}>
        {badge.type === 'tag' && <Tag size={11} strokeWidth={2.5} />}
        {badge.type === 'remote-branch' && <Cloud size={11} strokeWidth={2.5} />}
        {badge.text}
    </span>
);
