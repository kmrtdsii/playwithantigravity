# Contributing to GitGym

Thank you for your interest in contributing to GitGym! We welcome contributions from everyone.

## Getting Started

1.  Clone the repository:
    ```bash
    git clone https://github.com/kurobon/gitgym.git
    cd gitgym
    ```
2.  Install dependencies and verify environment:
    ```bash
    ./scripts/test-all.sh
    ```

## Development Workflow

We follow a strict "GenAI Native" development philosophy.

*   **Documentation First**: Before writing code, update or read `docs/`.
*   **Verification**: All changes must pass `./scripts/test-all.sh`.
*   **Coding Standards**: Please refer to [.ai/guidelines/coding_standards.md](.ai/guidelines/coding_standards.md) for detailed rules on Go and TypeScript style.

## Pull Request Process

1.  Create a feature branch (`git checkout -b feature/amazing-feature`).
2.  Commit your changes (`git commit -m 'feat: Add amazing feature'`).
    *   Please use [Conventional Commits](https://www.conventionalcommits.org/).
3.  Push to the branch (`git push origin feature/amazing-feature`).
4.  Open a Pull Request.

## License

By contributing, you agree that your contributions will be licensed under its MIT License.
