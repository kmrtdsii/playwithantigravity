# Efficient Context Loading (Standard Models)

> [!NOTE]
> For models with standard context limits (< 200K tokens), efficiency is paramount. This guide defines strategies for hierarchical loading and summarization.

---

## 1. The Context Budget Model

Think of context as a **finite currency**:

```
┌────────────────────────────────────────────────────────────┐
│ Context Window (e.g., 200K tokens)                         │
├────────────────────────────────────────────────────────────┤
│ [System Prompt]  │ [Task Context] │ [Code] │ [Response]   │
│     ~5%          │     ~15%       │  ~60%  │    ~20%      │
└────────────────────────────────────────────────────────────┘
```

**Key Insight**: Every token spent on irrelevant context is a token stolen from reasoning capacity.

---

## 2. Hierarchical Context Loading

### Level 0: Core Identity (Always Loaded)
- `.ai/guidelines/genai_base.md` — Core agent behavior
- Project-specific critical rules

### Level 1: Task-Relevant (Loaded per Task)
- `docs/architecture/*.md` — If touching architecture
- `docs/development/implementation-guide.md` — If implementing features
- Relevant `.ai/patterns/*.md` — Based on task type

### Level 2: File-Specific (Loaded on Demand)
- Source files being edited
- Direct dependencies (imports)
- Related test files

### Level 3: Reference (Loaded Sparingly)
- Similar implementations (for pattern reference)
- Historical context (previous decisions)

---

## 3. Context Compression Techniques

### A. Intelligent Summarization
Before loading a large file, generate a **structural summary**:

```markdown
## File: backend/internal/state/graph_traversal.go
- **Purpose**: BFS traversal to build Git graph visualization
- **Key Functions**:
  - `populateCommits` — Main entry
  - `bfsTraverse` — Core logic
- **Lines**: 380
```

### B. Outline-First Approach
1. Always use `view_file_outline` before `view_file`
2. Only load specific functions/classes needed
3. Use `view_code_item` for surgical precision

---

## 4. Context Eviction Policies

| Priority | What to Evict | Reason |
|----------|---------------|--------|
| 1 (First) | Historical conversation | Older turns less relevant |
| 2 | Reference files already processed | Can re-fetch if needed |
| 3 | Completed task artifacts | Summarize to `walkthrough.md` first |
| 4 (Last) | Current working files | Never evict active editing context |
