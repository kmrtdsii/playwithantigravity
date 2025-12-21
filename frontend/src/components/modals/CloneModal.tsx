import { useState, useRef, useEffect } from 'react';
import './CloneModal.css';

interface CloneModalProps {
    isOpen: boolean;
    onClose: () => void;
    onClone: (url: string) => Promise<void>;
}

const CloneModal = ({ isOpen, onClose, onClone }: CloneModalProps) => {
    const [url, setUrl] = useState('');
    const [isLoading, setIsLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const inputRef = useRef<HTMLInputElement>(null);

    useEffect(() => {
        if (isOpen && inputRef.current) {
            inputRef.current.focus();
        }
    }, [isOpen]);

    if (!isOpen) return null;

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setError(null);

        if (!url.trim()) {
            setError('Please enter a valid URL');
            return;
        }

        setIsLoading(true);
        try {
            await onClone(url);
            onClose();
        } catch (err: any) {
            setError(err.message || 'Failed to clone repository');
        } finally {
            setIsLoading(false);
        }
    };

    return (
        <div className="modal-overlay" onClick={onClose}>
            <div className="modal-content" onClick={e => e.stopPropagation()}>
                <div className="modal-header">
                    <h2>Clone Repository</h2>
                    <button className="close-button" onClick={onClose}>×</button>
                </div>
                <form onSubmit={handleSubmit}>
                    <div className="modal-body">
                        <label htmlFor="repo-url">Repository URL</label>
                        <input
                            ref={inputRef}
                            id="repo-url"
                            type="text"
                            placeholder="https://github.com/username/repo.git"
                            value={url}
                            onChange={(e) => setUrl(e.target.value)}
                            disabled={isLoading}
                        />
                        {error && <div className="error-message">{error}</div>}
                    </div>
                    <div className="modal-footer">
                        <button type="button" className="btn-secondary" onClick={onClose} disabled={isLoading}>
                            Cancel
                        </button>
                        <button type="submit" className="btn-primary" disabled={isLoading}>
                            {isLoading ? 'Cloning... (this may take a while)' : 'Clone'}
                        </button>
                    </div>
                </form>
            </div>
        </div>
    );
};

export default CloneModal;
