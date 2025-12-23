import type { Commit } from './gitTypes';

export interface SelectedObject {
    type: 'commit' | 'file';
    id: string; // Hash or Path
    data?: Commit | { view?: string; message?: string };
}
