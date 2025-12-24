# Design Guidelines & UX Standards

> [!TIP]
> Just because it works doesn't mean it's done. A product isn't "complete" until it feels "Premium".

## 1. The "Wow" Factor
*   **Premium Feel**:
    *   Use **Micro-interactions**: Buttons should react (hover, click, focus).
    *   **Transitions**: content shouldn't just "snap" into place. Use smooth fades and slides.
    *   **Glassmorphism/Modern UI**: Use subtle shadows, blurs, and consistent spacing variables.
*   **Colors**:
    *   Avoid "Default Blue/Red". Use semantic color palettes defined in `index.css` / CSS Variables.
    *   Support **Dark Mode** first-class.

## 2. Usability & Accessibility (a11y)
*   **Semantics**: Use correct HTML tags (`<button>` not `<div onClick>`).
*   **Keyboard Nav**: Ensure all interactive elements are reachable via `Tab`.
*   **Contrast**: Text must be legible against the background.

## 3. Feedback
*   **Loading States**: Never leave the user guessing. Use Skeletons or Spinners for async items.
*   **Error Handling**:
    *   Bad: "Error 500".
    *   Good: "Something went wrong fetching the data. Please check your connection."
    *   **GenAI Native Tip**: Make error messages detailed enough that an *Agent* reading a screenshot or log can understand what happened (e.g., include Error Codes).

## 4. Code Structure for Design
*   **Design Tokens**: Define colors, spacing, and fonts in global CSS variables (`:root`). Do not hardcode hex values in components.
*   **Component Composition**: Build small, reusable UI primitives (Button, Card, Input) before building complex Pages.

## 5. Agent-Friendly UI
*   **Test IDs**: Add `data-testid` to critical interaction elements. This allows Agents (and E2E tests) to reliably locate elements even if the visual design changes completely.
