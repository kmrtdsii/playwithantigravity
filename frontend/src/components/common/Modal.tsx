import React, { useEffect, useRef } from 'react';

interface ModalProps {
    isOpen: boolean;
    onClose: () => void;
    title: string;
    children: React.ReactNode;
    size?: 'default' | 'large' | 'fullscreen';
    resizable?: boolean;
    disableBackdropClose?: boolean;
    hideCloseButton?: boolean;
}

// Visual styles for the dialog box itself
const dialogStyle: React.CSSProperties = {
    backgroundColor: 'var(--bg-secondary)',
    border: '1px solid var(--border-subtle)',
    borderRadius: '8px',
    padding: '0',
    minWidth: '320px',
    boxShadow: '0 4px 24px rgba(0, 0, 0, 0.2)',
    color: 'var(--text-primary)',
};

// Internal layout (Flex)
const dialogInnerStyle: React.CSSProperties = {
    padding: '12px',
    display: 'flex',
    flexDirection: 'column',
    gap: '8px',
    height: '100%',
    boxSizing: 'border-box',
    position: 'relative',
};

const modalHeaderStyle: React.CSSProperties = {
    fontSize: '14px',
    fontWeight: 600,
    marginBottom: '4px',
};

const Modal: React.FC<ModalProps> = ({
    isOpen,
    onClose,
    title,
    children,
    size = 'default',
    resizable = false,
    disableBackdropClose = false,
    hideCloseButton = false
}) => {
    const dialogRef = useRef<HTMLDialogElement>(null);

    const getSizeStyles = (): React.CSSProperties => {
        if (size === 'fullscreen') {
            return { width: '700px', height: '800px', maxWidth: '95vw', maxHeight: '95vh' };
        }
        if (size === 'large') {
            return { maxWidth: '90vw', width: '900px' };
        }
        return {};
    };

    useEffect(() => {
        const dialog = dialogRef.current;
        if (!dialog) return;

        if (isOpen) {
            dialog.showModal();
            document.body.style.overflow = 'hidden';
        } else {
            dialog.close();
            document.body.style.overflow = 'unset';
        }

        return () => {
            document.body.style.overflow = 'unset';
            if (dialog?.open) dialog.close();
        };
    }, [isOpen]);

    const handleCancel = (e: React.SyntheticEvent<HTMLDialogElement, Event>) => {
        e.preventDefault();
        if (!disableBackdropClose) {
            onClose();
        }
    };

    const handleClick = (e: React.MouseEvent<HTMLDialogElement>) => {
        if (!disableBackdropClose && e.target === dialogRef.current) {
            onClose();
        }
    };

    const resizableStyles: React.CSSProperties = resizable
        ? { resize: 'both', overflow: 'auto', minWidth: '400px', minHeight: '400px' }
        : {};

    return (
        <dialog
            ref={dialogRef}
            onCancel={handleCancel}
            onClick={handleClick}
            style={{
                ...dialogStyle,
                ...getSizeStyles(),
                ...resizableStyles,
                margin: 'auto',
                position: 'fixed',
                zIndex: 1000
            }}
            aria-labelledby="modal-title"
        >
            <div style={dialogInnerStyle}>
                {/* Header with title and close button */}
                <div style={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center',
                    marginBottom: '8px'
                }}>
                    <div id="modal-title" style={modalHeaderStyle}>{title}</div>
                    {!hideCloseButton && (
                        <button
                            onClick={onClose}
                            style={{
                                width: '28px',
                                height: '28px',
                                borderRadius: '6px',
                                border: 'none',
                                background: 'transparent',
                                color: 'var(--text-secondary, #6b7280)',
                                fontSize: '18px',
                                cursor: 'pointer',
                                display: 'flex',
                                alignItems: 'center',
                                justifyContent: 'center',
                                transition: 'all 0.15s ease'
                            }}
                            onMouseEnter={(e) => {
                                e.currentTarget.style.background = '#ef4444';
                                e.currentTarget.style.color = 'white';
                            }}
                            onMouseLeave={(e) => {
                                e.currentTarget.style.background = 'transparent';
                                e.currentTarget.style.color = 'var(--text-secondary, #6b7280)';
                            }}
                        >
                            ✕
                        </button>
                    )}
                </div>
                {children}
                {/* Resize handle indicator */}
                {resizable && (
                    <div
                        style={{
                            position: 'absolute',
                            bottom: '4px',
                            right: '4px',
                            width: '20px',
                            height: '20px',
                            cursor: 'se-resize',
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            color: '#9ca3af',
                            fontSize: '14px',
                            pointerEvents: 'none'
                        }}
                    >
                        ⋮⋮
                    </div>
                )}
            </div>
        </dialog>
    );
};

export default Modal;
