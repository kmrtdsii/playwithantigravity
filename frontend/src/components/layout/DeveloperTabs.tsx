import React from 'react';

interface DeveloperTabsProps {
    developers: string[];
    activeDeveloper: string;
    onSwitchDeveloper: (name: string) => void;
    onAddDeveloper: () => void;
}

/**
 * Tab bar component for switching between developer sessions.
 * Extracted from AppLayout for better separation of concerns.
 */
const DeveloperTabs: React.FC<DeveloperTabsProps> = ({
    developers,
    activeDeveloper,
    onSwitchDeveloper,
    onAddDeveloper,
}) => {
    const tabListRef = React.useRef<HTMLDivElement>(null);

    const handleKeyDown = (e: React.KeyboardEvent, index: number) => {
        if (e.key === 'ArrowRight') {
            const nextIndex = (index + 1) % developers.length;
            onSwitchDeveloper(developers[nextIndex]);
            (tabListRef.current?.children[nextIndex] as HTMLElement)?.focus();
        } else if (e.key === 'ArrowLeft') {
            const prevIndex = (index - 1 + developers.length) % developers.length;
            onSwitchDeveloper(developers[prevIndex]);
            (tabListRef.current?.children[prevIndex] as HTMLElement)?.focus();
        } else if (e.key === 'Home') {
            onSwitchDeveloper(developers[0]);
            (tabListRef.current?.children[0] as HTMLElement)?.focus();
        } else if (e.key === 'End') {
            onSwitchDeveloper(developers[developers.length - 1]);
            (tabListRef.current?.children[developers.length - 1] as HTMLElement)?.focus();
        }
    };

    return (
        <div
            className="developer-tabs-container"
            style={{
                height: '32px',
                background: 'var(--bg-secondary)',
                display: 'flex',
                alignItems: 'flex-end',
                borderBottom: 'none',
            }}
        >
            <div
                className="tab-container"
                role="tablist"
                aria-label="Developer tabs"
                ref={tabListRef}
                style={{ display: 'flex', height: '100%', alignItems: 'flex-end' }}
            >
                {developers.map((dev, index) => {
                    const isActive = dev === activeDeveloper;
                    return (
                        <button
                            key={dev}
                            role="tab"
                            aria-selected={isActive}
                            aria-controls={`panel-${dev}`}
                            id={`tab-${dev}`}
                            tabIndex={isActive ? 0 : -1}
                            onClick={() => onSwitchDeveloper(dev)}
                            onKeyDown={(e) => handleKeyDown(e, index)}
                            className={`user-tab ${isActive ? 'active' : ''}`}
                        >
                            <span>🏠 {dev}</span>
                        </button>
                    );
                })}
            </div>
            <button
                className="tab-add-btn"
                title="Add Developer"
                aria-label="Add new developer"
                onClick={onAddDeveloper}
            >
                +
            </button>
        </div>
    );
};

export default DeveloperTabs;
