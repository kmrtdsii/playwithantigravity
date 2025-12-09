import { gitReducer, initialState, ACTION_TYPES } from './src/lib/git-simulation.js';

console.log("Starting Git Viz Verification...");

let state = initialState;
function dispatch(action) { state = gitReducer(state, action); }

// 1. Init (Check Mocked Modified Files)
dispatch({ type: ACTION_TYPES.INIT });
console.log("Modified after INIT:", state.modified);
if (!state.modified.includes('file.txt')) console.error("FAILED: mock files not present");

// 2. Add (Modified -> Staging)
dispatch({ type: ACTION_TYPES.ADD, payload: { files: ['file.txt'] } });
console.log("Modified after ADD file.txt:", state.modified);
console.log("Staging after ADD file.txt:", state.staging);
if (state.modified.includes('file.txt')) console.error("FAILED: file.txt should be removed from modified");
if (!state.staging.includes('file.txt')) console.error("FAILED: file.txt should be in staging");

// 3. Restore --staged (Staging -> Modified)
dispatch({ type: ACTION_TYPES.RESTORE, payload: { files: ['file.txt'], staged: true } });
console.log("Modified after RESTORE:", state.modified);
console.log("Staging after RESTORE:", state.staging);
if (!state.modified.includes('file.txt')) console.error("FAILED: file.txt should be back in modified");
if (state.staging.includes('file.txt')) console.error("FAILED: file.txt should be removed from staging");

// 4. Add All (Simulated by manual list for now in test, but UI does logic)
// Let's test 'add .' logic if I implemented it?
// Code: const filesToAdd = files.includes('.') ? state.modified : files;
dispatch({ type: ACTION_TYPES.ADD, payload: { files: ['.'] } });
console.log("Modified after ADD . :", state.modified);
console.log("Staging after ADD . :", state.staging);
if (state.modified.length > 0) console.error("FAILED: modified should be empty");

console.log("Verification Complete.");
