import { useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import '@xterm/xterm/css/xterm.css';
import { useGit } from '../../context/GitAPIContext';
import { useTerminal } from '../../hooks/useTerminal';

const GitTerminal = () => {
    const terminalRef = useRef<HTMLDivElement>(null);
    const xtermRef = useRef<Terminal | null>(null);
    const fitAddonRef = useRef<FitAddon | null>(null);

    const { t } = useTranslation('common');
    const { activeDeveloper, state } = useGit();
    const [allowEmpty, setAllowEmpty] = useState(true);

    useTerminal(terminalRef, xtermRef, fitAddonRef, allowEmpty);

    const currentBranch = state.HEAD?.ref || (state.HEAD?.id ? state.HEAD.id.substring(0, 7) : 'DETACHED');
    const isDetached = !state.HEAD?.ref && !!state.HEAD?.id;

    return (
        <div data-testid="git-terminal" style={{ width: '100%', height: '100%', display: 'flex', flexDirection: 'column', boxSizing: 'border-box', background: 'var(--bg-primary)' }}>
            <div style={{
                height: 'var(--header-height)',
                display: 'flex',
                alignItems: 'center',
                padding: '0 var(--space-3)',
                background: 'var(--bg-secondary)',
                borderBottom: '1px solid var(--border-subtle)',
                fontSize: 'var(--text-xs)',
                gap: 'var(--space-4)',
                flexShrink: 0
            }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--radius-md)' }}>
                    <span style={{ color: 'var(--text-secondary)', fontWeight: 'var(--font-extrabold)', fontSize: 'var(--text-xs)', letterSpacing: '0.05em' }}>{t('terminal.user')}:</span>
                    <span data-testid="user-value" style={{ color: 'var(--accent-primary)', fontWeight: 'var(--font-semibold)' }}>{activeDeveloper}</span>
                </div>
                <div style={{ width: '1px', height: '12px', background: 'var(--border-subtle)' }} />
                <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--radius-md)' }}>
                    <span style={{ color: 'var(--text-secondary)', fontWeight: 'var(--font-extrabold)', fontSize: 'var(--text-xs)', letterSpacing: '0.05em' }}>{t('terminal.branch')}:</span>
                    <span data-testid="branch-value" style={{
                        color: isDetached ? 'var(--text-warning)' : 'var(--text-secondary)',
                        fontFamily: 'monospace'
                    }}>
                        {currentBranch}
                    </span>
                </div>

                {/* Allow Empty Toggle */}
                <div style={{ marginLeft: 'auto', display: 'flex', alignItems: 'center' }}>
                    <label style={{ display: 'flex', alignItems: 'center', cursor: 'pointer', gap: '6px', userSelect: 'none' }} title={t('terminal.allowEmptyTooltip')}>
                        <input
                            type="checkbox"
                            checked={allowEmpty}
                            onChange={(e) => setAllowEmpty(e.target.checked)}
                            style={{ accentColor: 'var(--accent-primary)', cursor: 'pointer' }}
                        />
                        <span style={{ color: 'var(--text-secondary)', fontWeight: '600', fontSize: '11px' }}>{t('terminal.allowEmpty')}</span>
                    </label>
                </div>
            </div>

            <div style={{ flex: 1, minHeight: 0, paddingLeft: 'var(--space-4)', paddingTop: 'var(--space-3)', paddingBottom: 'var(--space-3)' }}>
                <div
                    ref={terminalRef}
                    data-testid="terminal-canvas-container"
                    style={{ width: '100%', height: '100%' }}
                />
            </div>
        </div>
    );
};

export default GitTerminal;
