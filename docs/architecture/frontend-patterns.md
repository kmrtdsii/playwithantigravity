# Frontend Patterns & Architecture

> [!IMPORTANT]
> Follow these patterns to maintain a clean, performant React application.

## 1. Component Rules

### "Smart" vs "Dumb" Components
- **Smart (Container)**:
  - Connects to `GitAPIContext` or `useQuery`.
  - Handles business logic.
  - Example: `AppLayout.tsx`, `GitTerminal.tsx`.
- **Dumb (Presentational)**:
  - Receives data via props.
  - Zero dependencies on global state.
  - **Must** be pure helpers.
  - Example: `GraphNode.tsx`, `Button.tsx`.

### Custom Hooks
- Extract complex logic (state machines, resize listeners) into `src/hooks/`.
- **Naming**: `use<FeatureName>`.
- **Example**: `useResizablePanes` moves layout math out of `AppLayout`.

## 2. styling
- **CSS Variables**: Use `src/index.css` variables for colors/spacing. **Do not hardcode hex values.**
  - Good: `var(--bg-primary)`
  - Bad: `#1a1a1a`
- **Scoped CSS**: Prefer CSS Modules or simple class names BEM-style if global.

## 3. Visualization Strategy
- Graphs (`GitGraphViz`) are **Derived State**.
- **Never** manually mutate the DOM of the graph.
- Always re-render based on `state.commits` from the backend.
