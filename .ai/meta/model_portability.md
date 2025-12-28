# Model Portability & Cross-LLM Compatibility

> [!NOTE]
> The `.ai/` knowledge base should work effectively across different AI models (Claude, Gemini, GPT, Mistral, etc.). This document defines principles for **model-agnostic** guidelines.

---

## 1. The Portability Challenge

### Current Risks
- Guidelines optimized for Claude's reasoning style
- Prompts using Claude-specific phrasing
- Assumptions about context window sizes
- Tool calling syntax that varies by model

### Goal
Create guidelines that:
- Work with 90%+ effectiveness across major models
- Degrade gracefully when model lacks capability
- Can be easily adapted for model-specific extensions

---

## 2. Model Capability Matrix (2025)

| Capability | Claude Opus | Gemini Pro 3 | GPT-4.5 | Notes |
|------------|-------------|--------------|---------|-------|
| Code generation | ★★★★★ | ★★★★★ | ★★★★☆ | All excellent |
| Long context | 200K | 2M | 128K | Gemini leads |
| Tool calling | ★★★★★ | ★★★★☆ | ★★★★★ | Similar |
| Reasoning chains | ★★★★★ | ★★★★★ | ★★★★☆ | Top tiers equal |
| Instruction following | ★★★★★ | ★★★★☆ | ★★★★★ | Minor variations |

---

## 3. Writing Portable Guidelines

### DO: Use Clear, Explicit Language
```markdown
# Good (Portable)
When encountering an error, you MUST:
1. Log the error details
2. Attempt recovery using the documented strategy
3. Escalate to user if recovery fails

# Risky (Model-dependent phrasing)
Use your best judgment to handle errors elegantly
```

### DO: Provide Concrete Examples
```markdown
# Good (Portable)
## Error Handling Pattern
```go
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}
```

# Risky (Abstract)
Handle errors using idiomatic Go patterns
```

### DON'T: Rely on Implicit Understanding
```markdown
# Risky
Follow the usual process for code review

# Good
Follow the 6-step checklist in `.ai/prompts/code_review.md`
```

---

## 4. Prompt Engineering for Portability

### Structure over Style
All models respond well to:
```markdown
## Role
You are a [specific role]

## Task
[Clear objective]

## Context
[Relevant background]

## Constraints
- [Explicit rule 1]
- [Explicit rule 2]

## Output Format
[Expected structure]
```

### Avoid Model-Specific Quirks
| Avoid | Replace With |
|-------|--------------|
| "As Claude, you should..." | "When processing this request..." |
| "Use your context window of 200K" | "Within available context limits" |
| XML-style tags (Claude-specific) | Markdown sections |

---

## 5. Tool Calling Abstraction

### Universal Tool Description Format
```yaml
tool_name: search_codebase
description: Find files or content matching a pattern
inputs:
  - name: query
    type: string
    required: true
    description: Search pattern or keyword
  - name: scope
    type: enum[file, content, both]
    default: both
outputs:
  - matches: list of file paths or content snippets
```

### Model-Specific Adapters (When Needed)
```
.ai/adapters/
├── claude/    # Claude-specific tool mappings
├── gemini/    # Gemini-specific adjustments
└── openai/    # OpenAI tool format
```

---

## 6. Context Window Strategies

### Adaptive Loading
```markdown
## Context Strategy

### For Long-Context Models (>500K tokens)
- Load full architecture documentation
- Include related test files
- Keep conversation history

### For Standard Models (100-200K tokens)
- Load only task-relevant files
- Summarize architecture to ~1K tokens
- Truncate history after 5 turns

### For Short-Context Models (<100K tokens)
- Load minimum viable context
- Use RAG for additional lookup
- Aggressive summarization
```

---

## 7. Testing Across Models

### Compatibility Test Suite
For each critical guideline, verify:

1. **Parse Test**: Can the model understand the instruction?
2. **Apply Test**: Does the model apply it correctly?
3. **Edge Test**: Does it handle edge cases?

### Test Case Template
```markdown
## Test: Command Phasing Pattern

### Input
"Implement a new git stash command following project standards"

### Expected Behavior (Any Model)
- Creates `parseArgs` function
- Creates `resolveContext` function
- Creates `performAction` function
- Follows existing command structure

### Model Variations (Acceptable)
- Minor syntax differences
- Comment style variations
- Import ordering
```

---

## 8. Graceful Degradation

### When Model Lacks Capability
```markdown
## Fallback Strategy

### If model cannot call tools:
1. Provide explicit file paths instead of search
2. Include relevant code snippets inline
3. Ask user to verify assumptions

### If model has limited context:
1. Pre-summarize relevant documentation
2. Use RAG for on-demand lookup
3. Break task into smaller subtasks

### If model struggles with complexity:
1. Decompose into simpler sub-tasks
2. Provide step-by-step walkthrough
3. Include worked examples
```

---

## 9. Implementation for GitGym

### Create Abstraction Layer
```
.ai/
├── guidelines/          # Model-agnostic core
├── adapters/            # Model-specific extensions
│   ├── claude-opus/
│   │   └── reasoning_hints.md
│   ├── gemini-pro/
│   │   └── long_context_tips.md
│   └── shared/
│       └── tool_mappings.yaml
└── tests/
    └── portability_suite.md
```

### Version Compatibility Notes
Each guideline file should include:
```markdown
---
min_model_capability: reasoning-v2
context_requirement: 50K tokens
tool_dependency: file_operations, search
model_tested: [claude-opus-4, gemini-pro-3]
---
```

