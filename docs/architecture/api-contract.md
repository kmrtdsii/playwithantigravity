# API Contract

This document defines the REST interface between the React Frontend and Go Backend.

## Endpoints

### 1. `GET /api/state`
Returns the current git status of the session.
- **Query Params**:
    - `sessionId`: (Optional) If managing multiple sessions.
- **Response**: `GitState` JSON object.
    ```json
    {
       "initialized": true,
       "commits": [...],
       "branches": {"main": "sha..."},
       "HEAD": {"type": "branch", "ref": "main"}
    }
    ```

### 2. `POST /api/command`
Executes a Git command.
- **Body**:
    ```json
    {
        "command": "git commit -m 'feat: new thing'",
        "args": ["commit", "-m", "feat: new thing"]
    }
    ```
- **Response**:
    ```json
    {
        "output": "[master (root-commit) 1234567] feat: new thing\n 1 file changed...",
        "state": { ...New GitState... }
    }
    ```

### 3. `POST /api/remote/clone`
Initiates a specific remote clone (simulated).
- **Body**: `{ "url": "https://github.com/..." }`
- **Response**: `{ "status": "queued" }` (Actual clone happens async or sync depending on implementation).

## Error Handling
- **400 Bad Request**: Invalid command or arguments.
- **500 Internal Server Error**: Go panic or unhandled filesystem error.
- **Response Shape**:
    ```json
    {
        "error": "Detailed error message",
        "code": "ERR_GIT_LOCK"
    }
    ```
