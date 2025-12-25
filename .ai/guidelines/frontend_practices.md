# Frontend Architecture & Best Practices

## 1. Internationalization (I18n)
*   **Library**: Use `react-i18next`.
*   **Structure**: Store locales in `public/locales/{lang}/common.json`.
*   **Usage**:
    *   Always use `useTranslation` hook: `const { t } = useTranslation('common');`.
    *   **Crucial**: Destructure `t` before returning JSX to avoid "undefined" errors.
    *   Keys should be nested for organization (e.g., `developer.addTitle`).

## 2. State Management (Context API)
*   **Pattern**: Use React Context for global state (e.g., `GitAPIContext`).
*   **Updates**:
    *   When adding new actions, update the `ContextType` interface first.
    *   Implement functions using `useCallback` to maintain referential equality.
    *   **Exposure**: Add the new function to the `contextValue` object in `useMemo`.
    *   **Dependencies**: Ensure all used state/functions are in the dependency arrays of both `useCallback` (implementation) and `useMemo` (exposure).

## 3. Component Design
*   **Separation of Concerns**: Extract complex UI logic into sub-components.
*   **Event Handling**: Use `e.stopPropagation()` when nesting interactive elements (e.g., a delete button inside a clickable tab).

## 4. Refactoring Patterns
*   **Components vs Render Functions**:
    *   **Bad**: `const renderSection = () => { ... }` defined inside a parent component. This causes scope issues and makes it hard to manage local state (hooks).
    *   **Good**: Extract to a separate functional component: `const SectionView: React.FC<Props> = ({...}) => { ... }`.
    *   **Effect**: Allows cleaner `useTranslation`, `useState`, and `useEffect` usage within the sub-component.

