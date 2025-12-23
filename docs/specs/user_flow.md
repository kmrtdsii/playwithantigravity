# GitGym E2E Specification & Basic User Flow

This document defines the core user journey for verifying the multi-user simulation features of GitGym. It serves as the "Source of Truth" for AI assistants and developers when designing UI/UX improvements or adding new features.

## 1. Remote Repository Setup (Pre-condition)
**Objective**: Connect the local GitGym instance to a real remote repository to enable simulation features (`clone`, `fetch`, `push`).

- **Action**:
  1. Navigate to the **Remote Repository** panel (Left Pane).
  2. Click "Connect Repository" or "Configure".
  3. Enter the URL: `https://github.com/kmrtdsii/gitgym.git`.
  4. Click "Update" / "Connect".
- **Verification**:
  - [ ] The "No Remote Configured" placeholder disappears.
  - [ ] The header displays the repository name (`gitgym`) and an "origin" badge.
  - [ ] Remote branches are listed in the "Remote Branches" section.

## 2. Multi-User Simulation Flow (The "Bob & Alice" Scenario)

### Step 1: Bob's Initial Clone
**Context**: "Bob" is the first active developer tab.
- **Action**:
  1. Ensure the "Bob" tab is active.
  2. In the terminal, execute:
     ```bash
     git clone https://github.com/kmrtdsii/gitgym.git
     ```
- **Verification**:
  - [ ] **Terminal**: Output shows standard git clone progress (`Cloning into 'gitgym'...`).
  - [ ] **File Explorer**: The `gitgym` folder appears.
  - [ ] **Graph**: (If implemented) The graph view should populate with the cloned history.

### Step 2: Switch to Second User (Alice)
**Context**: Simulating a second developer joining the environment.
- **Action**:
  1. Click the **"+"** button in the Developer Tabs bar OR click "Alice" if already added.
  2. Enter "Alice" if prompted for a name.
- **Verification**:
  - [ ] **Terminal**: The terminal clears or shows a fresh session for Alice.
  - [ ] **Context**: The `User: Alice` prompt (or similar indicator) confirms the active user.
  - [ ] **State**: Alice should *not* automatically see Bob's uncommitted changes (simulating distinct workspaces, though they share the underlying repo simulation logic).

### Step 3: Persistence Check (Bob Re-selection)
**Context**: Switching back to the original user to verified state retention.
- **Action**:
  1. Click the "Bob" tab.
- **Verification**:
  - [ ] **Terminal**: The previous command history (`git clone ...`) and output is visible. The session is restored exactly as left.
  - [ ] **Context**: Prompt returns to `User: Bob`.

### Step 4: Persistence Check (Alice Re-selection)
**Context**: Switching back to the second user.
- **Action**:
  1. Click the "Alice" tab.
- **Verification**:
  - [ ] **Terminal**: Alice's session state is restored (e.g., if it was empty, it remains empty).

## 3. UI/UX Improvement Goals (Roadmap)
Based on this flow, the following improvements are planned:

- [ ] **Visual Distinction**: clearer UI cues (borders, avatars, or theme colors) to distinguish between "Bob's View" and "Alice's View".
- [ ] **Terminal Feedback**: Enhance terminal output to clearly indicate when a command is simulated versus real.
- [ ] **Graph Ghosting**: When Alice fetches, show "Ghost Commits" that Bob hasn't seen yet if they diverge.
- [ ] **Auto-Discovery**: If a user runs `git clone <url>`, automatically configure the Remote Panel without requiring manual setup in Step 1.

## 4. Automated Verification
Future E2E tests (Playwright) should script steps 1-4 precisely to ensure no regression in multi-user state isolation.
