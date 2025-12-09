import { gitReducer, initialState, ACTION_TYPES } from './src/lib/git-simulation.js';

console.log("Starting Repro Restore...");

let state = initialState;
function dispatch(action) { state = gitReducer(state, action); }

// 1. Init
dispatch({ type: ACTION_TYPES.INIT });
// 2. Add file.txt
dispatch({ type: ACTION_TYPES.ADD, payload: { files: ['file.txt'] } });

if (!state.staging.includes('file.txt')) console.error("Setup failed: file.txt not staged");

// 3. Restore --staged .
console.log("Attempting `restore --staged .`");
dispatch({ type: ACTION_TYPES.RESTORE, payload: { files: ['.'], staged: true } });

console.log("Staging after restore:", state.staging);

if (state.staging.includes('file.txt')) {
    console.error("BUG REPRODUCED: file.txt still in staging after restore .");
} else {
    console.log("Test passed (bug not reproduced with . )");
}

// 4. Restore specific file
// Reset
dispatch({ type: ACTION_TYPES.ADD, payload: { files: ['file.txt'] } });
console.log("Attempting `restore --staged file.txt`");
dispatch({ type: ACTION_TYPES.RESTORE, payload: { files: ['file.txt'], staged: true } });
console.log("Staging after restore specific:", state.staging);
if (state.staging.includes('file.txt')) {
    console.error("BUG: file.txt still in staging after restore specific");
}
