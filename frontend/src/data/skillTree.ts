export interface SkillCommand {
    id: string;
    name: string;
    description: string;
    disabled?: boolean;
}

export interface SkillLevel {
    id: string;
    name: string;
    description: string;
    color: string;
    textColor: string;
    commands: SkillCommand[];
}

export const SKILL_TREE: SkillLevel[] = [
    {
        id: 'basic',
        name: 'Git Basic',
        description: 'The Essentials: Single-branch workflow',
        color: '#ffffff',
        textColor: '#000000',
        commands: [
            { id: 'init', name: 'git init', description: 'Initialize a new repository. (Usually handled by "git clone" or system setup)', disabled: true },
            { id: 'clone', name: 'git clone', description: 'Clone a repository. Try "git clone <url> <dir>" to specify a directory name.' },
            { id: 'status', name: 'git status', description: 'Check which files are changed. Pro tip: Use "git status -sb" for a cleaner view.' },
            { id: 'help', name: 'git help', description: 'Display help. Try "git help <command>" to see practical examples.' },
            { id: 'add', name: 'git add', description: 'Stage changes. Use "git add ." for everything or "git add <file>" for precision.' },
            { id: 'commit', name: 'git commit', description: 'Save changes history. Pro tip: "git commit --amend" fixes the last commit.' },
            { id: 'push', name: 'git push', description: 'Upload to remote. Note: GitGym simulates this safely.' },
            { id: 'pull', name: 'git pull', description: 'Fetch and merge. Try "git pull --rebase" to keep history linear.' },
        ]
    },
    {
        id: 'intermediate',
        name: 'Git Intermediate',
        description: 'Branching & Exploration',
        color: '#e5e7eb', // Gray 200
        textColor: '#000000',
        commands: [
            { id: 'branch', name: 'git branch', description: 'List/Create branches. Use "git branch -d" to delete merged branches.' },
            { id: 'switch', name: 'git switch', description: 'Switch branches. Use "git switch -c <name>" to create and switch instantly.' },
            { id: 'checkout', name: 'git checkout', description: 'Switch branches or restore files using "git checkout -- <file>".' },
            { id: 'merge', name: 'git merge', description: 'Join branches. Use "git merge --no-ff" to preserve the merge history.' },
            { id: 'fetch', name: 'git fetch', description: 'Download remote info. Use "git fetch --prune" to clean up deleted branches.' },
            { id: 'diff', name: 'git diff', description: 'Show changes. "git diff --staged" shows what you are about to commit.' },
            { id: 'log', name: 'git log', description: 'Show history. Try "git log --oneline --graph" for a visual tree.' },
            { id: 'blame', name: 'git blame', description: 'Show who changed lines. Useful for debugging specific lines.' },
        ]
    },
    {
        id: 'proficient',
        name: 'Git Proficient',
        description: 'Correction & Management',
        color: '#9ca3af', // Gray 400
        textColor: '#000000',
        commands: [
            { id: 'restore', name: 'git restore', description: 'Restore working tree files' },
            { id: 'reset', name: 'git reset', description: 'Reset current HEAD to the specified state' },
            { id: 'rm', name: 'git rm', description: 'Remove files from the working tree' },
            { id: 'clean', name: 'git clean', description: 'Remove untracked files' },
            { id: 'tag', name: 'git tag', description: 'Create, list, delete or verify tag object' },
            { id: 'remote', name: 'git remote', description: 'Manage tracked repositories' },
            { id: 'show', name: 'git show', description: 'Show various types of objects' },
            { id: 'stash', name: 'git stash', description: 'Stash the changes in a dirty working directory away' },
            { id: 'revert', name: 'git revert', description: 'Revert some existing commits' },
        ]
    },
    {
        id: 'advanced',
        name: 'Git Advanced',
        description: 'Rewrite & Internals',
        color: '#6b7280', // Gray 500
        textColor: '#ffffff',
        commands: [
            { id: 'rebase', name: 'git rebase', description: 'Reapply commits on top of another base tip' },
            { id: 'cherry_pick', name: 'git cherry-pick', description: 'Apply changes introduced by some existing commits' },
            { id: 'reflog', name: 'git reflog', description: 'Manage reflog information' },
            { id: 'worktree', name: 'git worktree', description: 'Manage multiple working trees', disabled: true },
        ]
    }
];
