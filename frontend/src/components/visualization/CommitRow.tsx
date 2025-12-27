import React from 'react';
import type { VizNode } from './graphTypes';
import { ROW_HEIGHT, TEXT_OFFSET_X, DATE_WIDTH } from './graphConstants';
import { CommitBadge } from './GraphBadges';

interface CommitRowProps {
    node: VizNode;
    badges: Array<{ text: string; type: string; isActive?: boolean }>;
    isSelected: boolean;
    onClick: () => void;
}

export const CommitRow: React.FC<CommitRowProps> = ({
    node,
    badges,
    isSelected,
    onClick
}) => (
    <div
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
            /* Hover/Select styles moved to CSS */
        }}
        className={`commit-row ${isSelected ? 'selected' : ''}`}
        data-testid="commit-row"
    >
        {/* Commit ID */}
        <span
            onClick={onClick} // Allow click to trigger selection
            className="commit-hash"
            style={{
                color: isSelected ? 'var(--text-primary)' : 'var(--text-tertiary)',
                fontSize: '10px',
                width: '60px',
                textAlign: 'left',
                flexShrink: 0,
                fontWeight: isSelected ? 'bold' : 'normal',
                marginRight: '8px',
                fontFamily: 'var(--font-mono)',
                userSelect: 'text',
                cursor: 'pointer', // Show pointer
                textDecoration: 'underline', // Make it look like a link
                textUnderlineOffset: '2px'
            }}>
            {node.id.substring(0, 7)}
        </span>

        {/* Badges */}
        {badges.length > 0 && (
            <div style={{ display: 'flex', gap: '4px' }} onClick={(e) => e.stopPropagation()}>
                {badges.map((badge, i) => (
                    <CommitBadge key={i} badge={badge} color={node.color} />
                ))}
            </div>
        )}

        {/* Message */}
        <span
            data-testid="commit-message"
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
            width: `${DATE_WIDTH}px`,
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
