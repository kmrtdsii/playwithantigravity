# Large Repository Performance Testing Workflow

This document outlines the workflow for verifying the performance of the Git Graph visualization when handling large repositories (6000+ commits), such as `fastapi`.

## Test Target
**Repository**: [https://github.com/fastapi/fastapi.git](https://github.com/fastapi/fastapi.git)
**Scale**: ~6400+ commits

## Testing Steps

### 1. Initial Launch
Start the application and ensure a clean state.

### 2. Configure Remote Repository
1. Click the **Configure** button in the top navigation.
2. In the "Remote Repository URL" field, clear any existing text.
3. Enter `https://github.com/fastapi/fastapi.git`.
4. Click **Update**.
5. Return to the main view (Graph View).
   - **Observation**: The remote repository graph may take 30+ seconds to calculate and render initially.
   - **Checkpoint**: Ensure the Remote Server URL is correctly updated.

### 3. Clone to Local
1. Open the **Terminal** in the bottom pane.
2. Execute the clone command:
   ```bash
   git clone https://github.com/fastapi/fastapi.git
   ```
3. Wait for the clone to complete.
   - **Observation**: The local graph tree should appear. Note the rendering time and "sluggishness" (lag) during this initial render.

### 4. UI Responsiveness Check
Perform the following actions and observe UI frame rate/latency:
1.  **Tab Switching**: Switch between "Alice" and "Bob" (or other available tabs).
    - *Expected*: Immediate switch without freezing.
    - *Issue Scenario*: Significant delay or freeze.
2.  **Toggle Views**: Toggle "Show Previous" (if available) or other view filters.
    - *Issue Scenario*: Slow update of the graph.
3.  **Pane Resizing**: Drag the divider between functionality panes (e.g., File Explorer vs Graph).
    - *Expected*: Smooth resizing.
    - *Issue Scenario*: "Janky" or stuttering movement.

## Goal
The goal of optimization is to reduce the "sluggishness" during Steps 3 and 4, ensuring smooth scrolling and interaction even with ~6400 nodes loaded.
