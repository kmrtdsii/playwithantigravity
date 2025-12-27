# GenAI Native Development Practices

> [!NOTE]
> This document aggregates best practices for "GenAI Native" software development. It serves as a guide for Agents to write code and documentation that is optimized for consumption and maintenance by other AI Agents.

## 1. Codebase Structure for AI
AI agents thrive on **context** and **predictability**.
-   **Modular Design**: Break down massive files. LLMs have context windows; smaller, self-contained modules (~250 lines) are easier to reason about and refactor safely.
-   **Explicit Context**:
    -   **Docstrings**: explaining *Why* a function exists is more valuable than *What* it does (AI can read the code).
    -   **Typing**: Use strict types (TypeScript interfaces, Go structs) to act as guardrails.
    -   **Naming**: Use descriptive, verbose names. `user_session_timeout_ms` is better than `timeout`.

## 2. Documentation Strategy (`AGENTS.md` style)
Documentation is the "Prompt" for the next agent.
-   **Architecture Records**: Keep a record of *decisions* (ADRs). If you choose a complex pattern, document it so the next AI doesn't "refactor" it away.
-   **Single Source of Truth**: `docs/` should be the absolute authority. Avoid scattering rules in comments.
-   **LLM-Friendly Formats**: Markdown is king. Use semantic headers, bullet points, and code blocks. Avoid screenshots for instructions.

## 3. Agentic Workflows
When acting as an Agent, follow these loops:
-   **Reflection**: Before finalizing a task, critique your own work. "Did I break existing tests? Does this match the user's intent?"
-   **Tool Use**: Prefer using provided tools (verified paths) over guessing.
-   **Environment Awareness**: Don't assume the world is ready. Check servers, ports, and file existence before acting. (e.g., "Is `localhost:80` responding?" before opening a browser).
-   **Iterative Planning**: Don't try to solve the whole world in one prompt. Break it down into artifacts (`implementation_plan.md`).

## 4. Anti-Patterns to Avoid
-   **Implicit Magic**: Code that relies on global state or hidden side effects is hallucination-prone.
-   **Ambiguity**: vague generic names like `Manager` or `Handler` confuse agents about boundaries.
-   **Staleness**: Outdated comments are worse than no comments. If you change code, you *must* update the docs.
