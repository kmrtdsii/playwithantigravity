import React, { useState, useRef, useCallback, useEffect } from 'react';
import FileExplorer from './FileExplorer';
import GitTerminal from '../terminal/GitTerminal';
import { Resizer } from '../common';
import type { SelectedObject } from '../../types/layoutTypes';

interface BottomPanelProps {
    onSelect: (obj: SelectedObject) => void;
}

const BottomPanel: React.FC<BottomPanelProps> = ({ onSelect }) => {
    // Layout State
    const [explorerWidth, setExplorerWidth] = useState(40); // Percentage

    // Resize State & Refs
    const containerRef = useRef<HTMLDivElement>(null);
    const isResizing = useRef(false);

    const startResize = useCallback(() => {
        isResizing.current = true;
        document.body.style.cursor = 'col-resize';
        document.body.style.userSelect = 'none';
    }, []);

    const stopResizing = useCallback(() => {
        isResizing.current = false;
        document.body.style.cursor = '';
        document.body.style.userSelect = '';
    }, []);

    const resize = useCallback((e: MouseEvent) => {
        if (!isResizing.current || !containerRef.current) return;

        const rect = containerRef.current.getBoundingClientRect();
        const newWidthPct = ((e.clientX - rect.left) / rect.width) * 100;

        // Constrain width (15% - 85%)
        if (newWidthPct > 15 && newWidthPct < 85) {
            setExplorerWidth(newWidthPct);
        }
    }, []);

    useEffect(() => {
        window.addEventListener('mousemove', resize);
        window.addEventListener('mouseup', stopResizing);
        return () => {
            window.removeEventListener('mousemove', resize);
            window.removeEventListener('mouseup', stopResizing);
        };
    }, [resize, stopResizing]);

    return (
        <div ref={containerRef} style={{ flex: 1, minHeight: 0, display: 'flex' }}>
            {/* File Explorer Panel */}
            <div style={{
                width: `${explorerWidth}%`,
                display: 'flex',
                flexDirection: 'column',
                borderRight: '1px solid var(--border-subtle)',
                minWidth: 0
            }}>
                <FileExplorer onSelect={onSelect} />
            </div>

            {/* Resize Handle */}
            <Resizer orientation="vertical" onMouseDown={startResize} />

            {/* Terminal Panel */}
            <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
                <GitTerminal />
            </div>
        </div>
    );
};

export default BottomPanel;
