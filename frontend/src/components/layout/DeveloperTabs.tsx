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
                style={{ display: 'flex', height: '100%', alignItems: 'flex-end' }}
            >
                {developers.map((dev) => {
                    const isActive = dev === activeDeveloper;
                    return (
                        <button
                            key={dev}
                            onClick={() => onSwitchDeveloper(dev)}
                            className={`user-tab ${isActive ? 'active' : ''}`}
                        >
                            <span>{dev}</span>
                        </button>
                    );
                })}
                <button
                    className="tab-add-btn"
                    title="Add Developer"
                    onClick={onAddDeveloper}
                >
                    +
                </button>
            </div>
        </div>
    );
};

export default DeveloperTabs;
