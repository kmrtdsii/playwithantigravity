# Adaptive & Generative Tooling

> [!NOTE]
> Static tools (grep, replace) are broad and blunt. An intelligent agent should **manufacture its own tools** for specific problems.

---

## 1. The Concept: Just-in-Time Intelligence

A human engineer doesn't just use `cat` and `sed`. If a task is complex, they write a throwaway python script.
**AI Agents should do the same.**

---

## 2. Generative Tool Patterns

### Pattern A: Custom Analysis Scripts
**Problem**: "Find all functions that call `Execute` but ignore the error return, across the whole repo."
**Static Tool**: `grep_search` (Hard to regex multiline calls).
**Adaptive Tool**:
1. Agent writes `scripts/audit_calls.go` using `go/ast`.
2. Agent runs `go run scripts/audit_calls.go`.
3. Agent gets perfect semantic analysis results.
4. Agent deletes script.

### Pattern B: The Verification Sandbox
**Problem**: "Verify this complex regex works on all edge cases."
**Static Tool**: Run unit tests (slow, heavy setup).
**Adaptive Tool**:
1. Agent creates `tmp_verify.py`.
2. Agent populates it with 50 test cases and the regex.
3. Agent runs it.
4. Immediate feedback loop.

### Pattern C: Bulk Migration
**Problem**: "Rename `HybridStorer` to `UnifiedStorer` in all files and comments, preserving case."
**Static Tool**: `multi_replace_file_content` (Risky, many calls).
**Adaptive Tool**:
1. Agent writes `migrate_names.sh` using `sed` or a python script with smart casing logic.
2. Agent reviews the script.
3. Agent runs the script once.

---

## 3. Dynamic Tool Protocol

### 1. Proposal
Agent decides: "Existing tools are inefficient for this. I will generate a tool."

### 2. Implementation
Agent writes the code to a temporary location (e.g., `.gemini/tmp/tools/`).

### 3. Safety Check
- **Sandboxed**: Tool must not access network/files outside workspace.
- **Review**: Agent "reads" its own tool code to verify specific safety (no `rm -rf`).

### 4. Execution & Cleanup
Run, capture output, analyze, delete tool.

---

## 4. Implementation for GitGym

### New Capability: `run_generated_script`
A wrapper around `run_command` that streamlines the write-run-delete loop.

### Example: HybridStorer Audit
Instead of reading 50 files to check for `IsHybrid` checks:
```go
// Generated tool: audit_hybrid.go
package main
import (...)
func main() {
  // Walk ast, find type assertions to localStorerProvider
  // Print locations
}
```

### Benefit
- **Accuracy**: AST parsing beats Regex.
- **Speed**: One command vs 50 `view_file` calls.
- **Context**: The filtered output fits easily in context, whereas 50 files might not.

---

## 5. Security & Safety

- **Constraint**: Generated tools should generally be **Read-Only** or **Local Write** (temp files).
- **Modification**: If generating a script to modify code, use `git` checkpoints before running.
- **User Approval**: If `SafeToAutoRun` is false, user reviews the generated script before execution.

