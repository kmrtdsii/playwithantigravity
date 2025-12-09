// Simulate logic from GitGraphViz.jsx
console.log("Starting Label Logic Verification...");

const branches = { main: 'abc', dev: 'abc', feat: 'def' };
const HEAD = { type: 'branch', ref: 'main' }; // HEAD -> main -> abc
// So abc should have [main, dev, HEAD] (order depends on Object.entries, but main/dev are branches)
// Wait, my code was:
/*
    Object.entries(branches).forEach(([name, commitId]) => {
        if (!commitId) return;
        if (!labelsMap[commitId]) labelsMap[commitId] = [];
        labelsMap[commitId].push({ text: name, type: 'branch' });
    });
    if (headCommitId) {
         if (!labelsMap[headCommitId]) labelsMap[headCommitId] = [];
         labelsMap[headCommitId].push({ text: 'HEAD', type: 'head' });
    }
*/
// So for 'abc':
// 1. push 'main'
// 2. push 'dev'
// 3. push 'HEAD'
// Array: ['main', 'dev', 'HEAD']
// Rendering loop:
// map((label, i) =>
//   yPos = node.y - NODE_RADIUS - 10 - (i * 14)
// i=0 ('main'): y = base - 0. (Lowest)
// i=1 ('dev'): y = base - 14. (Higher)
// i=2 ('HEAD'): y = base - 28. (Highest)
// Visual Result:
//   HEAD
//   dev
//   main
//   (Node)
//
// This satisfies "HEADが一番上（ブランチ名より上）".

const labelsMap = {};
const headCommitId = 'abc';

Object.entries(branches).forEach(([name, commitId]) => {
    if (!commitId) return;
    if (!labelsMap[commitId]) labelsMap[commitId] = [];
    labelsMap[commitId].push({ text: name, type: 'branch' });
});

if (headCommitId) {
    if (!labelsMap[headCommitId]) labelsMap[headCommitId] = [];
    labelsMap[headCommitId].push({ text: 'HEAD', type: 'head' });
}

console.log("Labels for abc:", labelsMap['abc'].map(l => l.text));
const abcLabels = labelsMap['abc'].map(l => l.text);
if (abcLabels[abcLabels.length - 1] !== 'HEAD') {
    console.error("FAILED: HEAD is not the last (top-most) label");
} else {
    console.log("SUCCESS: HEAD is on top.");
}
