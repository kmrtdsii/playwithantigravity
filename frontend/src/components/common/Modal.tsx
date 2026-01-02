import React, { useEffect, useRef, useCallback } from 'react';
import styles from './Modal.module.css';

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

/**
 * Modal component with focus trapping for accessibility.
 * Uses the native <dialog> element for proper modal behavior.
 */
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
    const previousActiveElement = useRef<Element | null>(null);

    // Focus trap: get all focusable elements inside the modal
    const getFocusableElements = useCallback((): HTMLElement[] => {
        if (!dialogRef.current) return [];
        return Array.from(
            dialogRef.current.querySelectorAll<HTMLElement>(
                'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
            )
        ).filter(el => !el.hasAttribute('disabled') && el.offsetParent !== null);
    }, []);

    // Handle tab key for focus trap
    const handleKeyDown = useCallback((e: KeyboardEvent) => {
        if (e.key !== 'Tab') return;

        const focusableElements = getFocusableElements();
        if (focusableElements.length === 0) return;

        const firstElement = focusableElements[0];
        const lastElement = focusableElements[focusableElements.length - 1];

        if (e.shiftKey) {
            // Shift+Tab: if on first element, move to last
            if (document.activeElement === firstElement) {
                e.preventDefault();
                lastElement.focus();
            }
        } else {
            // Tab: if on last element, move to first
            if (document.activeElement === lastElement) {
                e.preventDefault();
                firstElement.focus();
            }
        }
    }, [getFocusableElements]);

    useEffect(() => {
        const dialog = dialogRef.current;
        if (!dialog) return;

        if (isOpen) {
            // Save currently focused element
            previousActiveElement.current = document.activeElement;

            dialog.showModal();
            document.body.style.overflow = 'hidden';

            // Focus first focusable element
            const focusableElements = getFocusableElements();
            if (focusableElements.length > 0) {
                focusableElements[0].focus();
            }

            // Add focus trap listener
            dialog.addEventListener('keydown', handleKeyDown);
        } else {
            dialog.close();
            document.body.style.overflow = 'unset';

            // Restore focus to previously focused element
            if (previousActiveElement.current instanceof HTMLElement) {
                previousActiveElement.current.focus();
            }
        }

        return () => {
            document.body.style.overflow = 'unset';
            dialog.removeEventListener('keydown', handleKeyDown);
            if (dialog.open) dialog.close();
        };
    }, [isOpen, getFocusableElements, handleKeyDown]);

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

    const getSizeClass = () => {
        switch (size) {
            case 'fullscreen': return styles.sizeFullscreen;
            case 'large': return styles.sizeLarge;
            default: return styles.sizeDefault;
        }
    };

    const dialogClassName = [
        styles.dialog,
        getSizeClass(),
        resizable ? styles.resizable : ''
    ].filter(Boolean).join(' ');

    return (
        <dialog
            ref={dialogRef}
            onCancel={handleCancel}
            onClick={handleClick}
            className={dialogClassName}
            aria-labelledby="modal-title"
            aria-modal="true"
        >
            <div className={styles.dialogInner}>
                <div className={styles.header}>
                    <div id="modal-title" className={styles.title}>{title}</div>
                    {!hideCloseButton && (
                        <button
                            onClick={onClose}
                            className={styles.closeButton}
                            aria-label="Close modal"
                        >
                            ✕
                        </button>
                    )}
                </div>
                {children}
                {resizable && (
                    <div className={styles.resizeHandle} aria-hidden="true">
                        ⋮⋮
                    </div>
                )}
            </div>
        </dialog>
    );
};

export default Modal;
