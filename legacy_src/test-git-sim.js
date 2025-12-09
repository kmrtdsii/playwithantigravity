import { gitReducer, initialState, ACTION_TYPES } from './src/lib/git-simulation.js';

console.log("Starting Git Simulation Verification...");

let state = initialState;

function dispatch(action) {
    state = gitReducer(state, action);
}

// 1. Init
dispatch({ type: ACTION_TYPES.INIT });
console.log("Initialized:", state.initialized);

// 2. Add files
dispatch({ type: ACTION_TYPES.ADD, payload: { files: ['file.txt'] } });
console.log("Staging after ADD:", state.staging);

// 3. Commit
dispatch({ type: ACTION_TYPES.COMMIT, payload: { message: "Initial commit" } });
console.log("HEAD after COMMIT:", state.HEAD);

// 4. Switch -c (create branch)
dispatch({ type: ACTION_TYPES.SWITCH, payload: { ref: 'feature-branch', create: true } });
console.log("HEAD after SWITCH -c:", state.HEAD);
if (state.HEAD.ref !== 'feature-branch') console.error("FAILED: Did not switch to feature-branch");

// 5. Switch back to main
dispatch({ type: ACTION_TYPES.SWITCH, payload: { ref: 'main', create: false } });
console.log("HEAD after SWITCH main:", state.HEAD);
if (state.HEAD.ref !== 'main') console.error("FAILED: Did not switch back to main");

// 6. Restore --staged
dispatch({ type: ACTION_TYPES.ADD, payload: { files: ['wrong.txt'] } });
console.log("Staging before RESTORE:", state.staging);
dispatch({ type: ACTION_TYPES.RESTORE, payload: { files: ['wrong.txt'], staged: true } });
console.log("Staging after RESTORE:", state.staging);
if (state.staging.includes('wrong.txt')) console.error("FAILED: restore --staged did not remove file");

console.log("Verification Complete.");
