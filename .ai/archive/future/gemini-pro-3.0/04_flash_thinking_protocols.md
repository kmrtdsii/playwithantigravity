# Flash Thinking Protocols

> [!TIP]
> Speed enables quality. When inference is cheap and fast, we can afford **Parallel Thinking** and **Self-Correction loops** that were previously too expensive.

---

## 1. Parallel Branching Strategy

In a "Flash Thinking" model, we don't commit to the first idea.

### The A/B/C Protocol
**Task**: "Optimize the graph rendering algorithm."
**Old Way**: Plan A -> Execute A -> Fail -> Plan B.
**Flash Way**:
1. **Parallel Draft**:
    - Generates Approach A (D3 Force Layout)
    - Generates Approach B (Dagre Layout)
    - Generates Approach C (Custom Grid)
2. **Comparative Review**: Agent compares A, B, C outputs (code quality, complexity).
3. **Select & Refine**: Pick B, refine it.

**Implementation**: Since agents are typically single-threaded in thought, this is simulated by:
- "I will draft 3 approaches in scratchpad."
- "Now reviewing Approach 1... Approach 2... Approach 3."
- "Selecting Approach 2."

---

## 2. Rapid Prototyping (Write-Verify-Discard)

**Concept**: It is cheaper to write bad code and fix it than to plan perfect code abstractly.

### The Loop
1. **Flash Draft**: Write the full file immediately based on intuition.
2. **Lint/Compile**: Run `go build`.
3. **Flash Fix**: Fix errors immediately.
4. **Logic Check**: Now that it compiles, does it make sense?
5. **Final Polish**: Apply style guide.

**Why**: LLMs are often better at *editing* than *creating*. Getting to a compile-able state fast (even if buggy) provides ground truth for the model to reason upon.

---

## 3. Cognitive Momentum

### Avoid "Stop-and-Think" Paralysis
- **Don't**: Pause after every function to ask "Is this right?"
- **Do**: Flow through the entire implementation.
- **Then**: Review the whole.

### Token Streaming Advantage
- Utilize the stream. While outputting code, the model is "thinking".
- Verbose comments *during* generation = Chain of Thought.
- **Code Commenting Strategy**:
    ```go
    // I need to handle the case where remote is empty.
    // Checking standard library... Use plumbing.ReferenceName...
    if ref == nil { ... }
    ```
    (These thought-comments can be stripped later, but they improve the generated code logic).

---

## 4. Implementation for GitGym

### Prompts for Flash Thinking
Add to `.ai/prompts/`:

```markdown
## Flash Mode
- **Goal**: Speed to working prototype.
- **Constraint**: Do not stop for style/linting yet.
- **Output**: Full file dump.
- **Post-Action**: I will run the linter and fix immediately.
```

### "Scratchpad" Artifact
Create `.ai/temp/scratchpad.md`.
- Use this area for the "Parallel Branching" drafts.
- Do not clutter the main `implementation_plan.md` with discarded ideas.
- Wipe it clean after task.

---

## 5. When NOT to Flash Think

- **Destructive Ops**: `rm -rf`, DB migrations. (Use Claude's **Planner** mode).
- **Security**: Auth flows. (Use **Auditor** mode).
- **Public API Design**: Once released, can't change. (Measure twice, cut once).

