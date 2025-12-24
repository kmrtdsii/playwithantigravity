import React from 'react';
import type { VizNode } from './graphTypes';
import { ROW_HEIGHT } from './graphConstants';
import { CommitBadge } from './GraphBadges';

interface CommitRowProps {
    node: VizNode;
    badges: Array<{ text: string; type: string; isActive?: boolean }>;
    isHovered: boolean;
    isSelected: boolean;
    onHover: (id: string | null) => void;
    onClick: () => void;
}

const TEXT_OFFSET_X = 140;

export const CommitRow: React.FC<CommitRowProps> = ({
    node,
    badges,
    isHovered,
    isSelected,
    onHover,
    onClick
}) => (
    <div
        onMouseEnter={() => onHover(node.id)}
        onMouseLeave={() => onHover(null)}
        onClick={onClick}
        style={{
            position: 'absolute',
            left: 0,
            top: node.y - ROW_HEIGHT / 2,
            width: '100%',
            paddingLeft: TEXT_OFFSET_X,
            boxSizing: 'border-box',
            height: ROW_HEIGHT,
            display: 'flex',
            alignItems: 'center',
            whiteSpace: 'nowrap',
            gap: '8px',
            cursor: 'pointer',
            paddingRight: '16px',
            userSelect: 'none',
            opacity: node.opacity,
            backgroundColor: isHovered || isSelected ? 'rgba(59, 130, 246, 0.1)' : 'transparent',
            borderLeft: isHovered || isSelected ? '4px solid var(--accent-primary)' : '4px solid transparent',
        }}
        className="commit-row"
    >
        {/* Commit ID */}
        <span
            onClick={(e) => e.stopPropagation()}
            style={{
                color: isSelected ? 'var(--accent-primary, #3b82f6)' : 'var(--text-tertiary)',
                fontSize: '10px',
                width: '60px',
                textAlign: 'left',
                flexShrink: 0,
                fontWeight: isSelected ? 'bold' : 'normal',
                marginRight: '8px',
                fontFamily: 'var(--font-mono)',
                userSelect: 'text',
                cursor: 'text'
            }}>
            {node.id.substring(0, 7)}
        </span>

        {/* Badges */}
        {badges.length > 0 && (
            <div style={{ display: 'flex', gap: '4px' }}>
                {badges.map((badge, i) => (
                    <CommitBadge key={i} badge={badge} color={node.color} />
                ))}
            </div>
        )}

        {/* Message */}
        <span
            onClick={(e) => e.stopPropagation()}
            title={node.message}
            style={{
                color: node.isGhost ? 'var(--text-tertiary)' : 'var(--text-secondary)',
                fontStyle: node.isGhost ? 'italic' : 'normal',
                flex: 1,
                overflow: 'hidden',
                textOverflow: 'ellipsis',
                minWidth: 0,
                userSelect: 'text',
                cursor: 'text'
            }}
        >
            {node.isGhost && '[SIMULATION] '}
            {node.message}
        </span>

        {/* Timestamp */}
        <span style={{
            color: 'var(--text-tertiary)',
            fontSize: '10px',
            width: '140px',
            textAlign: 'right',
            flexShrink: 0,
            marginRight: '8px'
        }}>
            {new Date(node.timestamp).toLocaleString('ja-JP', {
                year: 'numeric', month: '2-digit', day: '2-digit',
                hour: '2-digit', minute: '2-digit', second: '2-digit'
            })}
        </span>

        {/* Commit ID */}
    </div>
);
