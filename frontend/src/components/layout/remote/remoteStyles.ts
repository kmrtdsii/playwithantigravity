import React from 'react';

/**
 * Shared styles for RemoteRepoView components
 */

export const containerStyle: React.CSSProperties = {
    height: '100%',
    display: 'flex',
    flexDirection: 'column',
    background: 'var(--bg-primary)',
    overflow: 'hidden'
};

export const actionButtonStyle: React.CSSProperties = {
    background: 'var(--accent-primary)',
    color: 'white',
    border: 'none',
    borderRadius: '4px',
    padding: '4px 10px',
    fontSize: '10px',
    fontWeight: 700,
    cursor: 'pointer'
};

export const sectionLabelStyle: React.CSSProperties = {
    fontSize: '0.75rem',
    fontWeight: 800,
    color: 'var(--text-secondary)',
    letterSpacing: '0.05em'
};

export const prCardStyle: React.CSSProperties = {
    display: 'flex',
    flexDirection: 'column',
    gap: '4px',
    padding: '10px 12px',
    background: 'var(--bg-secondary)',
    color: 'var(--text-primary)',
    borderRadius: '10px',
    border: '1px solid var(--border-subtle)',
};

export const mergeButtonStyle: React.CSSProperties = {
    width: '100%',
    padding: '5px',
    background: '#8957e5',
    color: 'white',
    border: 'none',
    borderRadius: '6px',
    fontSize: '11px',
    fontWeight: 700,
    cursor: 'pointer',
    marginTop: '4px'
};

export const emptyStyle: React.CSSProperties = {
    fontSize: '0.8rem',
    color: 'var(--text-tertiary)',
    fontStyle: 'italic',
    padding: '12px',
    border: '1px dashed var(--border-subtle)',
    borderRadius: '10px',
    textAlign: 'center'
};

export const inputStyle: React.CSSProperties = {
    flex: 1,
    padding: '4px 8px',
    borderRadius: '4px',
    border: '1px solid var(--accent-primary)',
    background: 'var(--bg-primary)',
    color: 'var(--text-primary)',
    fontSize: '13px',
    outline: 'none',
    height: '28px'
};

export const cancelButtonStyle: React.CSSProperties = {
    padding: '4px 8px',
    fontSize: '12px',
    background: 'transparent',
    color: 'var(--text-secondary)',
    border: '1px solid var(--border-subtle)',
    borderRadius: '4px',
    cursor: 'pointer',
    height: '28px',
    display: 'flex',
    alignItems: 'center'
};

export const submitButtonStyle: React.CSSProperties = {
    padding: '4px 12px',
    fontSize: '12px',
    fontWeight: 700,
    background: 'var(--accent-primary)',
    color: 'white',
    border: 'none',
    borderRadius: '4px',
    cursor: 'pointer',
    height: '28px',
    display: 'flex',
    alignItems: 'center'
};

export const headerStyle: React.CSSProperties = {
    display: 'flex',
    flexDirection: 'column',
    width: '100%',
    background: 'var(--bg-secondary)',
    flexShrink: 0
};

export const tabRowStyle: React.CSSProperties = {
    height: '36px',
    display: 'flex',
    alignItems: 'flex-end',
    padding: '0 8px',
    gap: '4px',
    borderBottom: '1px solid var(--border-subtle)',
    background: 'var(--bg-secondary)',
    overflowX: 'auto',
    scrollbarWidth: 'none'
};

export const toolbarRowStyle: React.CSSProperties = {
    height: '40px',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
    padding: '0 12px',
    gap: '8px',
    borderBottom: '1px solid var(--border-subtle)',
    background: 'var(--bg-toolbar)'
};
