import React, { useState, useEffect } from 'react';
import { Search, X } from 'lucide-react';

interface SearchBarProps {
    value: string;
    onChange: (value: string) => void;
    placeholder?: string;
}

const SearchBar: React.FC<SearchBarProps> = ({ value, onChange, placeholder = "Search..." }) => {
    const [localValue, setLocalValue] = useState(value);

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
            position: 'relative',
            display: 'flex',
            alignItems: 'center',
            width: '200px',
            height: '24px',
            background: 'var(--bg-secondary)',
            border: '1px solid var(--border-subtle)',
            borderRadius: '4px',
            padding: '0 8px',
            transition: 'border-color 0.2s'
        }}>
            <Search size={12} color="var(--text-tertiary)" />
            <input
                type="text"
                value={localValue}
                onChange={(e) => setLocalValue(e.target.value)}
                placeholder={placeholder}
                style={{
                    background: 'transparent',
                    border: 'none',
                    outline: 'none',
                    fontSize: '11px',
                    color: 'var(--text-primary)',
                    flex: 1,
                    marginLeft: '6px',
                    fontFamily: 'inherit'
                }}
            />
            {localValue && (
                <button
                    onClick={() => {
                        setLocalValue('');
                        onChange('');
                    }}
                    style={{
                        background: 'none',
                        border: 'none',
                        padding: 0,
                        cursor: 'pointer',
                        display: 'flex',
                        alignItems: 'center'
                    }}
                >
                    <X size={12} color="var(--text-tertiary)" />
                </button>
            )}
        </div>
    );
};

export default SearchBar;
