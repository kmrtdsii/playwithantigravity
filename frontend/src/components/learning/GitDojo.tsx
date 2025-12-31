import React, { useState, useMemo } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { useTranslation } from 'react-i18next';
import { useDojo } from '../../context/DojoContext';
import { DOJO_PROBLEMS, isLocked, type DojoProblem, type DojoCategory } from '../../data/dojoProblems';
import { Modal } from '../common';
import DojoChallenge from './DojoChallenge';
import DojoResult from './DojoResult';
import './GitDojo.css';

interface GitDojoProps {
    isOpen: boolean;
    onClose: () => void;
    onOpen: () => void;
}

type ViewMode = 'level' | 'command';

const GitDojo: React.FC<GitDojoProps> = ({ isOpen, onClose, onOpen }) => {
    const { t } = useTranslation('common');
    const { state, startChallenge, isCompleted, setOpenModalCallback } = useDojo();
    const [viewMode, setViewMode] = useState<ViewMode>('level');
    const [selectedCommand, setSelectedCommand] = useState<string | null>(null);

    // Register the modal open callback
    React.useEffect(() => {
        setOpenModalCallback(onOpen);
    }, [setOpenModalCallback, onOpen]);

    const categories: { key: DojoCategory; icon: string; label: string }[] = [
        { key: 'basic', icon: '📦', label: t('dojo.category.basic') },
        { key: 'intermediate', icon: '🚀', label: t('dojo.category.intermediate') },
        { key: 'advanced', icon: '⚡', label: t('dojo.category.advanced') },
    ];

    const completedCount = state.completedProblemIds.length;
    const totalCount = DOJO_PROBLEMS.length;

    // Get all unique commands
    const allCommands = useMemo(() => {
        const cmds = new Set<string>();
        DOJO_PROBLEMS.forEach(p => {
            p.commands?.forEach(c => cmds.add(c));
        });
        return Array.from(cmds).sort();
    }, []);

    const getDifficultyStars = (difficulty: number) => {
        return '⭐'.repeat(difficulty) + '☆'.repeat(3 - difficulty);
    };

    const handleStartProblem = (problem: DojoProblem) => {
        if (!isLocked(problem, state.completedProblemIds)) {
            startChallenge(problem.id);
        }
    };

    const handleRandomChallenge = () => {
        // Find unlocked and uncompleted problems first
        let candidates = DOJO_PROBLEMS.filter(p => !isLocked(p, state.completedProblemIds) && !isCompleted(p.id));

        // If all completed, use any unlocked
        if (candidates.length === 0) {
            candidates = DOJO_PROBLEMS.filter(p => !isLocked(p, state.completedProblemIds));
        }

        if (candidates.length > 0) {
            const random = candidates[Math.floor(Math.random() * candidates.length)];
            startChallenge(random.id);
        }
    };

    const renderProblemCard = (problem: DojoProblem) => {
        const locked = isLocked(problem, state.completedProblemIds);
        const completed = isCompleted(problem.id);

        return (
            <motion.button
                key={problem.id}
                className={`problem-card ${locked ? 'locked' : ''} ${completed ? 'completed' : ''}`}
                onClick={() => handleStartProblem(problem)}
                disabled={locked}
                whileHover={locked ? {} : { scale: 1.02 }}
                whileTap={locked ? {} : { scale: 0.98 }}
            >
                <div className="problem-status">
                    {locked ? '🔒' : completed ? '✅' : '○'}
                </div>
                <div className="problem-info">
                    <span className="problem-id">#{problem.id}</span>
                    <span className="problem-title">{t(problem.title)}</span>
                </div>
                <div className="problem-meta">
                    <span className="problem-difficulty">
                        {getDifficultyStars(problem.difficulty)}
                    </span>
                    <span className="problem-time">
                        ~{problem.estimatedMinutes}{t('dojo.minutes')}
                    </span>
                </div>
            </motion.button>
        );
    };

    const renderLevelView = () => (
        <div className="dojo-categories">
            {categories.map(cat => {
                const problems = DOJO_PROBLEMS.filter(p => p.category === cat.key);
                if (problems.length === 0) return null;

                return (
                    <div key={cat.key} className="dojo-category">
                        <div className="category-header">
                            <span className="category-icon">{cat.icon}</span>
                            <span className="category-label">{cat.label}</span>
                        </div>

                        <div className="problem-list">
                            {problems.map(renderProblemCard)}
                        </div>
                    </div>
                );
            })}
        </div>
    );

    const renderCommandView = () => {
        const filteredProblems = selectedCommand
            ? DOJO_PROBLEMS.filter(p => p.commands?.includes(selectedCommand))
            : DOJO_PROBLEMS;

        return (
            <div className="dojo-categories">
                <div className="command-filter-container">
                    <div className="command-chips">
                        <button
                            className={`command-chip ${!selectedCommand ? 'active' : ''}`}
                            onClick={() => setSelectedCommand(null)}
                        >
                            ALL
                        </button>
                        {allCommands.map(cmd => (
                            <button
                                key={cmd}
                                className={`command-chip ${selectedCommand === cmd ? 'active' : ''}`}
                                onClick={() => setSelectedCommand(cmd)}
                            >
                                {cmd}
                            </button>
                        ))}
                    </div>
                </div>

                <div className="flat-problem-list">
                    {filteredProblems.map(renderProblemCard)}
                </div>
            </div>
        );
    };

    const renderProblemList = () => (
        <div className="dojo-list">
            <div className="dojo-header">
                <h2 className="dojo-title">
                    <span className="dojo-icon">🥋</span>
                    {t('dojo.title')}
                </h2>
                <div className="dojo-header-actions">
                    <motion.button
                        className="random-button"
                        onClick={handleRandomChallenge}
                        whileHover={{ scale: 1.05 }}
                        whileTap={{ scale: 0.95 }}
                    >
                        🎲 {t('dojo.random')}
                    </motion.button>
                </div>
            </div>

            <div className="dojo-view-tabs">
                <button
                    className={`view-tab ${viewMode === 'level' ? 'active' : ''}`}
                    onClick={() => setViewMode('level')}
                >
                    {t('dojo.view.level')}
                </button>
                <button
                    className={`view-tab ${viewMode === 'command' ? 'active' : ''}`}
                    onClick={() => setViewMode('command')}
                >
                    {t('dojo.view.command')}
                </button>
            </div>

            {viewMode === 'level' ? renderLevelView() : renderCommandView()}

            <div className="dojo-progress">
                <div className="progress-bar">
                    <div
                        className="progress-fill"
                        style={{ width: `${(completedCount / totalCount) * 100}%` }}
                    />
                </div>
                <span className="progress-text">
                    📊 {t('dojo.progress')}: {completedCount}/{totalCount} {t('dojo.cleared')}
                </span>
            </div>
        </div>
    );

    // Render based on phase
    const renderContent = () => {
        switch (state.phase) {
            case 'challenge':
                return <DojoChallenge onStartAndClose={onClose} />;
            case 'result':
                return <DojoResult />;
            default:
                return renderProblemList();
        }
    };

    return (
        <Modal
            isOpen={isOpen}
            onClose={onClose}
            title=""
            size="fullscreen"
            hideCloseButton
        >
            <AnimatePresence mode="wait">
                <motion.div
                    key={state.phase}
                    initial={{ opacity: 0, y: 20 }}
                    animate={{ opacity: 1, y: 0 }}
                    exit={{ opacity: 0, y: -20 }}
                    transition={{ duration: 0.3 }}
                    className="dojo-content"
                >
                    {renderContent()}
                </motion.div>
            </AnimatePresence>
        </Modal>
    );
};

export default GitDojo;
