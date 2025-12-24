import React, { useState } from 'react';
import { Modal } from '../common';

interface AddDeveloperModalProps {
    isOpen: boolean;
    onClose: () => void;
    onAddDeveloper: (name: string) => void;
}

const AddDeveloperModal: React.FC<AddDeveloperModalProps> = ({ isOpen, onClose, onAddDeveloper }) => {
    const [newDevName, setNewDevName] = useState('');

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault();
        if (newDevName && newDevName.trim()) {
            onAddDeveloper(newDevName.trim());
            setNewDevName('');
            onClose();
        }
    };

    return (
        <Modal
            isOpen={isOpen}
            onClose={onClose}
            title="Add New Developer"
        >
            <form
                onSubmit={handleSubmit}
                style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}
            >
                <input
                    name="name"
                    value={newDevName}
                    onChange={(e) => setNewDevName(e.target.value)}
                    placeholder="Enter developer name (e.g., Alice)"
                    autoFocus
                    style={{
                        padding: '8px 12px',
                        borderRadius: '4px',
                        border: '1px solid var(--border-subtle)',
                        background: 'var(--bg-primary)',
                        color: 'var(--text-primary)',
                        fontSize: '14px'
                    }}
                />
                <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '8px' }}>
                    <button
                        type="button"
                        onClick={onClose}
                        style={{
                            padding: '8px 16px',
                            borderRadius: '4px',
                            border: '1px solid var(--border-subtle)',
                            background: 'transparent',
                            color: 'var(--text-secondary)',
                            cursor: 'pointer'
                        }}
                    >
                        Cancel
                    </button>
                    <button
                        type="submit"
                        style={{
                            padding: '8px 16px',
                            borderRadius: '4px',
                            border: 'none',
                            background: 'var(--accent-primary)',
                            color: 'white',
                            cursor: 'pointer',
                            fontWeight: 500
                        }}
                    >
                        Add Developer
                    </button>
                </div>
            </form>
        </Modal>
    );
};

export default AddDeveloperModal;
