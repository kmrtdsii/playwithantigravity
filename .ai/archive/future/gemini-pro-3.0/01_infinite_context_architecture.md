# Infinite Context Architecture

> [!TIP]
> With 2M+ token windows, "Context Management" becomes "Context Utilization". Stop summarizing; start correlating.

---

## 1. The Death of Summarization

In classical LLM strategies, we summarize large files to save tokens.
**Risk**: Summarization is *lossy compression*. It strips "implementation details"—but bugs *live* in the details.

### The Gemini Approach: Lossless Loading
- **Don't**: Summarize `backend/internal/git/commands/*.go` into a 1-page brief.
- **Do**: Load **all** 20 command files entirely.
- **Why**: To catch that `cherry-pick.go` handles errors differently than `rebase.go` in a way a summary would miss.

---

## 2. The "Whole-Repo" Cognition Model

Instead of hierarchies (Level 0-3), we define **Cognitive Spheres**:

### Sphere A: The Active Radius (High Precision)
- **Content**: All files involved in the feature + direct tests + 1-hop dependencies.
- **Usage**: Deep syntactic analysis, line-by-line editing.

### Sphere B: The Environmental Context (Pattern Matching)
- **Content**: The entire rest of the `backend/` directory.
- **Usage**: Checking for naming consistency, identifying duplicated utility functions, spotting architectural patterns.
- **Technique**: "Find similar implementations to X in the whole repo."

### Sphere C: The Infinite Reference (Retrieval)
- **Content**: Entire generic `.ai/` folder, third-party library docs (e.g., `go-git` full documentation).
- **Usage**: Looking up APIs without guessing.

---

## 3. RAG vs. Long Context

**RAG (Retrieval Augmented Generation)**:
- Good for: "Find the one file that mentions X".
- Bad for: "How does the `Session` struct flow through the entire request lifecycle?"

**Long Context**:
- **Usage**: Load the whole call stack trace source code.
- **Benefit**: The model "sees" the flow across 50 files simultaneously.

### Strategy for 2025
1. **Try Context First**: If it fits (e.g., < 2M tokens), load it. Physical proximity in prompt > Semantic search proximity.
2. **RAG as Fallback**: Only for archival data or gigabyte-scale logs.

---

## 4. New Context Patterns

### Pattern: "The Big Diff"
Instead of asking "What changed in this file?", load:
1. `git diff main...feature-branch` (complete output)
2. All modified files in full.
3. All verified tests.
**Prompt**: "Review this entire feature for coherence against the project style guide."

### Pattern: "Trace-Driven Debugging"
1. Run a failing test with verbose logging.
2. Capture the **entire** log output (even if 10MB).
3. Load the **entire** source tree.
4. **Prompt**: "Map this log trace to the code execution path and identify the divergence."

---

## 5. Implementation for GitGym

### Recommended Preload Sets (Gemini Edition)

```yaml
# Don't pick and choose. Load the domain.
task_type: "fix_backend_bug"
preload:
  - "docs/**" (All docs)
  - "backend/**/*.go" (Entire backend source)
  - "go.mod"
  - "go.sum"
warning: "If total tokens > 1M, exclude *_test.go"
```

### Context Eviction?
**Rarely needed**.
- **Exception**: Evict *previous* task's file dumps to avoid variable name collision confusion.
- **Keep**: All architecture, guidelines, and project specs loaded permanently.

---

## 6. Performance Logic
"Won't this be slow?"
- **Gemini 1.5/3.0** optimized specifically for long-context retrieval speed.
- The time cost of **"Load All → Answer Once"** is often *less* than **"Search → Load Partial → Fail → Search Again → Load More"**.

