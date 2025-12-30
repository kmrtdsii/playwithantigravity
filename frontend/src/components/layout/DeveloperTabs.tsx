import React from 'react';
import { Monitor, X, Plus } from 'lucide-react';
import './DeveloperTabs.css';

interface DeveloperTabsProps {
    developers: string[];
    activeDeveloper: string;
    onSwitchDeveloper: (name: string) => void;
    onAddDeveloper: () => void;
    onRemoveDeveloper?: (name: string) => void;
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
    onRemoveDeveloper,
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
        <div className="developer-tabs-container" data-testid="developer-tabs">
            <div
                className="tab-container"
                role="tablist"
                aria-label="Developer tabs"
                data-testid="tab-list"
                ref={tabListRef}
            >
                {developers.map((dev, index) => {
                    const isActive = dev === activeDeveloper;
                    const isRemovable = dev !== 'Alice' && dev !== 'Bob';

                    return (
                        <div key={dev} className={`user-tab-wrapper ${isActive ? 'active' : ''}`}>
                            <button
                                role="tab"
                                aria-selected={isActive}
                                aria-controls={`panel-${dev}`}
                                id={`tab-${dev}`}
                                tabIndex={isActive ? 0 : -1}
                                onClick={() => onSwitchDeveloper(dev)}
                                onKeyDown={(e) => handleKeyDown(e, index)}
                                className={`user-tab ${isActive ? 'active' : ''}`}
                                data-testid={`tab-${dev}`}
                            >
                                <Monitor size={14} />
                                {dev}
                            </button>
                            {isRemovable && (
                                <button
                                    onClick={(e) => {
                                        e.stopPropagation();
                                        if (onRemoveDeveloper) onRemoveDeveloper(dev);
                                    }}
                                    className="tab-close-btn"
                                    aria-label={`Remove ${dev}`}
                                    title={`Remove ${dev}`}
                                    data-testid={`remove-tab-${dev}`}
                                >
                                    <X size={14} />
                                </button>
                            )}
                        </div>
                    );
                })}
            </div>
            <button
                className="tab-add-btn"
                title="Add Developer"
                aria-label="Add new developer"
                onClick={onAddDeveloper}
                data-testid="add-developer-btn"
            >
                <Plus size={18} />
            </button>
        </div>
    );
};

export default DeveloperTabs;
