# Performance Standards & Optimization

> [!NOTE]
> Performance is a key feature of "Premium" software. Slow apps feel cheap. As an Antigravity Agent, you aim for code that is efficient by design.

## 1. Frontend Performance (React/Web)
*   **Bundle Size**:
    *   Avoid importing massive libraries for simple functions (e.g., don't import `lodash` if `Array.map` suffices).
    *   Use **Lazy Loading** (`React.lazy`, `Suspense`) for heavy routes or components.
*   **Render Cycles**:
    *   Use `React.memo`, `useMemo`, and `useCallback` to prevent unnecessary re-renders in complex component trees.
    *   Keep state as local as possible. Global contexts should be split if they cause wide re-renders.
*   **Assets**:
    *   Serve images in modern formats (WebP, AVIF).
    *   Preload critical fonts and scripts.

## 2. Backend Performance (Go)
*   **Concurrency**:
    *   Utilize Goroutines for I/O bound tasks, but manage them with `errgroup` or `WorkGen` patterns to prevent leaks.
    *   Always use `context` for cancellation and timeouts.
*   **Database**:
    *   **N+1 Problem**: Watch out for loops that query the DB. Use `JOIN`s or batch fetching.
    *   **Indexing**: Ensure Foreign Keys and searchable columns are indexed.
*   **Memory**:
    *   Avoid reading entire files into memory (`ioutil.ReadFile` on large files). Use `io.Reader` streams.

## 3. Agentic Performance Cost (GenAI Native)
*   **Context Efficiency**:
    *   When an Agent reads files, large files cost "attention". Minimize file size where possible.
    *   Split monolithic files into smaller modules. This helps LLMs reason about the code better and reduces "Context Window" pollution.
*   **Log Verbosity**:
    *   Logs should be meaningful. Flooding stdout makes it hard for Agents to debug runtime issues via terminal tools.
