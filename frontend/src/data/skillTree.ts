export interface SkillCommand {
    id: string;
    name: string;
    description: string;
    missionId?: string;
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
        name: 'skills.basic.name',
        description: 'skills.basic.desc',
        color: '#ffffff',
        textColor: '#000000',
        commands: [
            { id: 'init', name: 'skills.init.name', description: 'skills.init.desc', disabled: true },
            { id: 'clone', name: 'skills.clone.name', description: 'skills.clone.desc' },
            { id: 'status', name: 'skills.status.name', description: 'skills.status.desc' },
            { id: 'help', name: 'skills.help.name', description: 'skills.help.desc' },
            { id: 'add', name: 'skills.add.name', description: 'skills.add.desc' },
            { id: 'commit', name: 'skills.commit.name', description: 'skills.commit.desc' },
            { id: 'push', name: 'skills.push.name', description: 'skills.push.desc' },
            { id: 'pull', name: 'skills.pull.name', description: 'skills.pull.desc' },
        ]
    },
    {
        id: 'intermediate',
        name: 'skills.intermediate.name',
        description: 'skills.intermediate.desc',
        color: '#e5e7eb', // Gray 200
        textColor: '#000000',
        commands: [
            { id: 'branch', name: 'skills.branch.name', description: 'skills.branch.desc' },
            { id: 'switch', name: 'skills.switch.name', description: 'skills.switch.desc' },
            { id: 'checkout', name: 'skills.checkout.name', description: 'skills.checkout.desc' },
            { id: 'merge', name: 'skills.merge.name', description: 'skills.merge.desc', missionId: '001-conflict-crisis' },
            { id: 'fetch', name: 'skills.fetch.name', description: 'skills.fetch.desc' },
            { id: 'diff', name: 'skills.diff.name', description: 'skills.diff.desc' },
            { id: 'log', name: 'skills.log.name', description: 'skills.log.desc' },
            { id: 'blame', name: 'skills.blame.name', description: 'skills.blame.desc' },
        ]
    },
    {
        id: 'proficient',
        name: 'skills.proficient.name',
        description: 'skills.proficient.desc',
        color: '#9ca3af', // Gray 400
        textColor: '#000000',
        commands: [
            { id: 'restore', name: 'skills.restore.name', description: 'skills.restore.desc' },
            { id: 'reset', name: 'skills.reset.name', description: 'skills.reset.desc' },
            { id: 'rm', name: 'skills.rm.name', description: 'skills.rm.desc' },
            { id: 'clean', name: 'skills.clean.name', description: 'skills.clean.desc' },
            { id: 'tag', name: 'skills.tag.name', description: 'skills.tag.desc' },
            { id: 'remote', name: 'skills.remote.name', description: 'skills.remote.desc' },
            { id: 'show', name: 'skills.show.name', description: 'skills.show.desc' },
            { id: 'stash', name: 'skills.stash.name', description: 'skills.stash.desc' },
            { id: 'revert', name: 'skills.revert.name', description: 'skills.revert.desc' },
        ]
    },
    {
        id: 'advanced',
        name: 'skills.advanced.name',
        description: 'skills.advanced.desc',
        color: '#6b7280', // Gray 500
        textColor: '#ffffff',
        commands: [
            { id: 'rebase', name: 'skills.rebase.name', description: 'skills.rebase.desc' },
            { id: 'cherry_pick', name: 'skills.cherry_pick.name', description: 'skills.cherry_pick.desc' },
            { id: 'reflog', name: 'skills.reflog.name', description: 'skills.reflog.desc' },
            { id: 'worktree', name: 'skills.worktree.name', description: 'skills.worktree.desc', disabled: true },
        ]
    }
];
