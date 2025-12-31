// Git Dojo Problem Definitions
// Quiz-style learning challenges

export type DojoDifficulty = 1 | 2 | 3;
export type DojoCategory = 'basic' | 'intermediate' | 'advanced';

export interface DojoProblem {
    id: string;
    title: string; // i18n key
    description: string; // i18n key
    category: DojoCategory;
    difficulty: DojoDifficulty;
    estimatedMinutes: number;
    prerequisiteIds: string[];

    // Mission Backend linkage
    // Mission Backend linkage
    missionId: string;

    // Command tags for filtering
    commands: string[];

    // Goals displayed during challenge
    goals: string[]; // i18n keys

    // Solution content (shown after completion)
    solutionSteps: string[];
    trivia: string; // i18n key
}

export const DOJO_PROBLEMS: DojoProblem[] = [
    // === BASIC ===
    {
        id: '101',
        title: 'dojo.problems.101.title',
        description: 'dojo.problems.101.description',
        category: 'basic',
        difficulty: 1,
        estimatedMinutes: 3,
        prerequisiteIds: [],
        missionId: '101-first-commit',
        commands: ['git init', 'git add', 'git commit'],
        goals: [
            'dojo.problems.101.goals.0',
            'dojo.problems.101.goals.1',
        ],
        solutionSteps: [
            'git add README.md',
            'git commit -m "Initial commit"',
        ],
        trivia: 'dojo.problems.101.trivia',
    },
    {
        id: '102',
        title: 'dojo.problems.102.title',
        description: 'dojo.problems.102.description',
        category: 'basic',
        difficulty: 1,
        estimatedMinutes: 3,
        prerequisiteIds: ['101'],
        missionId: '102-create-branch',
        commands: ['git branch', 'git switch', 'git checkout'],
        goals: [
            'dojo.problems.102.goals.0',
            'dojo.problems.102.goals.1',
        ],
        solutionSteps: [
            'git branch feature',
            'git switch feature',
        ],
        trivia: 'dojo.problems.102.trivia',
    },
    {
        id: '103',
        title: 'dojo.problems.103.title',
        description: 'dojo.problems.103.description',
        category: 'basic',
        difficulty: 1,
        estimatedMinutes: 5,
        prerequisiteIds: ['101'],
        missionId: '103-history-check',
        commands: ['git log', 'git show'],
        goals: [
            'dojo.problems.103.goals.0',
            'dojo.problems.103.goals.1',
        ],
        solutionSteps: [
            'git log',
            'git show <hash>',
            'echo "4829" > answer.txt',
        ],
        trivia: 'dojo.problems.103.trivia',
    },
    {
        id: '104',
        title: 'dojo.problems.104.title',
        description: 'dojo.problems.104.description',
        category: 'basic',
        difficulty: 1,
        estimatedMinutes: 3,
        prerequisiteIds: ['101'],
        missionId: '104-amend-commit',
        commands: ['git commit --amend'],
        goals: [
            'dojo.problems.104.goals.0',
        ],
        solutionSteps: [
            'git commit --amend -m "Add feature implementation"',
        ],
        trivia: 'dojo.problems.104.trivia',
    },

    // === INTERMEDIATE (Undo/Fix) ===
    {
        id: '202',
        title: 'dojo.problems.202.title',
        description: 'dojo.problems.202.description',
        category: 'intermediate',
        difficulty: 2,
        estimatedMinutes: 5,
        prerequisiteIds: ['101'],
        missionId: '202-undo-commit',
        commands: ['git reset'],
        goals: [
            'dojo.problems.202.goals.0',
            'dojo.problems.202.goals.1',
        ],
        solutionSteps: [
            'git reset --soft HEAD~1',
        ],
        trivia: 'dojo.problems.202.trivia',
    },
    {
        id: '203',
        title: 'dojo.problems.203.title',
        description: 'dojo.problems.203.description',
        category: 'intermediate',
        difficulty: 2,
        estimatedMinutes: 5,
        prerequisiteIds: ['101'],
        missionId: '203-revert-commit',
        commands: ['git revert'],
        goals: [
            'dojo.problems.203.goals.0',
        ],
        solutionSteps: [
            'git revert HEAD',
        ],
        trivia: 'dojo.problems.203.trivia',
    },
    {
        id: '204',
        title: 'dojo.problems.204.title',
        description: 'dojo.problems.204.description',
        category: 'basic',
        difficulty: 1,
        estimatedMinutes: 3,
        prerequisiteIds: ['101'],
        missionId: '204-restore-file',
        commands: ['git restore'],
        goals: [
            'dojo.problems.204.goals.0',
        ],
        solutionSteps: [
            'git restore config.ini',
        ],
        trivia: 'dojo.problems.204.trivia',
    },

    // === INTERMEDIATE (Branching) ===
    {
        id: '301', // Moved from 103/001
        title: 'dojo.problems.301.title',
        description: 'dojo.problems.301.description',
        category: 'intermediate',
        difficulty: 2,
        estimatedMinutes: 7,
        prerequisiteIds: ['102'],
        missionId: '001-conflict-crisis',
        commands: ['git merge', 'git status'],
        goals: [
            'dojo.problems.301.goals.0',
            'dojo.problems.301.goals.1',
        ],
        solutionSteps: [
            'git status',
            '# Resolve conflict in editor',
            'git add README.md',
            'git commit -m "Resolve conflict"',
        ],
        trivia: 'dojo.problems.301.trivia',
    },
    {
        id: '302',
        title: 'dojo.problems.302.title',
        description: 'dojo.problems.302.description',
        category: 'intermediate',
        difficulty: 3,
        estimatedMinutes: 7,
        prerequisiteIds: ['102'],
        missionId: '302-rebase-basic',
        commands: ['git rebase'],
        goals: [
            'dojo.problems.302.goals.0',
        ],
        solutionSteps: [
            'git checkout feature',
            'git rebase main',
        ],
        trivia: 'dojo.problems.302.trivia',
    },
    {
        id: '303',
        title: 'dojo.problems.303.title',
        description: 'dojo.problems.303.description',
        category: 'intermediate',
        difficulty: 2,
        estimatedMinutes: 5,
        prerequisiteIds: ['102'],
        missionId: '303-cherry-pick',
        commands: ['git cherry-pick'],
        goals: [
            'dojo.problems.303.goals.0',
        ],
        solutionSteps: [
            'git log experimental',
            'git cherry-pick <hash>',
        ],
        trivia: 'dojo.problems.303.trivia',
    },
];

// Helper functions
export const getProblemById = (id: string): DojoProblem | undefined => {
    return DOJO_PROBLEMS.find(p => p.id === id);
};

export const getProblemsByCategory = (category: DojoCategory): DojoProblem[] => {
    return DOJO_PROBLEMS.filter(p => p.category === category);
};

export const getAvailableProblems = (completedIds: string[]): DojoProblem[] => {
    return DOJO_PROBLEMS.filter(p => {
        // All prerequisites must be completed
        return p.prerequisiteIds.every(prereq => completedIds.includes(prereq));
    });
};

export const isLocked = (problem: DojoProblem, completedIds: string[]): boolean => {
    return !problem.prerequisiteIds.every(prereq => completedIds.includes(prereq));
};
