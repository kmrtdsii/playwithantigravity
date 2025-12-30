# Agent Observability & Debugging Standards

> [!NOTE]
> When AI agents fail, debugging is often opaque. This document defines standards for making agent behavior **transparent and auditable**.

---

## 1. The Observability Problem

### Current Pain Points
- "Why did the agent make that decision?"
- "Where did the agent get that (wrong) information?"
- "What was the agent's state when it failed?"

### Goal
Enable humans to:
- Trace any agent action to its reasoning
- Identify failure points quickly
- Replay and debug agent sessions

---

## 2. Structured Logging Standards

### Log Levels for Agents

| Level | When to Use | Example |
|-------|-------------|---------|
| `TRACE` | Every tool call | `TRACE: view_file("/path/to/file.go")` |
| `DEBUG` | Decision points | `DEBUG: Selected replace_file_content over multi_replace` |
| `INFO` | Significant actions | `INFO: Created implementation_plan.md` |
| `WARN` | Recoverable issues | `WARN: grep_search returned 0 results, trying broader query` |
| `ERROR` | Failed operations | `ERROR: replace_file_content failed - content not found` |

### Structured Log Format
```json
{
  "timestamp": "2025-12-28T13:00:00Z",
  "level": "INFO",
  "agent_id": "executor-001",
  "task_id": "abc-123",
  "action": "file_edit",
  "tool": "replace_file_content",
  "inputs": {
    "file": "backend/internal/git/commands/cherry_pick.go",
    "target_content_preview": "func (c *CherryPickCommand)..."
  },
  "output_status": "success",
  "duration_ms": 245
}
```

---

## 3. Decision Audit Trail

### Recording Reasoning
For non-trivial decisions, log the reasoning:

```markdown
## Decision Log Entry

**Timestamp**: 2025-12-28T13:05:00Z
**Context**: Implementing error handling for cherry-pick

**Options Considered**:
1. Return error immediately (like rebase.go)
2. Collect all errors, return at end (like commit.go)
3. Interactive resolution (like merge.go)

**Selected**: Option 1

**Rationale**:
- User-facing behavior should be predictable
- Existing cherry-pick implementations use immediate abort
- Matches error handling pattern in implementation-guide.md

**Confidence**: High
**Evidence**: rebase.go:145, implementation-guide.md section 2.1
```

---

## 4. State Snapshots

### When to Capture State
- Before any destructive operation
- After completing a significant phase
- When transitioning between agent roles

### Snapshot Contents
```yaml
snapshot:
  timestamp: "2025-12-28T13:10:00Z"
  phase: "execution"
  task_progress:
    completed: ["parseArgs", "resolveContext"]
    in_progress: "performAction"
    pending: ["testing", "documentation"]
  files_modified:
    - path: "backend/internal/git/commands/cherry_pick.go"
      lines_added: 150
      lines_removed: 0
  context_loaded:
    - "docs/development/implementation-guide.md"
    - "backend/internal/git/commands/rebase.go"
  open_questions: []
  blockers: []
```

---

## 5. Error Attribution

### The "5 Whys" for Agent Errors

When an error occurs, trace back:

```markdown
## Error Analysis

**Error**: "TargetContent not found in file"

**Why 1**: replace_file_content couldn't match content
**Why 2**: File was modified between view and edit
**Why 3**: Another tool call edited the same file
**Why 4**: Parallel tool execution without coordination
**Why 5**: No file locking mechanism in current tool design

**Root Cause**: Parallel edits without synchronization
**Fix**: Sequential edits for same file, or implement locking
```

---

## 6. Replay Capability

### Session Recording
Every agent session should be replayable:

```yaml
session:
  id: "session-xyz-789"
  start: "2025-12-28T12:00:00Z"
  end: "2025-12-28T13:30:00Z"
  
  events:
    - seq: 1
      type: "user_message"
      content: "Implement git cherry-pick"
      
    - seq: 2
      type: "tool_call"
      tool: "find_by_name"
      args: { pattern: "*.go", directory: "/backend/internal/git/commands" }
      result: { files: ["rebase.go", "merge.go", ...] }
      
    - seq: 3
      type: "decision"
      description: "Use rebase.go as template"
      rationale: "Most similar in functionality"
      
    # ... more events
```

### Replay Benefits
- Debug failures by re-examining sequence
- Train new models on successful sessions
- Identify patterns in agent behavior

---

## 7. Metrics & Dashboards

### Key Agent Metrics

| Metric | Description | Target |
|--------|-------------|--------|
| **Task Success Rate** | % of tasks completed without human intervention | > 85% |
| **Tool Failure Rate** | % of tool calls that error | < 5% |
| **Escalation Rate** | % of tasks requiring human help | < 15% |
| **Avg. Tool Calls/Task** | Efficiency measure | < 30 for standard tasks |
| **Context Utilization** | % of context window used productively | > 70% |

### Health Indicators
```
🟢 Healthy: Success rate >90%, no recurring errors
🟡 Warning: Success rate 70-90%, some patterns of failure
🔴 Critical: Success rate <70%, systemic issues
```

---

## 8. Implementation for GitGym

### Add Agent Logging
```go
// In engine.go
func (e *Engine) Execute(ctx context.Context, cmd string) (string, error) {
    log.Info("agent_action", 
        zap.String("command", cmd),
        zap.String("session_id", ctx.SessionID),
    )
    // ... execution
    log.Info("agent_action_complete",
        zap.Duration("duration", elapsed),
        zap.Error(err),
    )
}
```

### Create Observability Artifacts
```
.ai/
├── guidelines/
│   └── observability.md        # These standards
├── templates/
│   └── decision_log.md         # Template for decisions
└── analysis/
    └── session_replay.yaml     # Sample session format
```

---

## 9. Privacy & Security Considerations

### What NOT to Log
- ❌ User credentials or secrets
- ❌ PII (personally identifiable information)
- ❌ Full file contents (use hashes or previews)
- ❌ Session tokens

### What TO Log
- ✅ File paths (not contents)
- ✅ Command names and arguments (sanitized)
- ✅ Decision rationale
- ✅ Error messages

### Retention Policy
```yaml
retention:
  trace_logs: 7 days
  decision_logs: 30 days
  session_replays: 14 days
  aggregated_metrics: 1 year
```

