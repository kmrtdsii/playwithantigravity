import React, { useState, useEffect } from 'react';
import { Search, X } from 'lucide-react';

interface SearchBarProps {
    value: string;
    onChange: (value: string) => void;
    placeholder?: string;
}

const SearchBar: React.FC<SearchBarProps> = ({ value, onChange, placeholder = "Search..." }) => {
    const [localValue, setLocalValue] = useState(value);
    const inputRef = React.useRef<HTMLInputElement>(null);

    useEffect(() => {
        setLocalValue(value);
    }, [value]);

    useEffect(() => {
        const timeoutId = setTimeout(() => {
            onChange(localValue);
        }, 200); // 200ms debounce
        return () => clearTimeout(timeoutId);
    }, [localValue, onChange]);

    return (
        <div style={{
            display: 'flex',
            alignItems: 'center',
            width: '100%',
            height: '28px',
            background: 'var(--bg-secondary)',
            border: '1px solid var(--border-subtle)',
            borderRadius: '4px',
            padding: '0 8px',
            transition: 'border-color 0.2s',
            boxShadow: '0 2px 6px rgba(0,0,0,0.1)'
        }}>
            <Search size={14} color="var(--text-tertiary)" style={{ marginRight: '6px' }} />
            <input
                ref={inputRef}
                type="text"
                value={localValue}
                onChange={(e) => setLocalValue(e.target.value)}
                placeholder={placeholder}
                autoFocus
                style={{
                    background: 'transparent',
                    border: 'none',
                    outline: 'none',
                    fontSize: '11px',
                    color: 'var(--text-primary)',
                    flex: 1,
                    fontFamily: 'inherit'
                }}
            />
            {localValue && (
                <button
                    onClick={() => {
                        setLocalValue('');
                        onChange('');
                        inputRef.current?.focus();
                    }}
                    style={{
                        background: 'none',
                        border: 'none',
                        padding: 0,
                        cursor: 'pointer',
                        display: 'flex',
                        alignItems: 'center',
                        marginLeft: '4px'
                    }}
                >
                    <X size={12} color="var(--text-tertiary)" />
                </button>
            )}
        </div>
    );
};

export default SearchBar;
