import React, { useEffect, useRef } from 'react';

interface ModalProps {
    isOpen: boolean;
    onClose: () => void;
    title: string;
    children: React.ReactNode;
}



const modalContentStyle: React.CSSProperties = {
    backgroundColor: 'var(--bg-secondary)',
    border: '1px solid var(--border-subtle)',
    borderRadius: '8px',
    padding: '24px',
    minWidth: '320px',
    maxWidth: '500px',
    boxShadow: '0 4px 24px rgba(0, 0, 0, 0.2)',
    display: 'flex',
    flexDirection: 'column',
    gap: '16px',
    color: 'var(--text-primary)',
};

const modalHeaderStyle: React.CSSProperties = {
    fontSize: '18px',
    fontWeight: 600,
    marginBottom: '8px',
};

const Modal: React.FC<ModalProps> = ({ isOpen, onClose, title, children }) => {
    const dialogRef = useRef<HTMLDialogElement>(null);

    useEffect(() => {
        const dialog = dialogRef.current;
        if (!dialog) return;

        if (isOpen) {
            dialog.showModal();
            // Lock body scroll
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

    // Handle Closing via Backdrop click or Escape
    const handleCancel = (e: React.SyntheticEvent<HTMLDialogElement, Event>) => {
        e.preventDefault();
        onClose();
    };

    const handleClick = (e: React.MouseEvent<HTMLDialogElement>) => {
        const dialog = dialogRef.current;
        if (dialog) {
            const rect = dialog.getBoundingClientRect();
            // Check if click is outside the dialog (on the backdrop)
            const isInDialog = (rect.top <= e.clientY && e.clientY <= rect.top + rect.height &&
                rect.left <= e.clientX && e.clientX <= rect.left + rect.width);

            if (!isInDialog) {
                onClose();
            }
        }
    };

    return (
        <dialog
            ref={dialogRef}
            onCancel={handleCancel}
            onClick={handleClick}
            style={{
                ...modalContentStyle,
                margin: 'auto', // Center natively
                position: 'fixed', // Ensure fixed positioning even for dialog
                zIndex: 1000
                // Backdrop is handled by ::backdrop pseudo-element usually, but we keep content style
            }}
            aria-labelledby="modal-title"
        >
            <div id="modal-title" style={modalHeaderStyle}>{title}</div>
            {children}
            {/* Native backdrop styling injected via style tag or global css if needed, 
                but browser default is usually passable (dimmed). 
                Ideally we add ::backdrop to global CSS. */}
        </dialog>
    );
};

export default Modal;
