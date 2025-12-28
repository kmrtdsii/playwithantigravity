# Tool Composition & Adaptive Strategies

> [!NOTE]
> Effective agents don't just *use* tools—they **compose** them into reliable workflows and **build** new tools when existing ones are insufficient.

---

## 1. Tool Usage Principles

### The Three Laws of Tool Use
1. **Verify Before Acting**: Check state before mutation.
2. **Fail Fast, Recover Gracefully**: Handle errors explicitly.
3. **Minimize Round Trips**: Batch operations when possible.

---

## 2. Standard Composition Patterns (Static)

### Pattern A: Parallel Independence
When operations don't depend on each other:
```
view_file A  ──┐
view_file B  ──┼──▶ [Process All]
grep_search  ──┘
```

### Pattern B: Sequential Dependency
```
find_by_name → view_file_outline → view_code_item → replace_file_content
```

### Pattern C: Verification Sandwich
```
view_file (before) → replace_file_content → view_file (after)
```

---

## 3. Adaptive Tooling (Generative)

> [!TIP]
> **"Buy vs Build"**: Use standard tools for 90% of tasks. Build custom tools for the complex 10%.

### When to Build a Tool
- **Complex Logic**: Regex is insufficient (needs AST parsing).
- **Scale**: Modifying 10+ files with complex rules.
- **Safety**: Need a "Dry Run" capability that standard tools lack.

### The Adaptive Protocol
1. **Draft**: Agent writes a script (e.g., `scripts/audit.go`) to a temp location.
2. **Review**: Agent reads the script to ensure safety (no `rm -rf`).
3. **Execute**: `run_command` the script.
4. **Cleanup**: Remove the script.

### Example: Custom Analysis
**Task**: Find functions that ignore error returns.
**Action**:
- Standard: `grep` (unreliable).
- Adaptive: Write `audit_errors.go` using `go/ast` to find ignored returns.

---

## 4. Error Recovery Strategies

| Error Type | Recovery Strategy |
|------------|-------------------|
| **Transient** | Retry with backoff |
| **Input Error** | Validate inputs, ask user |
| **Logic Error** | Fix regex/pattern, retry |
| **Tool Failure** | Fallback to alternative (e.g., `grep` failed → try `find`) |

### Recovery Template
```markdown
## Error Encountered
- **Tool**: replace_file_content
- **Error**: "TargetContent not found"
- **Recovery**: 
  1. Re-read file with view_file
  2. Locate current content
  3. Retry with updated TargetContent
```

---

## 5. Decision Tree: Which Tool?

```mermaid
flowchart TD
    A[Task] --> B{Simple/Standard?}
    B -->|Yes| C[Use Standard Tools]
    B -->|No| D{Complexity Type?}
    D -->|Scale (>10 files)| E[Build Migration Script]
    D -->|Logic (AST/Analysis)| F[Build Analysis Script]
    D -->|UI/Visual| G[Use Multimodal/Screenshot]
```
