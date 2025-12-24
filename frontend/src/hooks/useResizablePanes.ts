import { useState, useRef, useCallback, useEffect } from 'react';

export const useResizablePanes = () => {
    // --- Layout State ---
    const [leftPaneWidth, setLeftPaneWidth] = useState(33); // Percentage
    const [vizHeight, setVizHeight] = useState(500); // Height of Top Graph in Center
    const [remoteGraphHeight, setRemoteGraphHeight] = useState(500); // Height of Top Graph in Left

    const containerRef = useRef<HTMLDivElement>(null);
    const centerContentRef = useRef<HTMLDivElement>(null);
    const stackContainerRef = useRef<HTMLDivElement>(null); // Parent of graph + bottom
    const leftContentRef = useRef<HTMLDivElement>(null);

    // Resize Refs
    const isResizingLeft = useRef(false); // Between Left & Main
    const isResizingCenterVertical = useRef(false); // Center Pane Split
    const isResizingLeftVertical = useRef(false); // Left Pane Split

    // --- Resize Handlers ---

    const startResizeLeft = useCallback(() => { isResizingLeft.current = true; document.body.style.cursor = 'col-resize'; document.body.style.userSelect = 'none'; }, []);
    const startResizeCenterVert = useCallback(() => { isResizingCenterVertical.current = true; document.body.style.cursor = 'row-resize'; document.body.style.userSelect = 'none'; }, []);
    const startResizeLeftVert = useCallback(() => { isResizingLeftVertical.current = true; document.body.style.cursor = 'row-resize'; document.body.style.userSelect = 'none'; }, []);

    const stopResizing = useCallback(() => {
        isResizingLeft.current = false;
        isResizingCenterVertical.current = false;
        isResizingLeftVertical.current = false;
        document.body.style.cursor = '';
        document.body.style.userSelect = '';
    }, []);

    const resize = useCallback((e: MouseEvent) => {
        if (!containerRef.current) return;
        const containerRect = containerRef.current.getBoundingClientRect();

        // 1. Resize Left Pane
        if (isResizingLeft.current) {
            const newLeftPct = ((e.clientX - containerRect.left) / containerRect.width) * 100;
            if (newLeftPct > 10 && newLeftPct < 90) {
                setLeftPaneWidth(newLeftPct);
            }
        }

        // 2. Vertical Resize (Center Pane) - use stack container for full height
        if (isResizingCenterVertical.current && stackContainerRef.current) {
            const rect = stackContainerRef.current.getBoundingClientRect();
            const newH = e.clientY - rect.top;
            const minHeight = 100;
            const maxHeight = rect.height - 100; // Leave room for bottom panel
            if (newH >= minHeight && newH <= maxHeight) setVizHeight(newH);
        }

        // 3. Vertical Resize (Left Pane)
        if (isResizingLeftVertical.current && leftContentRef.current) {
            const rect = leftContentRef.current.getBoundingClientRect();
            const newH = e.clientY - rect.top;
            if (newH > 100 && newH < rect.height - 100) setRemoteGraphHeight(newH);
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

    return {
        leftPaneWidth,
        vizHeight,
        remoteGraphHeight,
        containerRef,
        centerContentRef,
        stackContainerRef,
        leftContentRef,
        startResizeLeft,
        startResizeCenterVert,
        startResizeLeftVert
    };
};
