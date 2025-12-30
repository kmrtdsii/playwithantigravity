# Feature Spec: Git Skill Radar ("God Level" Path)

## 1. Overview
A visual "Skill Tree" or "Radar" that maps Git commands from "Basic" to "God Level", based on the provided reference image.
Users can visualize their progression and select commands to launch specific "Practice Missions".

## 2. Core Value
-   **Gamification**: Visualizing usage/mastery levels (Basic -> God).
-   **Discovery**: Exposing users to advanced commands (`reflog`, `filter-branch`, `worktree`) they might not know.
-   **Actionable**: Directly linking concepts to practice scenarios.

## 3. The "Radar" UI Information Architecture (Based on Image)

The UI will be a **Concentric Sector Chart (Sunburst-like)**.

### Level 1: Git Basic (Center)
*Focus: Short turnings and movement*
-   `git init`
-   `git add`
-   `git commit`
-   `git status`
-   `git push`
-   `git remote`
-   `git pull`

### Level 2: Git Intermediate (Ring 2)
*Focus: Competent new changing forms*
-   `git branch`
-   `git checkout`
-   `git merge`
-   `git diff`

### Level 3: Git Proficient (Ring 3)
*Focus: Uses advanced commands and workflows*
-   `git rebase`
-   `git cherry-pick`
-   `git stash`
-   `git remote` (Advanced usage)
-   `git log`

### Level 4: Git Advanced (Ring 4)
*Focus: Run complete history rewrites*
-   `git reflog`
-   `git tag`
-   `submodules`
-   `hooks`
-   `worktrees`
-   `sparse-checkout`

### Level 5: Git God Level (Outer Ring)
*Focus: Git internals, plumbing, custom scripts*
-   `git filter-branch`
-   `git internals` (Plumbing: `cat-file`, `hash-object`)
-   `low-level plumbing`
-   `custom scripts/APIs`

## 4. Interaction Design
1.  **View State**:
    -   The Radar is displayed (likely in a Modal or a dedicated "Skills" Tab).
    -   Hovering over a sector highlights it and shows a tooltip/description.
    -   Colors indicate "Mastery Status" (e.g., Gray: Locked, Blue: Available, Gold: Mastered). *For prototype: All Blue/Clickable.*
2.  **Click Action**:
    -   Clicking a command (e.g., `git rebase`) opens a **"Mission Brief"** overlay.
    -   Mission Brief contains:
        -   **Concept**: What is it?
        -   **Scenario**: "You have a messy history. Clean it up."
        -   **Action**: "Start Practice" (Loads a specific repo state into the terminal).

## 5. Technical Approach (Prototype)
-   **Library**: Pure SVG + D3.js (or simple SVG math if simple sectors) or just CSS Grid/Flex for a simpler list view first?
-   **User Preference**: "Image's concept... UI". SVG implementation of the circular chart is preferred for impact.
-   **Component**: `SkillRadar.tsx`
