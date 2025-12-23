import React from 'react';

export type ResizerOrientation = 'horizontal' | 'vertical';

interface ResizerProps {
    /**
     * Orientation of the resizer
     * - 'horizontal': For row-resize (splits top/bottom)
     * - 'vertical': For col-resize (splits left/right)
     */
    orientation: ResizerOrientation;

    /**
     * Callback when user starts dragging the resizer
     */
    onMouseDown: () => void;

    /**
     * Optional additional class name
     */
    className?: string;
}

/**
 * Resizer component for draggable panel dividers.
 * 
 * Uses consistent styling from AppLayout.css classes:
 * - .resizer (horizontal)
 * - .resizer-vertical (vertical)
 */
const Resizer: React.FC<ResizerProps> = ({ orientation, onMouseDown, className }) => {
    const baseClassName = orientation === 'horizontal' ? 'resizer' : 'resizer-vertical';
    const combinedClassName = className ? `${baseClassName} ${className}` : baseClassName;

    return (
        <div
            className={combinedClassName}
            onMouseDown={onMouseDown}
            role="separator"
            aria-orientation={orientation}
        />
    );
};

export default Resizer;
