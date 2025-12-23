import React, { useEffect, useRef } from 'react';

interface ModalProps {
    isOpen: boolean;
    onClose: () => void;
    title: string;
    children: React.ReactNode;
}

const modalOverlayStyle: React.CSSProperties = {
    position: 'fixed',
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
    backgroundColor: 'rgba(0, 0, 0, 0.5)',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    zIndex: 1000,
    backdropFilter: 'blur(2px)',
};

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
    const dialogRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        const handleKeyDown = (e: KeyboardEvent) => {
            if (e.key === 'Escape') {
                onClose();
            }
        };

        if (isOpen) {
            document.addEventListener('keydown', handleKeyDown);
            // Lock body scroll
            document.body.style.overflow = 'hidden';
        }

        return () => {
            document.removeEventListener('keydown', handleKeyDown);
            document.body.style.overflow = 'unset';
        };
    }, [isOpen, onClose]);

    if (!isOpen) return null;

    return (
        <div style={modalOverlayStyle} onClick={(e) => {
            if (e.target === e.currentTarget) onClose();
        }}>
            <div
                ref={dialogRef}
                style={modalContentStyle}
                role="dialog"
                aria-modal="true"
                aria-labelledby="modal-title"
            >
                <div id="modal-title" style={modalHeaderStyle}>{title}</div>
                {children}
            </div>
        </div>
    );
};

export default Modal;
