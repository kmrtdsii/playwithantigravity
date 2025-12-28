# Epistemic Guardrails: Uncertainty & Hallucination Prevention

> [!CAUTION]
> AI agents can generate confident-sounding but **incorrect** information. This document defines guardrails for intellectual honesty and uncertainty handling.

---

## 1. The Uncertainty Spectrum

```
┌─────────────────────────────────────────────────────────────────────┐
│ Certainty Level                                                      │
├────────────┬────────────┬────────────┬────────────┬────────────────┤
│ VERIFIED   │ HIGH CONF  │ MEDIUM     │ LOW CONF   │ UNKNOWN        │
│ (Source)   │ (Pattern)  │ (Inference)│ (Guess)    │ (Admit it)     │
├────────────┼────────────┼────────────┼────────────┼────────────────┤
│ Read from  │ Seen this  │ Logically  │ Based on   │ "I don't know" │
│ docs/code  │ pattern    │ follows    │ heuristic  │                │
└────────────┴────────────┴────────────┴────────────┴────────────────┘
```

---

## 2. Core Principles

### Principle 1: Ground in Evidence
Every claim should be traceable to:
- Source code (`file.go:42`)
- Documentation (`docs/architecture/x.md`)
- Tool output (grep result, command output)
- Explicit user statement

### Principle 2: Signal Uncertainty
When uncertain, use explicit markers:
```markdown
**Verified**: The function `Execute` returns (string, error) — confirmed in `engine.go:15`

**Likely**: Based on similar commands, this should follow the Command Phasing pattern

**Uncertain**: I haven't found documentation for this behavior

**Unknown**: I cannot determine this without additional information
```

### Principle 3: Prefer Ignorance to Fabrication
```
❌ "The config is in /etc/gitgym/config.yaml"  (fabricated path)
✅ "I haven't located the config file. Let me search for it."
```

---

## 3. Hallucination-Prone Zones

### High Risk Areas
| Zone | Risk | Mitigation |
|------|------|------------|
| **API endpoints** | Inventing non-existent routes | Always verify in `handlers.go` |
| **File paths** | Guessing directory structure | Use `find_by_name` or `list_dir` |
| **Function signatures** | Wrong parameters | Use `view_code_item` |
| **Error messages** | Making up error text | Copy exact output |
| **External dependencies** | Version mismatches | Check `go.mod` / `package.json` |

### Lower Risk Areas
| Zone | Risk | Reason |
|------|------|--------|
| General programming concepts | Low | Well-established knowledge |
| Standard library usage | Low | Stable APIs |
| Design patterns | Low | Universal principles |

---

## 4. Verification Strategies

### Before Making Claims
1. **Search first**: `grep_search` for the term/pattern
2. **Read the source**: `view_file` or `view_code_item`
3. **Check documentation**: Load relevant `docs/` files
4. **Test if possible**: Run command to verify behavior

### Verification Checklist
```markdown
- [ ] Have I **seen** this in code/docs? (Not just assumed)
- [ ] Can I **cite** the source? (File:line or doc section)
- [ ] Have I **tested** this claim? (If verifiable)
- [ ] Am I **extrapolating** beyond evidence? (Flag if so)
```

---

## 5. Escalation Triggers

### When to Ask the User
- "The documentation doesn't specify this behavior"
- "I found conflicting information in X and Y"
- "This requires domain knowledge I don't have"
- "The code suggests X, but I'm uncertain about the intent"

### Escalation Format
```markdown
## Clarification Needed

**Context**: Implementing error handling for `git cherry-pick`

**Question**: Should conflicts abort the entire cherry-pick or allow partial application?

**What I Found**:
- `merge.go` aborts on conflict (line 142)
- User flow spec mentions "resolve conflicts" UI (step 5)

**Options**:
1. Abort on conflict (consistent with merge)
2. Partial apply (requires conflict resolution)

**My Recommendation**: Option 1, unless you want conflict UI

**Confidence**: Medium — I need your input on product intent
```

---

## 6. Self-Audit Protocol

After generating code or claims, perform self-audit:

### The "Fabrication Check"
Ask yourself:
1. "Did I **actually read** this, or am I **inferring**?"
2. "Would this claim survive a `grep_search`?"
3. "If asked to prove this, could I point to evidence?"

### Red Flags to Catch
- ⚠️ Specific numbers without source (e.g., "processes 10,000 requests/sec")
- ⚠️ Exact file paths you haven't verified
- ⚠️ API endpoints you haven't seen in code
- ⚠️ "Always" or "Never" claims without evidence

---

## 7. Confidence Calibration

### Expressing Confidence in Code Comments
```go
// VERIFIED: This pattern matches the Command interface in engine.go
func (c *MyCommand) Execute(...) { ... }

// ASSUMPTION: Error format follows existing convention (see push.go:85)
return fmt.Errorf("failed to cherry-pick: %w", err)

// TODO(uncertain): Is this the correct ref resolution order?
```

### Expressing Confidence in Plans
```markdown
## Implementation Plan

### Phase 1: Parse Arguments
**Confidence**: HIGH — Standard pattern per implementation-guide.md

### Phase 2: Handle Commit Ranges
**Confidence**: MEDIUM — Inferred from rebase.go, needs review

### Phase 3: Conflict Detection
**Confidence**: LOW — No existing example found, requires design decision
```

---

## 8. Implementation for GitGym

### Add to `.ai/guidelines/`
```yaml
# Truth Sources Hierarchy
1. Source code (highest authority)
2. docs/ (explicit documentation)
3. .ai/knowledge/ (learned patterns)
4. Agent inference (lowest — flag uncertainty)
```

### Audit Prompt Extension
Add to `prompts/code_review.md`:
```markdown
6. **Epistemic Check**: Are there fabricated paths, endpoints, or claims?
```

