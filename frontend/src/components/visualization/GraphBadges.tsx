import React from 'react';

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
        alignItems: 'center'
    }}>
        {badge.type === 'tag' && <TagIcon />}
        {badge.type === 'remote-branch' && <CloudIcon />}
        {badge.text}
    </span>
);

export const TagIcon: React.FC = () => (
    <svg
        viewBox="0 0 24 24"
        width="11"
        height="11"
        stroke="currentColor"
        strokeWidth="2.5"
        fill="none"
        strokeLinecap="round"
        strokeLinejoin="round"
        style={{ marginRight: '4px', verticalAlign: 'middle', display: 'inline-block', flexShrink: 0, opacity: 0.8 }}
    >
        <path d="M12 2H2v10l9.29 9.29c.94.94 2.48.94 3.42 0l7.29-7.29c.94-.94.94-2.48 0-3.42L12 2z" />
        <path d="M7 7h.01" />
    </svg>
);

export const CloudIcon: React.FC = () => (
    <svg
        viewBox="0 0 24 24"
        width="11"
        height="11"
        stroke="currentColor"
        strokeWidth="2.5"
        fill="none"
        strokeLinecap="round"
        strokeLinejoin="round"
        style={{ marginRight: '4px', verticalAlign: 'middle', display: 'inline-block', flexShrink: 0, opacity: 0.8 }}
    >
        <path d="M17.5 19c0-3.037-2.463-5.5-5.5-5.5S6.5 15.963 6.5 19" />
        <path d="M20.9 14.1a6 6 0 1 0-8.9-8.1 4 4 0 0 0-5.8 5.7" />
    </svg>
);
