# Coding Standards & Guidelines

## General Principles (Clean Code)
1. **Readable is better than clever**: Write code for humans first, compilers second.
2. **SRP (Single Responsibility Principle)**: Functions and classes should do one thing well.
3. **DRY (Don't Repeat Yourself)**: Extract common logic, but avoid premature optimization.
4. **Early Returns**: Reduce nesting by using guard clauses.

## Naming Conventions
- **Variables**: Descriptive and specific (e.g., `userList` instead of `list`).
- **Functions**: Verb-noun pairs (e.g., `calculateTotal`, `fetchData`).
- **Constants**: UPPER_CASE_WITH_UNDERSCORES.

## Language Specifics
### TypeScript / JavaScript
- Use `const` over `let`. Avoid `var`.
- Use `async/await` over raw Promises.
- Strict typing: Avoid `any` whenever possible.

### Go
- Use `gofmt` style.
- Error handling: Handle errors explicitly (check `if err != nil`).
- Short variable names are okay for short scopes (e.g., `i`, `err`).
