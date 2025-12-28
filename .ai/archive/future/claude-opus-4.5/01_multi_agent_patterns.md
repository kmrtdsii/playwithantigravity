# Multi-Agent Orchestration Patterns

> [!IMPORTANT]
> In 2025, complex software tasks benefit from **specialized agent collaboration** rather than monolithic single-agent execution.

---

## 1. The Problem

Single-agent architectures face limitations:
- **Context saturation**: One agent trying to hold entire codebase state
- **Role confusion**: Same agent switches between planning/executing/reviewing
- **No checks and balances**: Self-review is inherently biased

---

## 2. Agent Roles (Recommended Separation)

### Core Roles

| Role | Responsibility | When to Invoke |
|------|----------------|----------------|
| **Planner** | Breaks down tasks, creates `implementation_plan.md` | Start of any multi-step task |
| **Executor** | Writes code, runs commands | After plan approval |
| **Reviewer** | Critiques Executor's output | After execution complete |
| **Debugger** | Analyzes failures, proposes fixes | When tests fail |
| **Documenter** | Updates docs, creates walkthroughs | After verification |

### Extended Roles (Complex Projects)

| Role | Responsibility |
|------|----------------|
| **Security Auditor** | Reviews for vulnerabilities post-implementation |
| **Performance Analyst** | Profiles and optimizes critical paths |
| **UX Specialist** | Reviews user-facing changes for accessibility/design |

---

## 3. Handoff Protocol

### Standard Handoff Format
```markdown
## Agent Handoff
**From**: [Role] @ [Timestamp]
**To**: [Role]
**Context Summary**: [What was done, current state]
**Artifacts**:
- `implementation_plan.md` — Approved plan
- `changes.diff` — What changed
**Open Questions**: [List any unresolved issues]
**Next Action**: [Specific instruction for receiving agent]
```

### Example: Planner → Executor Handoff
```markdown
## Agent Handoff
**From**: Planner @ 2025-12-28T12:00:00
**To**: Executor
**Context Summary**: User requested `git cherry-pick` implementation. Plan approved.
**Artifacts**:
- `implementation_plan.md` — See Phase 1: parseArgs implementation
**Open Questions**: None
**Next Action**: Implement `backend/internal/git/commands/cherry_pick.go` per plan
```

---

## 4. Shared Context Management

### Artifact-Based Communication
Agents communicate through **artifacts** rather than message passing:
- `task.md` — Current progress (living document)
- `implementation_plan.md` — The agreed contract
- `walkthrough.md` — Proof of completion
- `blockers.md` — Issues requiring human intervention

### Context Window Allocation
When multiple agents operate:
```
Total Context Budget: 100%
├── Shared Artifacts: 30%     # Plans, specs, task state
├── Current File Focus: 40%   # Code being actively edited
├── Reference Files: 20%      # Dependencies, types, related code
└── Agent Memory: 10%         # Role-specific notes
```

---

## 5. Conflict Resolution

### Disagreement Scenarios
1. **Executor disagrees with Plan**: Executor documents objections, returns to Planner
2. **Reviewer rejects implementation**: Specific feedback with line references, Executor revises
3. **Deadlock**: Escalate to human with both perspectives

### Anti-Patterns
- ❌ Agent overriding another's decision without handoff
- ❌ Circular handoffs (A → B → A → B...)
- ❌ Silent modification of shared artifacts

---

## 6. Implementation Recommendations

### For GitGym `.ai/`
1. Add `personas/executor.md`, `personas/reviewer.md` alongside existing architect/qa
2. Create `prompts/handoff.md` template
3. Define artifact ownership rules in `guidelines/collaboration.md`

### Tool Support
Future agentic frameworks should support:
- Explicit role switching (`@role:reviewer`)
- Artifact locking (prevent concurrent edits)
- Handoff validation (check required fields)

---

## 7. Metrics for Multi-Agent Success

| Metric | Target | Measurement |
|--------|--------|-------------|
| Handoff Clarity | 100% have all required fields | Automated validation |
| Circular Handoffs | < 2 per task | Count role switches |
| Human Escalations | < 1 per 5 tasks | Count `blockers.md` creation |
| First-Pass Approval | > 80% | Reviewer acceptance rate |

