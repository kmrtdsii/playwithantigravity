# Feature Spec: Practice Missions (GitGym Scenarios)

> [!NOTE]
> This document defines the specifications for "Practice Missions", a gamified feature where users solve specific Git challenges to earn mastery on the Skill Radar.

## 1. Overview
Practice Missions are isolated, objective-based scenarios. Unlike the open sandbox, a mission has a clear **Start State** (e.g., a broken repository) and a specific **Goal** (e.g., "Fix the conflict" or "Recover the deleted commit").

## 2. User Experience (UX) Flow

### Phase 1: Selection (The Brief)
*   **Entry Point**: Clicking an item on the **Skill Radar** (e.g., "Merge").
*   **UI**: A "Mission Brief" modal appears.
    *   **Title**: "Mission: The Conflict Crisis"
    *   **Scenario Description**: "Bob and Alice edited the same file. The repository is in a broken merge state. Fix it so that both changes are preserved."
    *   **Difficulty**: ⭐⭐
    *   **Estimated Time**: 5 min
    *   **Action**: `[Start Mission]` button.

### Phase 2: Engagement (The Workbench)
*   **State Reset**: The Terminal/Graph is wiped and replaced with the *Mission State*.
    *   *Visual Cue*: The UI theme might shift slightly (e.g., "Simulation Mode" border) to indicate this is not the user's persistent playground.
*   **Mission Panel**: A persistent side-panel (collapsible) displays:
    *   **Current Objective**: "Resolve conflicts in `README.md`."
    *   **Validation Status**: "Core requirements met: 0/3"
    *   **Hint System**: "Stuck? [Reveal Hint 1]"

### Phase 3: Resolution (Success/Fail)
*   **Verification**: The backend analyzes the git state against success criteria.
*   **Success**:
    *   "Mission Accomplished" overlay.
    *   **Reward**: The "Merge" sector on Skill Radar turns Gold.
    *   **Choice**: `[Next Mission]` or `[Return to Sandbox]`.
*   **Failure**:
    *   Standard Git errors (e.g., "Repository is empty").
    *   Option to `[Retry]` (Resets the scenario to start).

## 3. Technical Architecture

### 3.1 Data Model: Mission Definition (YAML)
Missions are defined as data, making them extensible.

```yaml
id: "merge-conflict-001"
title: "The Conflict Crisis"
difficulty: "basic"
skill: "merge"

setup:
  # Commands needed to create the broken state
  - "git init"
  - "echo 'Line 1' > file.txt && git add . && git commit -m 'Initial'"
  - "git checkout -b feature"
  - "echo 'Line 2 (Feature)' >> file.txt && git commit -am 'Feature change'"
  - "git checkout master"
  - "echo 'Line 2 (Master)' >> file.txt && git commit -am 'Master change'"
  - "git merge feature" # This causes a conflict and exit code 1 (allowed)

validation:
  # Logic to verify success
  type: "function" # Or a simple state check
  checks:
    - type: "no_conflict"
      description: "No merge conflicts remain"
    - type: "commit_exists"
      message_pattern: "Merge branch 'feature'"
      description: "A merge commit was created"
    - type: "file_content"
      path: "file.txt"
      contains: ["Line 2 (Feature)", "Line 2 (Master)"]
      description: "Both changes are preserved"

hints:
  - "Run `git status` to see which files are in conflict."
  - "Edit the file to remove the `<<<<` markers."
  - "After fixing the file, don't forget to `git add` and `git commit` to finish the merge."
```

### 3.2 Backend Implementation (`internal/mission/`)
*   **Mission Engine**:
    *   `LoadMission(id string)`: Reads the YAML.
    *   `StartMission(sessionID, missionID)`:
        1.  Creates a **new** temporary directory (e.g., `/tmp/gym_mission_<id>`).
        2.  Executes `setup` commands in sequence.
        3.  Returns the new session state to Frontend.
*   **Mission Validator**:
    *   `VerifyMission(sessionID, missionID)`:
        1.  Inspects the `go-git` Repository object.
        2.  Runs the checks defined in `validation` (e.g., check `repo.Status()`, traverse `repo.Log()`).
        3.  Returns `{ success: boolean, progress: CheckResult[] }`.

### 3.3 Frontend Integration
*   **`MissionContext`**: Manages the active mission state (`activeMissionId`, `currentObjective`, `hintsRevealed`).
*   **`GitAPIContext` Switch**:
    *   When a mission starts, the `GitAPIContext` must switch its `sessionId` pointer to the ephemeral mission session.
    *   When the mission ends, it switches back to the user's main persistent session.

## 4. Content Roadmap: First 3 Missions

| Skill | Mission Title | Scenario | Goal |
| :--- | :--- | :--- | :--- |
| **Commit** | "The Forgotten File" | You committed, but forgot to add `config.json`. | Use `git commit --amend` to add the file without making a new commit. |
| **Merge** | "Conflict Crisis" | (Defined above) | Fix a merge conflict manually. |
| **Rebase** | "Cleanup Crew" | 3 messy "WIP" commits on a feature branch. | Use `git rebase -i` to squash them into one clean commit. |

## 5. Development Phases

1.  **Phase 1: Engine Core**: Implement the Backend Engine to parse YAML and setup scenarios.
2.  **Phase 2: UI Overlay**: Build the "Brief" and "Mission Panel" React components.
3.  **Phase 3: Validation Logic**: Implement the specialized check logic (content searching, graph topology verification).
